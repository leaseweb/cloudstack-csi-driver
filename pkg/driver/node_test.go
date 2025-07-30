package driver

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/ktesting"

	"github.com/leaseweb/cloudstack-csi-driver/pkg/cloud"
	"github.com/leaseweb/cloudstack-csi-driver/pkg/mount"
	"github.com/leaseweb/cloudstack-csi-driver/pkg/util"
)

const (
	sourceTest = "./source_test"
	targetTest = "./target_test"
)

func TestNodePublishVolumeIdempotentMount(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Test requires root")
	}
	logger := ktesting.NewLogger(t, ktesting.NewConfig(ktesting.Verbosity(10), ktesting.BufferLogs(true)))
	ctx := klog.NewContext(context.Background(), logger)

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	driver := &NodeService{
		connector:   cloud.NewMockCloud(mockCtl),
		mounter:     mount.New(),
		volumeLocks: util.NewVolumeLocks(),
	}

	err := driver.mounter.MakeDir(sourceTest)
	require.NoError(t, err)
	err = driver.mounter.MakeDir(targetTest)
	require.NoError(t, err)

	volCapAccessMode := csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}
	volCapAccessType := csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{}}
	req := csi.NodePublishVolumeRequest{
		VolumeCapability:  &csi.VolumeCapability{AccessMode: &volCapAccessMode, AccessType: &volCapAccessType},
		VolumeId:          "vol_1",
		TargetPath:        targetTest,
		StagingTargetPath: sourceTest,
		Readonly:          true,
	}

	underlyingLogger, ok := logger.GetSink().(ktesting.Underlier)
	if !ok {
		t.Fatalf("should have had ktesting LogSink, got %T", logger.GetSink())
	}

	_, err = driver.NodePublishVolume(ctx, &req)
	require.NoError(t, err)
	_, err = driver.NodePublishVolume(ctx, &req)
	require.NoError(t, err)

	logEntries := underlyingLogger.GetBuffer().String()
	assert.Contains(t, logEntries, "Target path is already mounted")

	// ensure the target not be mounted twice
	targetAbs, err := filepath.Abs(targetTest)
	require.NoError(t, err)

	mountList, err := driver.mounter.List()
	require.NoError(t, err)
	mountPointNum := 0
	for _, mountPoint := range mountList {
		if mountPoint.Path == targetAbs {
			mountPointNum++
		}
	}
	assert.Equal(t, 1, mountPointNum)
	err = driver.mounter.Unmount(targetTest)
	require.NoError(t, err)
	_ = driver.mounter.Unmount(targetTest)
	err = os.RemoveAll(sourceTest)
	require.NoError(t, err)
	err = os.RemoveAll(targetTest)
	require.NoError(t, err)
}

func TestNodeGetInfo(t *testing.T) {
	logger := ktesting.NewLogger(t, ktesting.NewConfig(ktesting.Verbosity(10), ktesting.BufferLogs(true)))
	ctx := klog.NewContext(context.Background(), logger)

	testCases := []struct {
		name        string
		cloudMock   func(ctrl *gomock.Controller) *cloud.MockCloud
		vmName      string
		vmZoneID    string
		vmZoneName  string
		expectedRes *csi.NodeGetInfoResponse
	}{
		{
			name: "test-1",
			cloudMock: func(ctrl *gomock.Controller) *cloud.MockCloud {
				mockCloud := cloud.NewMockCloud(ctrl)
				mockCloud.EXPECT().GetNodeInfo(ctx, "test-vm").Return(&cloud.VM{
					ID:       "test-vm",
					ZoneID:   "test-zone-id",
					ZoneName: "test-zone-name",
				}, nil)

				return mockCloud
			},
			vmName:     "test-vm",
			vmZoneID:   "test-zone-id",
			vmZoneName: "test-zone-name",
			expectedRes: &csi.NodeGetInfoResponse{
				NodeId: "test-vm",
				AccessibleTopology: &csi.Topology{
					Segments: map[string]string{
						ZoneTopologyKey:          "test-zone-id",
						WellKnownZoneTopologyKey: "test-zone-name",
						OSTopologyKey:            runtime.GOOS,
					},
				},
				MaxVolumesPerNode: DefaultMaxVolAttachLimit,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()

			mockCloud := tc.cloudMock(mockCtl)
			driver := &NodeService{
				connector:         mockCloud,
				mounter:           mount.New(),
				nodeName:          tc.vmName,
				maxVolumesPerNode: DefaultMaxVolAttachLimit,
				volumeLocks:       util.NewVolumeLocks(),
			}

			res, err := driver.NodeGetInfo(ctx, &csi.NodeGetInfoRequest{})
			require.NoError(t, err)
			assert.Equal(t, tc.expectedRes, res)
		})
	}
}
