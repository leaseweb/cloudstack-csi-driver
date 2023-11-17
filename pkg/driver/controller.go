package driver

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/leaseweb/cloudstack-csi-driver/pkg/cloud"
	"github.com/leaseweb/cloudstack-csi-driver/pkg/util"
)

// onlyVolumeCapAccessMode is the only volume capability access
// mode possible for CloudStack: SINGLE_NODE_WRITER, since a
// CloudStack volume can only be attached to a single node at
// any given time.
var onlyVolumeCapAccessMode = csi.VolumeCapability_AccessMode{
	Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
}

type controllerServer struct {
	csi.UnimplementedControllerServer
	connector   cloud.Interface
	volumeLocks *util.VolumeLocks
}

// NewControllerServer creates a new Controller gRPC server.
func NewControllerServer(connector cloud.Interface) csi.ControllerServer {
	return &controllerServer{
		connector:   connector,
		volumeLocks: util.NewVolumeLocks(),
	}
}

func (cs *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {

	// Check arguments

	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume name missing in request")
	}
	name := req.GetName()

	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities missing in request")
	}
	if !isValidVolumeCapabilities(volCaps) {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not supported. Only SINGLE_NODE_WRITER supported.")
	}

	if req.GetParameters() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume parameters missing in request")
	}
	diskOfferingID := req.GetParameters()[DiskOfferingKey]
	if diskOfferingID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Missing parameter %v", DiskOfferingKey)
	}

	if acquired := cs.volumeLocks.TryAcquire(name); !acquired {
		ctxzap.Extract(ctx).Sugar().Errorf(util.VolumeOperationAlreadyExistsFmt, name)
		return nil, status.Errorf(codes.Aborted, util.VolumeOperationAlreadyExistsFmt, name)
	}
	defer cs.volumeLocks.Release(name)

	// Check if a volume with that name already exists
	if vol, err := cs.connector.GetVolumeByName(ctx, name); err == cloud.ErrNotFound {
		// The volume does not exist
	} else if err != nil {
		// Error with CloudStack
		return nil, status.Errorf(codes.Internal, "CloudStack error: %v", err)
	} else {
		// The volume exists. Check if it suits the request.
		if ok, message := checkVolumeSuitable(vol, diskOfferingID, req.GetCapacityRange(), req.GetAccessibilityRequirements()); !ok {
			return nil, status.Errorf(codes.AlreadyExists, "Volume %v already exists but does not satisfy request: %s", name, message)
		}
		// Existing volume is ok
		return &csi.CreateVolumeResponse{
			Volume: &csi.Volume{
				VolumeId:      vol.ID,
				CapacityBytes: vol.Size,
				VolumeContext: req.GetParameters(),
				ContentSource: req.GetVolumeContentSource(),
				AccessibleTopology: []*csi.Topology{
					Topology{ZoneID: vol.ZoneID}.ToCSI(),
				},
			},
		}, nil
	}

	// We have to create the volume

	// Determine volume size using requested capacity range
	sizeInGB, err := determineSize(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Determine zone using topology constraints
	var zoneID string
	topologyRequirement := req.GetAccessibilityRequirements()
	if topologyRequirement == nil || topologyRequirement.GetRequisite() == nil {
		// No topology requirement. Use random zone
		zones, err := cs.connector.ListZonesID(ctx)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		n := len(zones)
		if n == 0 {
			return nil, status.Error(codes.Internal, "No zone available")
		}
		zoneID = zones[rand.Intn(n)]
	} else {
		reqTopology := topologyRequirement.GetRequisite()
		if len(reqTopology) > 1 {
			return nil, status.Error(codes.InvalidArgument, "Too many topology requirements")
		}
		t, err := NewTopology(reqTopology[0])
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "Cannot parse topology requirements")
		}
		zoneID = t.ZoneID
	}
	content := req.GetVolumeContentSource()
	var snapshotID string

	if content != nil && content.GetSnapshot() != nil {
		snapshotID = content.GetSnapshot().GetSnapshotId()
		_, err := cs.connector.GetSnapshotByID(ctx, snapshotID)
		if err != nil {
			if err == cloud.ErrNotFound {
				return nil, status.Errorf(codes.NotFound, "VolumeContentSource Snapshot %s not found", snapshotID)
			}
			return nil, status.Errorf(codes.Internal, "Failed to retrieve the snapshot %s: %v", snapshotID, err)
		} else {
			ctxzap.Extract(ctx).Sugar().Infow("Creating new volume from snapshot",
				"name", name,
				"size", sizeInGB,
				"offering", diskOfferingID,
				"zone", zoneID,
				"snapshotid", snapshotID,
			)
		}
	} else {
		ctxzap.Extract(ctx).Sugar().Infow("Creating new volume",
			"name", name,
			"size", sizeInGB,
			"offering", diskOfferingID,
			"zone", zoneID,
		)
	}

	volumeID, err := cs.connector.CreateVolume(ctx, diskOfferingID, zoneID, name, sizeInGB, snapshotID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Cannot create volume %s: %v", name, err.Error())
	}

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volumeID,
			CapacityBytes: util.GigaBytesToBytes(sizeInGB),
			VolumeContext: req.GetParameters(),
			ContentSource: req.GetVolumeContentSource(),
			AccessibleTopology: []*csi.Topology{
				Topology{ZoneID: zoneID}.ToCSI(),
			},
		},
	}, nil
}

func checkVolumeSuitable(vol *cloud.Volume,
	diskOfferingID string, capRange *csi.CapacityRange, topologyRequirement *csi.TopologyRequirement) (bool, string) {

	if vol.DiskOfferingID != diskOfferingID {
		return false, fmt.Sprintf("Disk offering %s; requested disk offering %s", vol.DiskOfferingID, diskOfferingID)
	}

	if capRange != nil {
		if capRange.GetLimitBytes() > 0 && vol.Size > capRange.GetLimitBytes() {
			return false, fmt.Sprintf("Disk size %v bytes > requested limit size %v bytes", vol.Size, capRange.GetLimitBytes())
		}
		if capRange.GetRequiredBytes() > 0 && vol.Size < capRange.GetRequiredBytes() {
			return false, fmt.Sprintf("Disk size %v bytes < requested required size %v bytes", vol.Size, capRange.GetRequiredBytes())
		}
	}

	if topologyRequirement != nil && topologyRequirement.GetRequisite() != nil {
		reqTopology := topologyRequirement.GetRequisite()
		if len(reqTopology) > 1 {
			return false, "Too many topology requirements"
		}
		t, err := NewTopology(reqTopology[0])
		if err != nil {
			return false, "Cannot parse topology requirements"
		}
		if t.ZoneID != vol.ZoneID {
			return false, fmt.Sprintf("Volume in zone %s, requested zone is %s", vol.ZoneID, t.ZoneID)
		}
	}

	return true, ""
}

func determineSize(req *csi.CreateVolumeRequest) (int64, error) {
	var sizeInGB int64

	if req.GetCapacityRange() != nil {
		capRange := req.GetCapacityRange()

		required := capRange.GetRequiredBytes()
		sizeInGB = util.RoundUpBytesToGB(required)
		if sizeInGB == 0 {
			sizeInGB = 1
		}

		if limit := capRange.GetLimitBytes(); limit > 0 {
			if util.GigaBytesToBytes(sizeInGB) > limit {
				return 0, fmt.Errorf("after round-up, volume size %v GB exceeds the limit specified of %v bytes", sizeInGB, limit)
			}
		}
	}

	if sizeInGB == 0 {
		sizeInGB = 1
	}
	return sizeInGB, nil
}

func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	volumeID := req.GetVolumeId()

	if acquired := cs.volumeLocks.TryAcquire(volumeID); !acquired {
		ctxzap.Extract(ctx).Sugar().Errorf(util.VolumeOperationAlreadyExistsFmt, volumeID)
		return nil, status.Errorf(codes.Aborted, util.VolumeOperationAlreadyExistsFmt, volumeID)
	}
	defer cs.volumeLocks.Release(volumeID)

	ctxzap.Extract(ctx).Sugar().Infow("Deleting volume",
		"volumeID", volumeID,
	)

	err := cs.connector.DeleteVolume(ctx, volumeID)
	if err != nil && err != cloud.ErrNotFound {
		return nil, status.Errorf(codes.Internal, "Cannot delete volume %s: %s", volumeID, err.Error())
	}
	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *controllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	// Check arguments

	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	volumeID := req.GetVolumeId()

	if req.GetNodeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Node ID missing in request")
	}
	nodeID := req.GetNodeId()

	if req.GetReadonly() {
		return nil, status.Error(codes.InvalidArgument, "Readonly not possible")
	}

	if req.GetVolumeCapability() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability missing in request")
	}
	if req.GetVolumeCapability().AccessMode.Mode != onlyVolumeCapAccessMode.GetMode() {
		return nil, status.Error(codes.InvalidArgument, "Access mode not accepted")
	}

	ctxzap.Extract(ctx).Sugar().Infow("Initiating attaching volume",
		"volumeID", volumeID,
		"nodeID", nodeID,
	)

	// Check volume
	vol, err := cs.connector.GetVolumeByID(ctx, volumeID)
	if err == cloud.ErrNotFound {
		return nil, status.Errorf(codes.NotFound, "Volume %v not found", volumeID)
	} else if err != nil {
		// Error with CloudStack
		return nil, status.Errorf(codes.Internal, "Error %v", err)
	}

	if vol.VirtualMachineID != "" && vol.VirtualMachineID != nodeID {
		ctxzap.Extract(ctx).Sugar().Errorw("Volume already attached to another node",
			"volumeID", volumeID,
			"nodeID", nodeID,
			"attached nodeID", vol.VirtualMachineID,
		)
		return nil, status.Error(codes.AlreadyExists, "Volume already assigned to another node")
	}

	if _, err := cs.connector.GetVMByID(ctx, nodeID); err == cloud.ErrNotFound {
		return nil, status.Errorf(codes.NotFound, "VM %v not found", nodeID)
	} else if err != nil {
		// Error with CloudStack
		return nil, status.Errorf(codes.Internal, "Error %v", err)
	}

	if vol.VirtualMachineID == nodeID {
		// volume already attached
		ctxzap.Extract(ctx).Sugar().Infow("Volume already attached to node",
			"volumeID", volumeID,
			"nodeID", nodeID,
			"deviceID", vol.DeviceID,
		)
		publishContext := map[string]string{
			deviceIDContextKey: vol.DeviceID,
		}
		return &csi.ControllerPublishVolumeResponse{PublishContext: publishContext}, nil
	}

	ctxzap.Extract(ctx).Sugar().Infow("Attaching volume to node",
		"volumeID", volumeID,
		"nodeID", nodeID,
	)

	deviceID, err := cs.connector.AttachVolume(ctx, volumeID, nodeID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Cannot attach volume %s: %s", volumeID, err.Error())
	}

	ctxzap.Extract(ctx).Sugar().Infow("Attached volume to node successfully",
		"volumeID", volumeID,
		"nodeID", nodeID,
	)

	publishContext := map[string]string{
		deviceIDContextKey: deviceID,
	}
	return &csi.ControllerPublishVolumeResponse{PublishContext: publishContext}, nil
}

func (cs *controllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	// Check arguments

	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	volumeID := req.GetVolumeId()
	nodeID := req.GetNodeId()

	// Check volume
	if vol, err := cs.connector.GetVolumeByID(ctx, volumeID); err == cloud.ErrNotFound {
		// Volume does not exist in CloudStack. We can safely assume this volume is no longer attached
		// The spec requires us to return OK here
		return &csi.ControllerUnpublishVolumeResponse{}, nil
	} else if err != nil {
		// Error with CloudStack
		return nil, status.Errorf(codes.Internal, "Error %v", err)
	} else if nodeID != "" && vol.VirtualMachineID != nodeID {
		// Volume is present but not attached to this particular nodeID
		return &csi.ControllerUnpublishVolumeResponse{}, nil
	}

	// Check VM existence
	if _, err := cs.connector.GetVMByID(ctx, nodeID); err == cloud.ErrNotFound {
		// volumes cannot be attached to deleted VMs
		ctxzap.Extract(ctx).Sugar().Warnw("VM not found, marking ControllerUnpublishVolume successful",
			"volumeID", volumeID,
			"nodeID", nodeID,
		)
		return &csi.ControllerUnpublishVolumeResponse{}, nil
	} else if err != nil {
		// Error with CloudStack
		return nil, status.Errorf(codes.Internal, "Error %v", err)
	}

	ctxzap.Extract(ctx).Sugar().Infow("Detaching volume from node",
		"volumeID", volumeID,
		"nodeID", nodeID,
	)

	err := cs.connector.DetachVolume(ctx, volumeID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Cannot detach volume %s: %s", volumeID, err.Error())
	}

	ctxzap.Extract(ctx).Sugar().Infow("Detached volume from node successfully",
		"volumeID", volumeID,
		"nodeID", nodeID,
	)

	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

func (cs *controllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not provided")
	}

	if _, err := cs.connector.GetVolumeByID(ctx, volumeID); err == cloud.ErrNotFound {
		return nil, status.Errorf(codes.NotFound, "Volume %v not found", volumeID)
	} else if err != nil {
		// Error with CloudStack
		return nil, status.Errorf(codes.Internal, "Error %v", err)
	}

	if !isValidVolumeCapabilities(volCaps) {
		return &csi.ValidateVolumeCapabilitiesResponse{Message: "Requested VolumeCapabilities are invalid"}, nil
	}

	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeContext:      req.GetVolumeContext(),
			VolumeCapabilities: volCaps,
			Parameters:         req.GetParameters(),
		}}, nil
}

func isValidVolumeCapabilities(volCaps []*csi.VolumeCapability) bool {
	for _, c := range volCaps {
		if c.GetAccessMode() != nil && c.GetAccessMode().GetMode() != onlyVolumeCapAccessMode.GetMode() {
			return false
		}
	}
	return true
}

func (cs *controllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: []*csi.ControllerServiceCapability{
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
					},
				},
			},
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
					},
				},
			},
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
					},
				},
			},
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS,
					},
				},
			},
		},
	}, nil
}

func (cs *controllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	ctxzap.Extract(ctx).Sugar().Infow("CreateSnapshot: called with args", "args", protosanitizer.StripSecrets(*req))
	name := req.Name
	volumeID := req.GetSourceVolumeId()
	snapshotCreateLock := util.NewOperationLock(ctx)

	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "Snapshot name must be provided in CreateSnapshot request")
	}

	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "VolumeID must be provided in CreateSnapshot request")
	}

	err := snapshotCreateLock.GetSnapshotCreateLock(volumeID)
	if err != nil {

		ctxzap.Extract(ctx).Sugar().Errorf(util.VolumeOperationAlreadyExistsFmt, volumeID)
		return nil, status.Errorf(codes.Aborted, util.VolumeOperationAlreadyExistsFmt, volumeID)
	}
	defer snapshotCreateLock.ReleaseSnapshotCreateLock(volumeID)

	// Check if a snapshot with that name already exists
	if snap, err := cs.connector.GetSnapshotByName(ctx, name); err == cloud.ErrNotFound {
		// The snapshot does not exist
	} else if err != nil {
		// Error with CloudStack
		return nil, status.Errorf(codes.Internal, "CloudStack error: %v", err)
	} else {
		// The snapshot exists. Check if it suits the request.
		if snap.VolumeID != volumeID {
			// Snapshot exists with a different source volume ID
			return nil, status.Errorf(codes.AlreadyExists, "Snapshot with given name already exists, with different source volume ID")
		}
		s, err := toCSISnapshot(snap)
		if err != nil {
			return nil, status.Errorf(codes.Internal,
				"couldn't convert CloudStack snapshot to CSI snapshot: %s", err.Error())
		}
		// Existing snapshot is ok
		return &csi.CreateSnapshotResponse{
			Snapshot: s,
		}, nil
	}

	// Verify a snapshot with the provided name doesn't already exist for this domain
	var snap *cloud.Snapshot
	filters := map[string]string{}
	filters["Name"] = name
	snapshots, _, err := cs.connector.ListSnapshots(ctx, filters)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to get snapshots")
	}

	if len(snapshots) == 1 {
		snap = &snapshots[0]

		if snap.VolumeID != volumeID {
			return nil, status.Error(codes.AlreadyExists, "Snapshot with given name already exists, with different source volume ID")
		}
	} else if len(snapshots) > 1 {
		return nil, status.Errorf(codes.Internal, "Multiple snapshots reported by CSI with same name(%s)", name)

	} else {
		snapID, err := cs.connector.CreateSnapshot(ctx, name, volumeID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Cannot create volume %s: %v", name, err.Error())
		}
		snap, err = cs.connector.GetSnapshotByID(ctx, snapID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Cannot find snapshot %s: %v", snap.Name, err.Error())
		}

		fmt.Printf("CreateSnapshot %s with ID %s from volume with ID: %s", name, snapID, volumeID)
	}

	s, err := toCSISnapshot(snap)
	if err != nil {
		return nil, status.Errorf(codes.Internal,
			"couldn't convert CloudStack snapshot to CSI snapshot: %s", err.Error())
	}
	err = cs.connector.WaitSnapshotReady(ctx, snap.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("CreateSnapshot failed with error %v", err))
	}

	return &csi.CreateSnapshotResponse{
		Snapshot: s,
	}, nil
}

func (cs *controllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	ctxzap.Extract(ctx).Sugar().Infow("DeleteSnapshot called with args", "args", protosanitizer.StripSecrets(*req))

	snapshotDeleteLock := util.NewOperationLock(ctx)
	snapshotID := req.GetSnapshotId()
	if snapshotID == "" {
		return nil, status.Error(codes.InvalidArgument, "DeleteSnapshot Snapshot ID must be provided")
	}

	errLock := snapshotDeleteLock.GetSnapshotDeleteLock(snapshotID)
	if errLock != nil {
		ctxzap.Extract(ctx).Sugar().Errorf(util.SnapshotOperationAlreadyExistsFmt, snapshotID)
		return nil, status.Errorf(codes.Aborted, util.SnapshotOperationAlreadyExistsFmt, snapshotID)
	}
	defer snapshotDeleteLock.ReleaseSnapshotDeleteLock(snapshotID)

	err := cs.connector.DeleteSnapshot(ctx, snapshotID)
	if err != nil && err != cloud.ErrNotFound {
		return nil, status.Errorf(codes.Internal, "Cannot delete volume %s: %s", snapshotID, err.Error())
	}
	return &csi.DeleteSnapshotResponse{}, nil
}

func (cs *controllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	snapshotID := req.GetSnapshotId()
	if len(snapshotID) != 0 {
		snap, err := cs.connector.GetSnapshotByID(ctx, snapshotID)
		if err != nil {
			if err == cloud.ErrNotFound {
				fmt.Printf("Snapshot %s not found", snapshotID)
				return &csi.ListSnapshotsResponse{}, nil
			}
			return nil, status.Errorf(codes.Internal, "Failed to GetSnapshot %s: %v", snapshotID, err)
		}

		s, err := toCSISnapshot(snap)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Couldn't convert CloudStack snapshot to CSI snapshot: %s", err.Error())
		}
		entry := &csi.ListSnapshotsResponse_Entry{
			Snapshot: s,
		}

		entries := []*csi.ListSnapshotsResponse_Entry{entry}
		return &csi.ListSnapshotsResponse{
			Entries: entries,
		}, nil

	}

	filters := map[string]string{}
	if volID := req.GetSourceVolumeId(); len(volID) != 0 {
		filters["VolumeID"] = volID
	}

	var maxEntries int32
	if req.MaxEntries > 0 {
		maxEntries = req.MaxEntries
	}

	//  Marker is the last-seen item and Limit is a page size
	filters["Limit"] = strconv.Itoa(int(maxEntries))
	filters["Marker"] = req.StartingToken

	// Fetch all snapshots from CloudStack
	allSnapshotsList, nextPageToken, err := cs.connector.ListSnapshots(ctx, filters)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error listing snapshots: %v", err)
	}
	var sentries []*csi.ListSnapshotsResponse_Entry
	for _, v := range allSnapshotsList {
		s, err := toCSISnapshot(&v)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "couldn't convert CloudStack snapshot to CSI snapshot: %s", err.Error())
		}
		sentry := &csi.ListSnapshotsResponse_Entry{
			Snapshot: s,
		}
		sentries = append(sentries, sentry)
	}
	return &csi.ListSnapshotsResponse{
		Entries:   sentries,
		NextToken: nextPageToken,
	}, nil
}

// toCSISnapshot converts a CloudStack Snapshot struct into a csi.Snapshot struct
func toCSISnapshot(snap *cloud.Snapshot) (*csi.Snapshot, error) {
	createdAt, err := time.Parse("2006-01-02T15:04:05-0700", snap.Created)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse snapshot's created field: %s", err.Error())
	}

	tstamp := timestamppb.New(createdAt)
	return &csi.Snapshot{
		SizeBytes:       int64(snap.VirtualSize),
		SnapshotId:      snap.ID,
		SourceVolumeId:  snap.VolumeID,
		CreationTime:    tstamp,
		ReadyToUse:      true,
		GroupSnapshotId: "",
	}, nil
}
