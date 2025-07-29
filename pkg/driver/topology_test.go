package driver

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"go.uber.org/mock/gomock"

	"github.com/leaseweb/cloudstack-csi-driver/pkg/cloud"
)

const (
	WellKnownZoneName = "foobar"
	WellKnownZoneID   = "dfc1a085-eca4-4495-a19a-e87fbbc994a2"
)

func TestPickAvailabilityZone(t *testing.T) {
	ctx := context.Background()
	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()
	mockCloud := cloud.NewMockCloud(mockCtl)

	testCases := []struct {
		name         string
		requirement  *csi.TopologyRequirement
		expMockCalls func(mockCloud *cloud.MockCloud)
		expZone      string
	}{
		{
			name: "Return WellKnownZoneTopologyKey if present from preferred",
			requirement: &csi.TopologyRequirement{
				Requisite: []*csi.Topology{
					{
						Segments: map[string]string{ZoneTopologyKey: ""},
					},
				},
				Preferred: []*csi.Topology{
					{
						Segments: map[string]string{ZoneTopologyKey: FakeZoneID, WellKnownZoneTopologyKey: WellKnownZoneName},
					},
				},
			},
			expMockCalls: func(mockCloud *cloud.MockCloud) {
				mockCloud.EXPECT().GetZoneIDByName(gomock.Eq(ctx), gomock.Eq(WellKnownZoneName)).Return(WellKnownZoneID, nil)
			},
			expZone: WellKnownZoneID,
		},
		{
			name: "Return WellKnownZoneTopologyKey if present from requisite",
			requirement: &csi.TopologyRequirement{
				Requisite: []*csi.Topology{
					{
						Segments: map[string]string{ZoneTopologyKey: FakeZoneID, WellKnownZoneTopologyKey: WellKnownZoneName},
					},
				},
			},
			expMockCalls: func(mockCloud *cloud.MockCloud) {
				mockCloud.EXPECT().GetZoneIDByName(gomock.Eq(ctx), gomock.Eq(WellKnownZoneName)).Return(WellKnownZoneID, nil)
			},
			expZone: WellKnownZoneID,
		},
		{
			name: "Pick from preferred",
			requirement: &csi.TopologyRequirement{
				Requisite: []*csi.Topology{
					{
						Segments: map[string]string{ZoneTopologyKey: ""},
					},
				},
				Preferred: []*csi.Topology{
					{
						Segments: map[string]string{ZoneTopologyKey: FakeZoneID},
					},
				},
			},
			expMockCalls: func(_ *cloud.MockCloud) {},
			expZone:      FakeZoneID,
		},
		{
			name: "Pick from requisite",
			requirement: &csi.TopologyRequirement{
				Requisite: []*csi.Topology{
					{
						Segments: map[string]string{ZoneTopologyKey: FakeZoneID},
					},
				},
			},
			expMockCalls: func(_ *cloud.MockCloud) {},
			expZone:      FakeZoneID,
		},
		{
			name: "Pick from empty topology",
			requirement: &csi.TopologyRequirement{
				Preferred: []*csi.Topology{{}},
				Requisite: []*csi.Topology{{}},
			},
			expMockCalls: func(_ *cloud.MockCloud) {},
			expZone:      "",
		},
		{
			name:         "Topology Requirement is nil",
			requirement:  nil,
			expMockCalls: func(_ *cloud.MockCloud) {},
			expZone:      "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.expMockCalls(mockCloud)

			actual, err := pickAvailabilityZone(ctx, mockCloud, tc.requirement)
			if err != nil {
				t.Fatalf("Expected no error, got error: %v", err)
			}
			if actual != tc.expZone {
				t.Fatalf("Expected zone ID %v, got zone ID: %v", tc.expZone, actual)
			}
		})
	}
}

func TestGetZonesFromTopology(t *testing.T) {
	testCases := []struct {
		name         string
		topList      []*csi.Topology
		expZoneIDs   []string
		expZoneNames []string
		expErr       error
	}{
		{
			name: "Test with zoneID and zoneName",
			topList: []*csi.Topology{
				{
					Segments: map[string]string{ZoneTopologyKey: "zone1", WellKnownZoneTopologyKey: "zone1-name"},
				},
			},
			expZoneIDs:   []string{"zone1"},
			expZoneNames: []string{"zone1-name"},
			expErr:       nil,
		},
		{
			name: "Test with zoneID only",
			topList: []*csi.Topology{
				{
					Segments: map[string]string{ZoneTopologyKey: "zone1"},
				},
			},
			expZoneIDs:   []string{"zone1"},
			expZoneNames: []string{},
			expErr:       nil,
		},
		{
			name: "Test with zoneName only",
			topList: []*csi.Topology{
				{
					Segments: map[string]string{WellKnownZoneTopologyKey: "zone1-name"},
				},
			},
			expZoneIDs:   []string{},
			expZoneNames: []string{"zone1-name"},
			expErr:       nil,
		},
		{
			name: "Test with no zones",
			topList: []*csi.Topology{
				{
					Segments: map[string]string{},
				},
			},
			expZoneIDs:   nil,
			expZoneNames: nil,
			expErr:       errors.New("could not get zone from topology: topology specified but could not find zone in segment: map[]"),
		},
		{
			name: "Test with invalid segment",
			topList: []*csi.Topology{
				{
					Segments: map[string]string{ZoneTopologyKey: "zone1", "invalid": "invalid"},
				},
			},
			expZoneIDs:   nil,
			expZoneNames: nil,
			expErr:       errors.New("could not get zone from topology: topology segment has unknown key invalid"),
		},
		{
			name: "Test with nil topology",
			topList: []*csi.Topology{
				{
					Segments: nil,
				},
			},
			expZoneIDs:   nil,
			expZoneNames: nil,
			expErr:       errors.New("topologies specified but no segments"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			zoneIDs, zoneNames, err := getZonesFromTopology(tc.topList)
			if err != nil {
				if tc.expErr != nil {
					if err.Error() != tc.expErr.Error() {
						t.Fatalf("Expected error %v, got error: %v", tc.expErr, err)
					}
				} else {
					t.Fatalf("Expected no error, got error: %v", err)
				}
			}
			if !reflect.DeepEqual(zoneIDs, tc.expZoneIDs) {
				t.Fatalf("Expected zoneIDs %v, got zoneIDs: %v", tc.expZoneIDs, zoneIDs)
			}
			if !reflect.DeepEqual(zoneNames, tc.expZoneNames) {
				t.Fatalf("Expected zoneNames %v, got zoneNames: %v", tc.expZoneNames, zoneNames)
			}
		})
	}
}
