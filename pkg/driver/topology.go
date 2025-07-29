package driver

import (
	"context"
	"errors"
	"fmt"

	"github.com/container-storage-interface/spec/lib/go/csi"

	"github.com/leaseweb/cloudstack-csi-driver/pkg/cloud"
)

// pickAvailabilityZone selects 1 zone given topology requirement.
// if not found, empty string is returned.
func pickAvailabilityZone(ctx context.Context, connector cloud.Cloud, requirement *csi.TopologyRequirement) (string, error) {
	if requirement == nil {
		return "", nil
	}
	for _, topology := range requirement.GetPreferred() {
		zone, exists := topology.GetSegments()[WellKnownZoneTopologyKey]
		if exists {
			zoneid, err := connector.GetZoneIDByName(ctx, zone)
			if err != nil {
				return "", fmt.Errorf("failed to get zone ID by name: %w", err)
			}

			return zoneid, nil
		}

		zoneid, exists := topology.GetSegments()[ZoneTopologyKey]
		if exists {
			return zoneid, nil
		}
	}
	for _, topology := range requirement.GetRequisite() {
		zone, exists := topology.GetSegments()[WellKnownZoneTopologyKey]
		if exists {
			zoneid, err := connector.GetZoneIDByName(ctx, zone)
			if err != nil {
				return "", fmt.Errorf("failed to get zone ID by name: %w", err)
			}

			return zoneid, nil
		}
		zoneid, exists := topology.GetSegments()[ZoneTopologyKey]
		if exists {
			return zoneid, nil
		}
	}

	return "", nil
}

func getZonesFromTopology(topList []*csi.Topology) ([]string, []string, error) {
	zoneIDs := []string{}
	zoneNames := []string{}
	for _, top := range topList {
		if top.GetSegments() == nil {
			return nil, nil, errors.New("topologies specified but no segments")
		}

		zoneID, zoneName, err := getZoneFromSegment(top.GetSegments())
		if err != nil {
			return nil, nil, fmt.Errorf("could not get zone from topology: %w", err)
		}
		if len(zoneID) > 0 {
			zoneIDs = append(zoneIDs, zoneID)
		}
		if len(zoneName) > 0 {
			zoneNames = append(zoneNames, zoneName)
		}
	}

	return zoneIDs, zoneNames, nil
}

func getZoneFromSegment(seg map[string]string) (string, string, error) {
	var zoneName, zoneID string
	for k, v := range seg {
		switch k {
		case ZoneTopologyKey:
			zoneID = v
		case WellKnownZoneTopologyKey:
			zoneName = v
		default:
			return "", "", fmt.Errorf("topology segment has unknown key %v", k)
		}
	}
	if len(zoneID) == 0 && len(zoneName) == 0 {
		return "", "", fmt.Errorf("topology specified but could not find zone in segment: %v", seg)
	}

	return zoneID, zoneName, nil
}
