package cloud

import (
	"context"

	"k8s.io/klog/v2"
)

func (c *client) GetVMByID(ctx context.Context, vmID string) (*VM, error) {
	logger := klog.FromContext(ctx)
	p := c.VirtualMachine.NewListVirtualMachinesParams()
	p.SetId(vmID)
	p.SetListall(true)
	if c.projectID != "" {
		p.SetProjectid(c.projectID)
	}
	logger.V(2).Info("CloudStack API call", "command", "ListVirtualMachines", "params", map[string]string{
		"id": vmID,
	})
	l, err := c.VirtualMachine.ListVirtualMachines(p)
	if err != nil {
		return nil, err
	}
	if l.Count == 0 {
		return nil, ErrNotFound
	}
	if l.Count > 1 {
		return nil, ErrTooManyResults
	}
	vm := l.VirtualMachines[0]

	return &VM{
		ID:       vm.Id,
		ZoneID:   vm.Zoneid,
		ZoneName: vm.Zonename,
	}, nil
}

func (c *client) getVMByName(ctx context.Context, name string) (*VM, error) {
	logger := klog.FromContext(ctx)
	p := c.VirtualMachine.NewListVirtualMachinesParams()
	p.SetName(name)
	p.SetListall(true)
	if c.projectID != "" {
		p.SetProjectid(c.projectID)
	}
	logger.V(2).Info("CloudStack API call", "command", "ListVirtualMachines", "params", map[string]string{
		"name": name,
	})
	l, err := c.VirtualMachine.ListVirtualMachines(p)
	if err != nil {
		return nil, err
	}
	if l.Count == 0 {
		return nil, ErrNotFound
	}
	if l.Count > 1 {
		return nil, ErrTooManyResults
	}
	vm := l.VirtualMachines[0]

	return &VM{
		ID:       vm.Id,
		ZoneID:   vm.Zoneid,
		ZoneName: vm.Zonename,
	}, nil
}
