// Package cloud contains CloudStack related
// functions.
package cloud

import (
	"context"
	"errors"

	"github.com/apache/cloudstack-go/v2/cloudstack"
)

//go:generate ../../hack/tools/bin/mockgen -destination=./mock_cloud.go -package=cloud -source ./cloud.go

// Cloud is the CloudStack client interface.
type Cloud interface {
	GetNodeInfo(ctx context.Context, vmName string) (*VM, error)
	GetVMByID(ctx context.Context, vmID string) (*VM, error)

	ListZonesID(ctx context.Context) ([]string, error)

	GetVolumeByID(ctx context.Context, volumeID string) (*Volume, error)
	GetVolumeByName(ctx context.Context, name string) (*Volume, error)
	CreateVolume(ctx context.Context, diskOfferingID, zoneID, name string, sizeInGB int64) (string, error)
	DeleteVolume(ctx context.Context, id string) error
	AttachVolume(ctx context.Context, volumeID, vmID string) (string, error)
	DetachVolume(ctx context.Context, volumeID string) error
	ExpandVolume(ctx context.Context, cs *ControllerService, volumeID string, newSizeInGB int64) error
}

// Volume represents a CloudStack volume.
type Volume struct {
	ID   string
	Name string

	// Size in Bytes
	Size int64

	DiskOfferingID string
	ZoneID         string

	VirtualMachineID string
	DeviceID         string
}

// VM represents a CloudStack Virtual Machine.
type VM struct {
	ID     string
	ZoneID string
}

// Specific errors.
var (
	ErrNotFound       = errors.New("not found")
	ErrTooManyResults = errors.New("too many results")
)

// client is the implementation of Cloud.
type client struct {
	*cloudstack.CloudStackClient
	projectID string
}

// New creates a new cloud connector, given its configuration.
func New(config *Config) Cloud {
	csClient := &client{
		projectID: config.ProjectID,
	}
	csClient.CloudStackClient = cloudstack.NewAsyncClient(config.APIURL, config.APIKey, config.SecretKey, config.VerifySSL)

	return csClient
}
