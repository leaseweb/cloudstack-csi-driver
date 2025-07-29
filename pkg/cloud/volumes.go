package cloud

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"k8s.io/klog/v2"
)

func (c *client) listVolumes(p *cloudstack.ListVolumesParams) (*Volume, error) {
	l, err := c.Volume.ListVolumes(p)
	if err != nil {
		return nil, err
	}
	if l.Count == 0 {
		return nil, ErrNotFound
	}
	if l.Count > 1 {
		return nil, ErrTooManyResults
	}
	vol := l.Volumes[0]
	v := Volume{
		ID:               vol.Id,
		Name:             vol.Name,
		Size:             vol.Size,
		DiskOfferingID:   vol.Diskofferingid,
		ZoneID:           vol.Zoneid,
		ZoneName:         vol.Zonename,
		VirtualMachineID: vol.Virtualmachineid,
		DeviceID:         strconv.FormatInt(vol.Deviceid, 10),
	}

	return &v, nil
}

func (c *client) GetVolumeByID(ctx context.Context, volumeID string) (*Volume, error) {
	logger := klog.FromContext(ctx)
	p := c.Volume.NewListVolumesParams()
	p.SetId(volumeID)
	if c.projectID != "" {
		p.SetProjectid(c.projectID)
	}
	logger.V(2).Info("CloudStack API call", "command", "ListVolumes", "params", map[string]string{
		"id":        volumeID,
		"projectid": c.projectID,
	})

	return c.listVolumes(p)
}

func (c *client) GetVolumeByName(ctx context.Context, name string) (*Volume, error) {
	logger := klog.FromContext(ctx)
	p := c.Volume.NewListVolumesParams()
	p.SetName(name)
	if c.projectID != "" {
		p.SetProjectid(c.projectID)
	}
	logger.V(2).Info("CloudStack API call", "command", "ListVolumes", "params", map[string]string{
		"name": name,
	})

	return c.listVolumes(p)
}

func (c *client) CreateVolume(ctx context.Context, diskOfferingID, zoneID, name string, sizeInGB int64) (*Volume, error) {
	logger := klog.FromContext(ctx)

	if zoneID == "" {
		// No topology requirement. Use random zone.
		zones, err := c.ListZonesID(ctx)
		if err != nil {
			return nil, err
		}
		n := len(zones)
		if n == 0 {
			return nil, errors.New("no zone available")
		}
		zoneID = zones[rand.Intn(n)] //nolint:gosec
	}

	p := c.Volume.NewCreateVolumeParams()
	p.SetDiskofferingid(diskOfferingID)
	p.SetZoneid(zoneID)
	p.SetName(name)
	p.SetSize(sizeInGB)
	if c.projectID != "" {
		p.SetProjectid(c.projectID)
	}
	logger.V(2).Info("CloudStack API call", "command", "CreateVolume", "params", map[string]string{
		"diskofferingid": diskOfferingID,
		"zoneid":         zoneID,
		"name":           name,
		"size":           strconv.FormatInt(sizeInGB, 10),
	})
	vol, err := c.Volume.CreateVolume(p)
	if err != nil {
		return nil, err
	}

	return &Volume{
		ID:             vol.Id,
		Name:           vol.Name,
		Size:           vol.Size,
		DiskOfferingID: vol.Diskofferingid,
		ZoneID:         vol.Zoneid,
		ZoneName:       vol.Zonename,
	}, nil
}

func (c *client) DeleteVolume(ctx context.Context, id string) error {
	logger := klog.FromContext(ctx)
	p := c.Volume.NewDeleteVolumeParams(id)
	logger.V(2).Info("CloudStack API call", "command", "DeleteVolume", "params", map[string]string{
		"id": id,
	})
	_, err := c.Volume.DeleteVolume(p)
	if err != nil && strings.Contains(err.Error(), "4350") {
		// CloudStack error InvalidParameterValueException
		return ErrNotFound
	}

	return err
}

func (c *client) AttachVolume(ctx context.Context, volumeID, vmID string) (string, error) {
	logger := klog.FromContext(ctx)
	p := c.Volume.NewAttachVolumeParams(volumeID, vmID)
	logger.V(2).Info("CloudStack API call", "command", "AttachVolume", "params", map[string]string{
		"id":               volumeID,
		"virtualmachineid": vmID,
	})
	r, err := c.Volume.AttachVolume(p)
	if err != nil {
		return "", err
	}

	return strconv.FormatInt(r.Deviceid, 10), nil
}

func (c *client) DetachVolume(ctx context.Context, volumeID string) error {
	logger := klog.FromContext(ctx)
	p := c.Volume.NewDetachVolumeParams()
	p.SetId(volumeID)
	logger.V(2).Info("CloudStack API call", "command", "DetachVolume", "params", map[string]string{
		"id": volumeID,
	})
	_, err := c.Volume.DetachVolume(p)

	return err
}

// ExpandVolume expands the volume to new size.
func (c *client) ExpandVolume(ctx context.Context, volumeID string, newSizeInGB int64) error {
	logger := klog.FromContext(ctx)

	p := c.Volume.NewResizeVolumeParams(volumeID)
	p.SetId(volumeID)
	p.SetSize(newSizeInGB)
	logger.V(2).Info("CloudStack API call", "command", "ExpandVolume", "params", map[string]string{
		"id":   volumeID,
		"size": strconv.FormatInt(newSizeInGB, 10),
	})
	// Execute the API call to resize the volume.
	_, err := c.Volume.ResizeVolume(p)
	if err != nil {
		// Handle the error accordingly
		return fmt.Errorf("failed to expand volume '%s': %w", volumeID, err)
	}

	return nil
}
