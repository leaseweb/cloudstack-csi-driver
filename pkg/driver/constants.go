package driver

// DriverName is the name of the CSI plugin.
const DriverName = "csi.cloudstack.apache.org"

// Mode is the operating mode of the CSI driver.
type Mode string

// Driver operating modes.
const (
	// ControllerMode is the mode that only starts the controller service.
	ControllerMode Mode = "controller"
	// NodeMode is the mode that only starts the node service.
	NodeMode Mode = "node"
	// AllMode is the mode that only starts both the controller and the node service.
	AllMode Mode = "all"
)

// constants for default command line flag values.
const (
	// DefaultCSIEndpoint is the default CSI endpoint for the driver.
	DefaultCSIEndpoint             = "unix://tmp/csi.sock"
	DefaultMaxVolAttachLimit int64 = 256
)

// Filesystem types.
const (
	// FSTypeExt2 represents the ext2 filesystem type.
	FSTypeExt2 = "ext2"
	// FSTypeExt3 represents the ext3 filesystem type.
	FSTypeExt3 = "ext3"
	// FSTypeExt4 represents the ext4 filesystem type.
	FSTypeExt4 = "ext4"
	// FSTypeXfs represents the xfs filesystem type.
	FSTypeXfs = "xfs"
)

// Topology keys.
const (
	WellKnownZoneTopologyKey = "topology.kubernetes.io/zone"
	// ZoneTopologyKey is deprecated in favor of WellKnownZoneTopologyKey, only here for backwards compatibility.
	ZoneTopologyKey = "topology." + DriverName + "/zone"
	OSTopologyKey   = "kubernetes.io/os"
)

// Volume parameters keys.
const (
	DiskOfferingKey = DriverName + "/disk-offering-id"
)

const deviceIDContextKey = "deviceID"
