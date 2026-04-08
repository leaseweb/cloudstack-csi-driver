// Package driver provides the implementation of the CSI plugin.
//
// It contains the gRPC server implementation of CSI specification.
package driver

import (
	"context"
	"fmt"
	"net"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"

	"github.com/leaseweb/cloudstack-csi-driver/pkg/cloud"
	"github.com/leaseweb/cloudstack-csi-driver/pkg/mount"
	"github.com/leaseweb/cloudstack-csi-driver/pkg/util"
)

// Interface is the CloudStack CSI driver interface.
type Interface interface {
	// Run the CSI driver gRPC server
	Run(ctx context.Context) error
}

type Driver struct {
	csi.UnimplementedIdentityServer

	controller *ControllerService
	node       *NodeService
	srv        *grpc.Server
	options    *Options
}

// NewDriver instantiates a new CloudStack CSI driver.
func NewDriver(ctx context.Context, csConnector cloud.Cloud, options *Options, mounter mount.Mounter) (*Driver, error) {
	logger := klog.FromContext(ctx)
	logger.Info("Driver starting", "Driver", DriverName, "Version", driverVersion)

	if err := validateMode(options.Mode); err != nil {
		return nil, fmt.Errorf("invalid driver options: %w", err)
	}

	driver := &Driver{
		options: options,
	}

	switch options.Mode {
	case ControllerMode:
		driver.controller = NewControllerService(csConnector)
	case NodeMode:
		driver.node = NewNodeService(csConnector, mounter, options)
	case AllMode:
		driver.controller = NewControllerService(csConnector)
		driver.node = NewNodeService(csConnector, mounter, options)
	default:
		return nil, fmt.Errorf("unknown mode: %s", options.Mode)
	}

	return driver, nil
}

func (d *Driver) Run(ctx context.Context) error {
	logger := klog.FromContext(ctx)
	scheme, addr, err := util.ParseEndpoint(d.options.Endpoint)
	if err != nil {
		return err
	}

	listener, err := net.Listen(scheme, addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// Log every request and payloads (request + response)
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			resp, err := handler(klog.NewContext(ctx, logger), req)
			if err != nil {
				logger.Error(err, "GRPC method failed", "method", info.FullMethod)
			}

			return resp, err
		}),
	}
	d.srv = grpc.NewServer(opts...)
	csi.RegisterIdentityServer(d.srv, d)

	switch d.options.Mode {
	case ControllerMode:
		csi.RegisterControllerServer(d.srv, d.controller)
	case NodeMode:
		csi.RegisterNodeServer(d.srv, d.node)
	case AllMode:
		csi.RegisterControllerServer(d.srv, d.controller)
		csi.RegisterNodeServer(d.srv, d.node)
	default:
		return fmt.Errorf("unknown mode: %s", d.options.Mode)
	}

	logger.Info("Listening for connections", "address", listener.Addr())

	return d.srv.Serve(listener)
}

func validateMode(mode Mode) error {
	if mode != AllMode && mode != ControllerMode && mode != NodeMode {
		return fmt.Errorf("mode is not supported (actual: %s, supported: %v)", mode, []Mode{AllMode, ControllerMode, NodeMode})
	}

	return nil
}
