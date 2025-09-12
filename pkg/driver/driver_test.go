package driver

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/leaseweb/cloudstack-csi-driver/pkg/cloud/fake"
	"github.com/leaseweb/cloudstack-csi-driver/pkg/mount"
)

func TestNewDriver(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := t.Context()
	fakeCloud := fake.New()
	fakeMounter := mount.NewFake()
	testCases := []struct {
		name          string
		o             *Options
		expectError   bool
		hasController bool
		hasNode       bool
	}{
		{
			name: "Valid driver controllerMode",
			o: &Options{
				Mode:              ControllerMode,
				VolumeAttachLimit: 16,
			},
			expectError:   false,
			hasController: true,
			hasNode:       false,
		},
		{
			name: "Valid driver nodeMode",
			o: &Options{
				Mode: NodeMode,
			},
			expectError:   false,
			hasController: false,
			hasNode:       true,
		},
		{
			name: "Valid driver allMode",
			o: &Options{
				Mode:              AllMode,
				VolumeAttachLimit: 16,
			},
			expectError:   false,
			hasController: true,
			hasNode:       true,
		},
		{
			name: "Invalid driver options",
			o: &Options{
				Mode: "InvalidMode",
			},
			expectError:   true,
			hasController: false,
			hasNode:       false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			driver, err := NewDriver(ctx, fakeCloud, tc.o, fakeMounter)
			if tc.hasNode && driver.node == nil {
				t.Fatalf("Expected driver to have node but driver does not have node")
			}
			if tc.hasController && driver.controller == nil {
				t.Fatalf("Expected driver to have controller but driver does not have controller")
			}
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
