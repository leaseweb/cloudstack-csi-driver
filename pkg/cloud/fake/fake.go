// Package fake provides a fake implementation of the cloud
// connector interface, to be used in tests.
package fake

import (
	"context"
	"errors"

	"github.com/hashicorp/go-uuid"

	"github.com/leaseweb/cloudstack-csi-driver/pkg/cloud"
	"github.com/leaseweb/cloudstack-csi-driver/pkg/util"
)

const zoneID = "a1887604-237c-4212-a9cd-94620b7880fa"

type fakeConnector struct {
	node          *cloud.VM
	volumesByID   map[string]cloud.Volume
	volumesByName map[string]cloud.Volume
}

// New returns a new fake implementation of the
// CloudStack connector.
func New() cloud.Cloud {
	volume := cloud.Volume{
		ID:               "ace9f28b-3081-40c1-8353-4cc3e3014072",
		Name:             "vol-1",
		Size:             10,
		DiskOfferingID:   "9743fd77-0f5d-4ef9-b2f8-f194235c769c",
		ZoneID:           zoneID,
		VirtualMachineID: "",
		DeviceID:         "",
	}
	node := &cloud.VM{
		ID:     "0d7107a3-94d2-44e7-89b8-8930881309a5",
		ZoneID: zoneID,
	}

	return &fakeConnector{
		node:          node,
		volumesByID:   map[string]cloud.Volume{volume.ID: volume},
		volumesByName: map[string]cloud.Volume{volume.Name: volume},
	}
}

func (f *fakeConnector) GetVMByID(_ context.Context, vmID string) (*cloud.VM, error) {
	if vmID == f.node.ID {
		return f.node, nil
	}

	return nil, cloud.ErrNotFound
}

func (f *fakeConnector) GetNodeInfo(_ context.Context, _ string) (*cloud.VM, error) {
	return f.node, nil
}

func (f *fakeConnector) ListZonesID(_ context.Context) ([]string, error) {
	return []string{zoneID}, nil
}

func (f *fakeConnector) GetVolumeByID(_ context.Context, volumeID string) (*cloud.Volume, error) {
	vol, ok := f.volumesByID[volumeID]
	if ok {
		return &vol, nil
	}

	return nil, cloud.ErrNotFound
}

func (f *fakeConnector) GetVolumeByName(_ context.Context, name string) (*cloud.Volume, error) {
	vol, ok := f.volumesByName[name]
	if ok {
		return &vol, nil
	}

	return nil, cloud.ErrNotFound
}

func (f *fakeConnector) CreateVolume(_ context.Context, diskOfferingID, zoneID, name string, sizeInGB int64) (string, error) {
	id, _ := uuid.GenerateUUID()
	vol := cloud.Volume{
		ID:             id,
		Name:           name,
		Size:           util.GigaBytesToBytes(sizeInGB),
		DiskOfferingID: diskOfferingID,
		ZoneID:         zoneID,
	}
	f.volumesByID[vol.ID] = vol
	f.volumesByName[vol.Name] = vol

	return vol.ID, nil
}

func (f *fakeConnector) DeleteVolume(_ context.Context, id string) error {
	if vol, ok := f.volumesByID[id]; ok {
		name := vol.Name
		delete(f.volumesByName, name)
	}
	delete(f.volumesByID, id)

	return nil
}

func (f *fakeConnector) AttachVolume(_ context.Context, volumeID, nodeID string) (string, error) {
	if f.getVolumesPerServer(nodeID) >= 16 {
		return "", errors.New("The specified VM already has the maximum number of data disks (16) attached. Please specify another VM.") //nolint:revive,stylecheck
	}

	if vol, ok := f.volumesByID[volumeID]; ok {
		vol.VirtualMachineID = nodeID
		f.volumesByID[volumeID] = vol
		f.volumesByName[vol.Name] = vol

		return "1", nil
	}

	return "", cloud.ErrNotFound
}

func (f *fakeConnector) DetachVolume(_ context.Context, volumeID string) error {
	if vol, ok := f.volumesByID[volumeID]; ok {
		vol.VirtualMachineID = ""
		f.volumesByID[volumeID] = vol
		f.volumesByName[vol.Name] = vol

		return nil
	}

	return cloud.ErrNotFound
}

func (f *fakeConnector) ExpandVolume(_ context.Context, cs *ControllerService, volumeID string, newSizeInGB int64) error {
	if vol, ok := f.volumesByID[volumeID]; ok {
		newSizeInBytes := newSizeInGB * 1024 * 1024 * 1024
		if newSizeInBytes > vol.Size {
			vol.Size = newSizeInBytes
			f.volumesByID[volumeID] = vol
			f.volumesByName[vol.Name] = vol
		}

		return nil
	}

	return cloud.ErrNotFound
}

func (f *fakeConnector) getVolumesPerServer(nodeID string) int {
	volumesCount := 0
	for _, v := range f.volumesByID {
		if v.VirtualMachineID == nodeID {
			volumesCount++
		}
	}

	return volumesCount
}
