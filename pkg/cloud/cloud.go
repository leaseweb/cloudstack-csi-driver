// Package cloud contains CloudStack related
// functions.
package cloud

import (
	"context"
	"errors"

	"github.com/apache/cloudstack-go/v2/cloudstack"
)

// Interface is the CloudStack client interface.
type Interface interface {
	GetNodeInfo(ctx context.Context, vmName string) (*VM, error)
	GetVMByID(ctx context.Context, vmID string) (*VM, error)

	ListZonesID(ctx context.Context) ([]string, error)

	GetVolumeByID(ctx context.Context, volumeID string) (*Volume, error)
	GetVolumeByName(ctx context.Context, name string) (*Volume, error)
	CreateVolume(ctx context.Context, diskOfferingID, zoneID, name string, sizeInGB int64, snapshotID string) (string, error)
	DeleteVolume(ctx context.Context, id string) error
	AttachVolume(ctx context.Context, volumeID, vmID string) (string, error)
	DetachVolume(ctx context.Context, volumeID string) error

	CreateSnapshot(ctx context.Context, name, volumeID string) (string, error)
	ListSnapshots(ctx context.Context, filters map[string]string) ([]Snapshot, string, error)
	DeleteSnapshot(ctx context.Context, snapshotID string) error
	WaitSnapshotReady(ctx context.Context, snapshotID string) error
	GetSnapshotByID(ctx context.Context, snapshotID string) (*Snapshot, error)
	GetSnapshotByName(ctx context.Context, name string) (*Snapshot, error)
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
	SnapshotID       string
}

// VM represents a CloudStack Virtual Machine.
type VM struct {
	ID     string
	ZoneID string
}

// Snapshot contains all the information associated with a CloudStack Snapshot.
type Snapshot struct {
	// Unique identifier.
	ID string `json:"id"`

	// Date created.
	Created string `json:"created"`

	// Display name.
	Name string `json:"name"`

	// ID of the Volume from which this Snapshot was created.
	VolumeID string `json:"volume_id"`

	// ID of the Zone.
	ZoneID string `json:"zone_id"`

	// Currect state of the Snapshot.
	State string `json:"state"`

	// Size of the Snapshot, in GB.
	VirtualSize int `json:"size"`
}

// Specific errors
var (
	ErrNotFound       = errors.New("not found")
	ErrTooManyResults = errors.New("too many results")
)

// client is the implementation of Interface.
type client struct {
	*cloudstack.CloudStackClient
}

// New creates a new cloud connector, given its configuration.
func New(config *Config) Interface {
	csClient := cloudstack.NewAsyncClient(config.APIURL, config.APIKey, config.SecretKey, config.VerifySSL)
	return &client{csClient}
}
