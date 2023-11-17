// Package fake provides a fake implementation of the cloud
// connector interface, to be used in tests.
package fake

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/go-uuid"

	"github.com/leaseweb/cloudstack-csi-driver/pkg/cloud"
	"github.com/leaseweb/cloudstack-csi-driver/pkg/util"
)

const zoneID = "a1887604-237c-4212-a9cd-94620b7880fa"
const snapshotReadyStatus = "BackedUp"

type fakeConnector struct {
	node            *cloud.VM
	volumesByID     map[string]cloud.Volume
	volumesByName   map[string]cloud.Volume
	snapshotsByID   map[string]cloud.Snapshot
	snapshotsByName map[string]cloud.Snapshot
}

// New returns a new fake implementation of the
// CloudStack connector.
func New() cloud.Interface {
	volume := cloud.Volume{
		ID:               "ace9f28b-3081-40c1-8353-4cc3e3014072",
		Name:             "vol-1",
		Size:             10,
		DiskOfferingID:   "9743fd77-0f5d-4ef9-b2f8-f194235c769c",
		ZoneID:           zoneID,
		VirtualMachineID: "",
		DeviceID:         "",
	}
	snapshot := cloud.Snapshot{
		ID:          "fc8621ac-168f-4f51-b3cf-c11681609c12",
		Created:     "2006-01-02T15:04:05-0700",
		Name:        "snapshot-1",
		VolumeID:    "CSIVolumeID",
		ZoneID:      zoneID,
		VirtualSize: 10,
		State:       "Backedup",
	}
	node := &cloud.VM{
		ID:     "0d7107a3-94d2-44e7-89b8-8930881309a5",
		ZoneID: zoneID,
	}
	return &fakeConnector{
		node:            node,
		volumesByID:     map[string]cloud.Volume{volume.ID: volume},
		volumesByName:   map[string]cloud.Volume{volume.Name: volume},
		snapshotsByID:   map[string]cloud.Snapshot{snapshot.ID: snapshot},
		snapshotsByName: map[string]cloud.Snapshot{snapshot.Name: snapshot},
	}
}

func (f *fakeConnector) GetVMByID(ctx context.Context, vmID string) (*cloud.VM, error) {
	if vmID == f.node.ID {
		return f.node, nil
	}
	return nil, cloud.ErrNotFound
}

func (f *fakeConnector) GetNodeInfo(ctx context.Context, vmName string) (*cloud.VM, error) {
	return f.node, nil
}

func (f *fakeConnector) ListZonesID(ctx context.Context) ([]string, error) {
	return []string{zoneID}, nil
}

func (f *fakeConnector) GetVolumeByID(ctx context.Context, volumeID string) (*cloud.Volume, error) {
	vol, ok := f.volumesByID[volumeID]
	if ok {
		return &vol, nil
	}
	return nil, cloud.ErrNotFound
}

func (f *fakeConnector) GetVolumeByName(ctx context.Context, name string) (*cloud.Volume, error) {
	vol, ok := f.volumesByName[name]
	if ok {
		return &vol, nil
	}
	return nil, cloud.ErrNotFound
}

func (f *fakeConnector) CreateVolume(ctx context.Context, diskOfferingID, zoneID, name string, sizeInGB int64, snapshotID string) (string, error) {
	id, _ := uuid.GenerateUUID()
	vol := cloud.Volume{}
	if snapshotID != "" {
		vol = cloud.Volume{
			ID:             id,
			Name:           name,
			Size:           util.GigaBytesToBytes(sizeInGB),
			DiskOfferingID: diskOfferingID,
			ZoneID:         zoneID,
			SnapshotID:     snapshotID,
		}
	} else {
		vol = cloud.Volume{
			ID:             id,
			Name:           name,
			Size:           util.GigaBytesToBytes(sizeInGB),
			DiskOfferingID: diskOfferingID,
			ZoneID:         zoneID,
		}
	}
	f.volumesByID[vol.ID] = vol
	f.volumesByName[vol.Name] = vol
	return vol.ID, nil
}

func (f *fakeConnector) DeleteVolume(ctx context.Context, id string) error {
	if vol, ok := f.volumesByID[id]; ok {
		name := vol.Name
		delete(f.volumesByName, name)
	}
	delete(f.volumesByID, id)
	return nil
}

func (f *fakeConnector) AttachVolume(ctx context.Context, volumeID, vmID string) (string, error) {
	return "1", nil
}

func (f *fakeConnector) DetachVolume(ctx context.Context, volumeID string) error {
	return nil
}

func (f *fakeConnector) CreateSnapshot(ctx context.Context, name, volID string) (string, error) {
	id, _ := uuid.GenerateUUID()
	createdAt := time.Now().Format("2006-01-02T15:04:05-0700")

	snap := cloud.Snapshot{
		ID:       id,
		Name:     name,
		VolumeID: volID,
		Created:  createdAt,
		State:    "Backedup",
	}
	f.snapshotsByID[snap.ID] = snap
	f.snapshotsByName[snap.Name] = snap
	return snap.ID, nil
}

func (f *fakeConnector) DeleteSnapshot(ctx context.Context, snapID string) error {
	if snap, ok := f.snapshotsByID[snapID]; ok {
		name := snap.Name
		delete(f.snapshotsByName, name)
	}
	delete(f.snapshotsByID, snapID)
	return nil
}

func (f *fakeConnector) WaitSnapshotReady(ctx context.Context, snapshotName string) error {
	snap, ok := f.snapshotsByName[snapshotName]
	if ok && snap.State == snapshotReadyStatus {
		return nil
	}
	return nil
}

func (f *fakeConnector) GetSnapshotByID(ctx context.Context, snapshotID string) (*cloud.Snapshot, error) {
	snap, ok := f.snapshotsByID[snapshotID]
	if ok {
		return &snap, nil
	}
	return nil, cloud.ErrNotFound
}

func (f *fakeConnector) GetSnapshotByName(ctx context.Context, name string) (*cloud.Snapshot, error) {
	snap, ok := f.snapshotsByName[name]
	if ok {
		return &snap, nil
	}
	return nil, cloud.ErrNotFound
}

func (f *fakeConnector) ListSnapshots(ctx context.Context, filters map[string]string) ([]cloud.Snapshot, string, error) {
	var snaplist []cloud.Snapshot

	name := filters["Name"]
	volumeID := filters["VolumeID"]
	startingToken := filters["Marker"]
	limitfilter := filters["Limit"]
	limit, _ := strconv.Atoi(limitfilter)

	for _, value := range f.snapshotsByID {
		if volumeID != "" {
			if value.VolumeID == volumeID {
				snaplist = append(snaplist, value)
				break
			}
		} else if name != "" {
			if value.Name == name {
				snaplist = append(snaplist, value)
				break
			}
		} else {
			snaplist = append(snaplist, value)
		}
	}

	if startingToken != "" && len(snaplist) > limit {
		t, _ := strconv.Atoi(startingToken)
		snaplist = snaplist[t:]
	}

	var nextPageToken string

	if limit != 0 {
		snaplist = snaplist[:limit]
		nextPageToken = limitfilter
	}
	return snaplist, nextPageToken, nil
}
