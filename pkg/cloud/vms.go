package cloud

import (
	"context"

	"github.com/apache/cloudstack-go/v2/cloudstack"
)

func (c *client) GetVMByID(ctx context.Context, vmID string) (*VM, error) {
	p := c.VirtualMachine.NewListVirtualMachinesParams()
	p.SetId(vmID)
	p.SetListall(true)
	if c.projectID != "" {
		p.SetProjectid(c.projectID)
	}
	logAPICall(ctx, "ListVirtualMachines", map[string]string{
		paramID: vmID,
	})

	return c.listVM(p)
}

func (c *client) getVMByName(ctx context.Context, name string) (*VM, error) {
	p := c.VirtualMachine.NewListVirtualMachinesParams()
	p.SetName(name)
	p.SetListall(true)
	if c.projectID != "" {
		p.SetProjectid(c.projectID)
	}
	logAPICall(ctx, "ListVirtualMachines", map[string]string{
		paramName: name,
	})

	return c.listVM(p)
}

func (c *client) listVM(p *cloudstack.ListVirtualMachinesParams) (*VM, error) {
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
