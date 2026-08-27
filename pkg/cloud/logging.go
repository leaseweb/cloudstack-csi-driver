package cloud

import (
	"context"

	"k8s.io/klog/v2"
)

// Names of the CloudStack API request parameters, used as keys when logging
// API calls.
const (
	paramAvailable      = "available"
	paramDiskOfferingID = "diskofferingid"
	paramID             = "id"
	paramName           = "name"
	paramProjectID      = "projectid"
	paramSize           = "size"
	paramVirtualMachine = "virtualmachineid"
	paramZoneID         = "zoneid"
)

// logAPICall logs a CloudStack API call and its parameters.
func logAPICall(ctx context.Context, command string, params map[string]string) {
	klog.FromContext(ctx).V(2).Info("CloudStack API call", "command", command, "params", params)
}
