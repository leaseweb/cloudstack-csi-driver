package cloud

import (
	"context"
	"errors"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"k8s.io/klog/v2"
)

func (c *client) GetZoneIDByName(ctx context.Context, name string) (string, error) {
	logger := klog.FromContext(ctx)
	logger.V(2).Info("CloudStack API call", "command", "GetZoneID", "params", map[string]string{
		"name": name,
	})
	id, count, err := c.Zone.GetZoneID(name, cloudstack.OptionFunc(func(_ *cloudstack.CloudStackClient, p interface{}) error {
		ps, ok := p.(*cloudstack.ListZonesParams)
		if !ok {
			return errors.New("invalid params type")
		}
		ps.SetAvailable(true)

		return nil
	}))
	if err != nil {
		return "", err
	}

	if count == 0 {
		return "", errors.New("zone not found")
	}

	return id, nil
}

func (c *client) ListZonesID(ctx context.Context) ([]string, error) {
	logger := klog.FromContext(ctx)
	result := make([]string, 0)
	p := c.Zone.NewListZonesParams()
	p.SetAvailable(true)
	logger.V(2).Info("CloudStack API call", "command", "ListZones", "params", map[string]string{
		"available": "true",
	})
	r, err := c.Zone.ListZones(p)
	if err != nil {
		return result, err
	}
	for _, zone := range r.Zones {
		result = append(result, zone.Id)
	}

	return result, nil
}
