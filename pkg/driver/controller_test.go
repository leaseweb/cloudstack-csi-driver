package driver

import (
	"context"
	"reflect"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/leaseweb/cloudstack-csi-driver/pkg/cloud"
	"github.com/leaseweb/cloudstack-csi-driver/pkg/util"
)

var (
	FakeCapacityGiB    = 1
	FakeVolName        = "CSIVolumeName"
	FakeVolID          = "CSIVolumeID"
	FakeZoneID         = "cc269538-2b2c-438d-bb9a-b531b2aba7d9"
	FakeDiskOfferingID = "9743fd77-0f5d-4ef9-b2f8-f194235c769c"
	FakeVol            = cloud.Volume{
		ID:     FakeVolID,
		Name:   FakeVolName,
		Size:   int64(FakeCapacityGiB),
		ZoneID: FakeZoneID,
	}
)

func TestDetermineVolumeSizeInGB(t *testing.T) {
	cases := []struct {
		name          string
		capacityRange *csi.CapacityRange
		expectedSize  int64
		expectError   bool
	}{
		{"no range", nil, 1, false},
		{"only limit", &csi.CapacityRange{LimitBytes: 100 * 1024 * 1024 * 1024}, 1, false},
		{"only limit (too small)", &csi.CapacityRange{LimitBytes: 1024 * 1024}, 0, true},
		{"only required", &csi.CapacityRange{RequiredBytes: 50 * 1024 * 1024 * 1024}, 50, false},
		{"required and limit", &csi.CapacityRange{RequiredBytes: 25 * 1024 * 1024 * 1024, LimitBytes: 100 * 1024 * 1024 * 1024}, 25, false},
		{"required = limit", &csi.CapacityRange{RequiredBytes: 30 * 1024 * 1024 * 1024, LimitBytes: 30 * 1024 * 1024 * 1024}, 30, false},
		{"required = limit (not GB int)", &csi.CapacityRange{RequiredBytes: 3_000_000_000, LimitBytes: 3_000_000_000}, 0, true},
		{"no int GB int possible", &csi.CapacityRange{RequiredBytes: 4_000_000_000, LimitBytes: 1_000_001_000}, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := &csi.CreateVolumeRequest{
				CapacityRange: c.capacityRange,
			}
			size, err := determineVolumeSizeInGB(req)
			if err != nil && !c.expectError {
				t.Errorf("Unexepcted error: %v", err.Error())
			}
			if err == nil && c.expectError {
				t.Error("Expected an error")
			}
			if size != c.expectedSize {
				t.Errorf("Expected size %v, got %v", c.expectedSize, size)
			}
		})
	}
}

func TestCreateVolume(t *testing.T) {
	stdVolCap := []*csi.VolumeCapability{
		{
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{},
			},
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		},
	}
	invalidVolCap := []*csi.VolumeCapability{
		{
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_SINGLE_WRITER,
			},
		},
	}
	stdVolSize := int64(5 * 1024 * 1024 * 1024)
	stdCapRange := &csi.CapacityRange{RequiredBytes: stdVolSize}
	stdParams := map[string]string{
		DiskOfferingKey: FakeDiskOfferingID,
	}

	cases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "success valid request",
			testFunc: func(t *testing.T) {
				t.Helper()
				req := &csi.CreateVolumeRequest{
					Name:               "my-volume",
					VolumeCapabilities: stdVolCap,
					CapacityRange:      stdCapRange,
					Parameters:         stdParams,
				}

				ctx := context.Background()

				mockVol := cloud.Volume{
					ID:             FakeVolID,
					Name:           req.GetName(),
					DiskOfferingID: FakeDiskOfferingID,
					Size:           int64(FakeCapacityGiB),
					ZoneID:         FakeZoneID,
				}

				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockCloud := cloud.NewMockCloud(mockCtl)
				mockCloud.EXPECT().CreateVolume(gomock.Eq(ctx), FakeDiskOfferingID, "", gomock.Eq(req.GetName()), gomock.Any()).Return(&mockVol, nil)
				mockCloud.EXPECT().GetVolumeByName(gomock.Eq(ctx), gomock.Eq(req.GetName())).Return(nil, cloud.ErrNotFound)
				fakeCs := NewControllerService(mockCloud)

				if _, err := fakeCs.CreateVolume(ctx, req); err != nil {
					srvErr, ok := status.FromError(err)
					if !ok {
						t.Fatalf("Could not get error status code from error: %v", srvErr)
					}
					t.Fatalf("Unexpected error: %v message %s", srvErr.Code(), srvErr.Message())
				}
			},
		},
		{
			name: "fail no name",
			testFunc: func(t *testing.T) {
				t.Helper()
				req := &csi.CreateVolumeRequest{
					Name:               "",
					VolumeCapabilities: stdVolCap,
					CapacityRange:      stdCapRange,
					Parameters:         stdParams,
				}

				ctx := context.Background()

				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockCloud := cloud.NewMockCloud(mockCtl)
				fakeCs := NewControllerService(mockCloud)

				if _, err := fakeCs.CreateVolume(ctx, req); err != nil {
					srvErr, ok := status.FromError(err)
					if !ok {
						t.Fatalf("Could not get error status code from error: %v", srvErr)
					}
					if srvErr.Code() != codes.InvalidArgument {
						t.Fatalf("Expected error code %d, got %d message %s", codes.InvalidArgument, srvErr.Code(), srvErr.Message())
					}
				} else {
					t.Fatalf("Expected error %v, got no error", codes.InvalidArgument)
				}
			},
		},
		{
			name: "fail invalid volume capability",
			testFunc: func(t *testing.T) {
				t.Helper()
				req := &csi.CreateVolumeRequest{
					Name:               "my-volume",
					VolumeCapabilities: invalidVolCap,
					CapacityRange:      stdCapRange,
					Parameters:         stdParams,
				}

				ctx := context.Background()

				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockCloud := cloud.NewMockCloud(mockCtl)
				fakeCs := NewControllerService(mockCloud)

				_, err := fakeCs.CreateVolume(ctx, req)
				if err == nil {
					t.Fatalf("Expected error, got no error")
				}

				srvErr, ok := status.FromError(err)
				if !ok {
					t.Fatalf("Could not get error status code from error: %v", srvErr)
				}
				if srvErr.Code() != codes.InvalidArgument {
					t.Fatalf("Expected error code %d, got %d message %s", codes.InvalidArgument, srvErr.Code(), srvErr.Message())
				}
			},
		},
		{
			name: "fail no disk offering provided",
			testFunc: func(t *testing.T) {
				t.Helper()
				req := &csi.CreateVolumeRequest{
					Name:               "my-volume",
					VolumeCapabilities: stdVolCap,
					CapacityRange:      stdCapRange,
					Parameters:         map[string]string{},
				}

				ctx := context.Background()

				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockCloud := cloud.NewMockCloud(mockCtl)
				fakeCs := NewControllerService(mockCloud)

				if _, err := fakeCs.CreateVolume(ctx, req); err != nil {
					srvErr, ok := status.FromError(err)
					if !ok {
						t.Fatalf("Could not get error status code from error: %v", srvErr)
					}
					if srvErr.Code() != codes.InvalidArgument {
						t.Fatalf("Expected error code %d, got %d message %s", codes.InvalidArgument, srvErr.Code(), srvErr.Message())
					}
				} else {
					t.Fatalf("Expected error %v, got no error", codes.InvalidArgument)
				}
			},
		},
		{
			name: "success volume already exists",
			testFunc: func(t *testing.T) {
				t.Helper()
				req := &csi.CreateVolumeRequest{
					Name:               "my-volume",
					VolumeCapabilities: stdVolCap,
					CapacityRange:      stdCapRange,
					Parameters:         stdParams,
				}

				ctx := context.Background()

				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockVol := cloud.Volume{
					ID:             FakeVolID,
					Name:           req.GetName(),
					DiskOfferingID: FakeDiskOfferingID,
					Size:           stdVolSize,
					ZoneID:         FakeZoneID,
				}

				mockCloud := cloud.NewMockCloud(mockCtl)
				mockCloud.EXPECT().GetVolumeByName(gomock.Eq(ctx), gomock.Eq(req.GetName())).Return(&mockVol, nil)
				mockCloud.EXPECT().CreateVolume(gomock.Eq(ctx), FakeDiskOfferingID, "", gomock.Eq(req.GetName()), gomock.Any()).Times(0)
				fakeCs := NewControllerService(mockCloud)

				if _, err := fakeCs.CreateVolume(ctx, req); err != nil {
					srvErr, ok := status.FromError(err)
					if !ok {
						t.Fatalf("Could not get error status code from error: %v", srvErr)
					}
					t.Fatalf("Unexpected error: %v message %s", srvErr.Code(), srvErr.Message())
				}
			},
		},
		{
			name: "success same name and same capacity",
			testFunc: func(t *testing.T) {
				t.Helper()
				req := &csi.CreateVolumeRequest{
					Name:               "my-volume",
					VolumeCapabilities: stdVolCap,
					CapacityRange:      stdCapRange,
					Parameters:         stdParams,
				}
				extraReq := &csi.CreateVolumeRequest{
					Name:               "my-volume",
					VolumeCapabilities: stdVolCap,
					CapacityRange:      stdCapRange,
					Parameters:         stdParams,
				}
				expVol := &csi.Volume{
					VolumeId:      FakeVolID,
					CapacityBytes: stdVolSize,
					VolumeContext: stdParams,
					AccessibleTopology: []*csi.Topology{
						{
							Segments: map[string]string{ZoneTopologyKey: FakeZoneID},
						},
					},
				}

				ctx := context.Background()

				mockVol := cloud.Volume{
					ID:             FakeVolID,
					Name:           req.GetName(),
					DiskOfferingID: FakeDiskOfferingID,
					Size:           stdVolSize,
					ZoneID:         FakeZoneID,
				}

				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockCloud := cloud.NewMockCloud(mockCtl)
				mockCloud.EXPECT().GetVolumeByName(gomock.Eq(ctx), gomock.Eq(req.GetName())).Return(nil, cloud.ErrNotFound)
				mockCloud.EXPECT().CreateVolume(gomock.Eq(ctx), FakeDiskOfferingID, "", gomock.Eq(req.GetName()), gomock.Any()).Return(&mockVol, nil)
				fakeCs := NewControllerService(mockCloud)

				if _, err := fakeCs.CreateVolume(ctx, req); err != nil {
					srvErr, ok := status.FromError(err)
					if !ok {
						t.Fatalf("Could not get error status code from error: %v", srvErr)
					}
					t.Fatalf("Unexpected error: %v message %s", srvErr.Code(), srvErr.Message())
				}

				// Subsequent request returns the existing volume
				mockCloud.EXPECT().GetVolumeByName(gomock.Eq(ctx), gomock.Eq(extraReq.GetName())).Return(&mockVol, nil)
				mockCloud.EXPECT().CreateVolume(gomock.Eq(ctx), FakeDiskOfferingID, "", gomock.Eq(extraReq.GetName()), gomock.Any()).Times(0)
				resp, err := fakeCs.CreateVolume(ctx, extraReq)
				if err != nil {
					srvErr, ok := status.FromError(err)
					if !ok {
						t.Fatalf("Could not get error status code from error: %v", srvErr)
					}
					t.Fatalf("Unexpected error: %v message %s", srvErr.Code(), srvErr.Message())
				}

				vol := resp.GetVolume()
				if vol == nil {
					t.Fatalf("Expected volume, got nil")
				}
				if vol.GetCapacityBytes() != expVol.GetCapacityBytes() {
					t.Fatalf("Expected volume capacity %d, got %d", expVol.GetCapacityBytes(), vol.GetCapacityBytes())
				}
				if vol.GetVolumeId() != expVol.GetVolumeId() {
					t.Fatalf("Expected volume ID %s, got %s", expVol.GetVolumeId(), vol.GetVolumeId())
				}
				if vol.GetAccessibleTopology() != nil {
					if !reflect.DeepEqual(vol.GetAccessibleTopology(), expVol.GetAccessibleTopology()) {
						t.Fatalf("Expected volume topology %v, got %v", expVol.GetAccessibleTopology(), vol.GetAccessibleTopology())
					}
				}
			},
		},
		{
			name: "fail same name and different capacity",
			testFunc: func(t *testing.T) {
				t.Helper()
				req := &csi.CreateVolumeRequest{
					Name:               "my-volume",
					VolumeCapabilities: stdVolCap,
					CapacityRange:      stdCapRange,
					Parameters:         stdParams,
				}
				extraReq := &csi.CreateVolumeRequest{
					Name:               "my-volume",
					VolumeCapabilities: stdVolCap,
					CapacityRange:      &csi.CapacityRange{RequiredBytes: int64(6 * 1024 * 1024 * 1024)},
					Parameters:         stdParams,
				}

				ctx := context.Background()

				mockVol := cloud.Volume{
					ID:             FakeVolID,
					Name:           req.GetName(),
					DiskOfferingID: FakeDiskOfferingID,
					ZoneID:         FakeZoneID,
				}
				volSize, err := determineVolumeSizeInGB(req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				mockVol.Size = util.GigaBytesToBytes(volSize)

				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockCloud := cloud.NewMockCloud(mockCtl)
				mockCloud.EXPECT().GetVolumeByName(gomock.Eq(ctx), gomock.Eq(req.GetName())).Return(nil, cloud.ErrNotFound)
				mockCloud.EXPECT().CreateVolume(gomock.Eq(ctx), FakeDiskOfferingID, "", gomock.Eq(req.GetName()), gomock.Any()).Return(&mockVol, nil).Times(1)
				fakeCs := NewControllerService(mockCloud)

				_, err = fakeCs.CreateVolume(ctx, req)
				if err != nil {
					srvErr, ok := status.FromError(err)
					if !ok {
						t.Fatalf("Could not get error status code from error: %v", srvErr)
					}
					t.Fatalf("Unexpected error: %v message %s", srvErr.Code(), srvErr.Message())
				}

				// Subsequent request should fail with AlreadyExists error
				mockCloud.EXPECT().GetVolumeByName(gomock.Eq(ctx), gomock.Eq(extraReq.GetName())).Return(&mockVol, nil)
				mockCloud.EXPECT().CreateVolume(gomock.Eq(ctx), FakeDiskOfferingID, "", gomock.Eq(extraReq.GetName()), gomock.Any()).Times(0)
				if _, err := fakeCs.CreateVolume(ctx, extraReq); err != nil {
					srvErr, ok := status.FromError(err)
					if !ok {
						t.Fatalf("Could not get error status code from error: %v", srvErr)
					}
					if srvErr.Code() != codes.AlreadyExists {
						t.Fatalf("Expected error code %d, got %d message %s", codes.AlreadyExists, srvErr.Code(), srvErr.Message())
					}
				} else {
					t.Fatalf("Expected error %v, got no error", codes.AlreadyExists)
				}
			},
		},
		{
			name: "success no capacity range",
			testFunc: func(t *testing.T) {
				t.Helper()
				req := &csi.CreateVolumeRequest{
					Name:               "my-volume",
					VolumeCapabilities: stdVolCap,
					Parameters:         stdParams,
				}
				expVol := &csi.Volume{
					VolumeId:      FakeVolID,
					CapacityBytes: util.GigaBytesToBytes(1),
					VolumeContext: stdParams,
				}

				ctx := context.Background()

				mockVol := cloud.Volume{
					ID:             FakeVolID,
					Name:           req.GetName(),
					DiskOfferingID: FakeDiskOfferingID,
					Size:           stdVolSize,
					ZoneID:         FakeZoneID,
				}

				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockCloud := cloud.NewMockCloud(mockCtl)
				mockCloud.EXPECT().GetVolumeByName(gomock.Eq(ctx), gomock.Eq(req.GetName())).Return(nil, cloud.ErrNotFound)
				mockCloud.EXPECT().CreateVolume(gomock.Eq(ctx), FakeDiskOfferingID, "", gomock.Eq(req.GetName()), gomock.Any()).Return(&mockVol, nil)
				fakeCs := NewControllerService(mockCloud)

				resp, err := fakeCs.CreateVolume(ctx, req)
				if err != nil {
					srvErr, ok := status.FromError(err)
					if !ok {
						t.Fatalf("Could not get error status code from error: %v", srvErr)
					}
					t.Fatalf("Unexpected error: %v message %s", srvErr.Code(), srvErr.Message())
				}

				vol := resp.GetVolume()
				if vol == nil {
					t.Fatalf("Expected volume, got nil")
				}
				if vol.GetVolumeId() != expVol.GetVolumeId() {
					t.Fatalf("Expected volume ID %s, got %s", expVol.GetVolumeId(), vol.GetVolumeId())
				}
				if vol.GetCapacityBytes() != expVol.GetCapacityBytes() {
					t.Fatalf("Expected volume capacity %d, got %d", expVol.GetCapacityBytes(), vol.GetCapacityBytes())
				}
			},
		},
		{
			name: "success with correct round up",
			testFunc: func(t *testing.T) {
				t.Helper()
				req := &csi.CreateVolumeRequest{
					Name:               "my-volume",
					VolumeCapabilities: stdVolCap,
					CapacityRange:      &csi.CapacityRange{RequiredBytes: 1073741825}, // 1 GB + 1 byte
					Parameters:         stdParams,
				}
				expVol := &csi.Volume{
					VolumeId:      req.GetName(),
					CapacityBytes: util.GigaBytesToBytes(2),
					VolumeContext: stdParams,
				}

				ctx := context.Background()

				mockVol := cloud.Volume{
					ID:             FakeVolID,
					Name:           req.GetName(),
					DiskOfferingID: FakeDiskOfferingID,
					Size:           util.GigaBytesToBytes(expVol.GetCapacityBytes()),
					ZoneID:         FakeZoneID,
				}

				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockCloud := cloud.NewMockCloud(mockCtl)
				mockCloud.EXPECT().GetVolumeByName(gomock.Eq(ctx), gomock.Eq(req.GetName())).Return(nil, cloud.ErrNotFound)
				mockCloud.EXPECT().CreateVolume(gomock.Eq(ctx), FakeDiskOfferingID, "", gomock.Eq(req.GetName()), gomock.Any()).Return(&mockVol, nil)
				fakeCs := NewControllerService(mockCloud)

				resp, err := fakeCs.CreateVolume(ctx, req)
				if err != nil {
					srvErr, ok := status.FromError(err)
					if !ok {
						t.Fatalf("Could not get error status code from error: %v", srvErr)
					}
					t.Fatalf("Unexpected error: %v message %s", srvErr.Code(), srvErr.Message())
				}

				vol := resp.GetVolume()
				if vol == nil {
					t.Fatalf("Expected volume, got nil")
				}
				if vol.GetCapacityBytes() != expVol.GetCapacityBytes() {
					t.Fatalf("Expected volume capacity %d, got %d", expVol.GetCapacityBytes(), vol.GetCapacityBytes())
				}
			},
		},
		{
			name: "success when volume exists and contains VolumeContext and AccessibleTopology",
			testFunc: func(t *testing.T) {
				t.Helper()
				req := &csi.CreateVolumeRequest{
					Name:               "my-volume",
					VolumeCapabilities: stdVolCap,
					CapacityRange:      stdCapRange,
					Parameters:         stdParams,
					AccessibilityRequirements: &csi.TopologyRequirement{
						Requisite: []*csi.Topology{
							{
								Segments: map[string]string{ZoneTopologyKey: FakeZoneID},
							},
						},
					},
				}
				extraReq := &csi.CreateVolumeRequest{
					Name:               "my-volume",
					VolumeCapabilities: stdVolCap,
					CapacityRange:      stdCapRange,
					Parameters:         stdParams,
					AccessibilityRequirements: &csi.TopologyRequirement{
						Requisite: []*csi.Topology{
							{
								Segments: map[string]string{ZoneTopologyKey: FakeZoneID},
							},
						},
					},
				}
				expVol := &csi.Volume{
					VolumeId:      FakeVolID,
					CapacityBytes: stdVolSize,
					VolumeContext: stdParams,
					AccessibleTopology: []*csi.Topology{
						{
							Segments: map[string]string{ZoneTopologyKey: FakeZoneID},
						},
					},
				}

				ctx := context.Background()

				mockVol := cloud.Volume{
					ID:             FakeVolID,
					Name:           req.GetName(),
					DiskOfferingID: FakeDiskOfferingID,
					Size:           stdVolSize,
					ZoneID:         FakeZoneID,
				}

				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockCloud := cloud.NewMockCloud(mockCtl)
				mockCloud.EXPECT().GetVolumeByName(gomock.Eq(ctx), gomock.Eq(req.GetName())).Return(&mockVol, nil)
				mockCloud.EXPECT().CreateVolume(gomock.Eq(ctx), FakeDiskOfferingID, "", gomock.Eq(req.GetName()), gomock.Any()).Times(0)
				fakeCs := NewControllerService(mockCloud)

				if _, err := fakeCs.CreateVolume(ctx, req); err != nil {
					srvErr, ok := status.FromError(err)
					if !ok {
						t.Fatalf("Could not get error status code from error: %v", srvErr)
					}
					t.Fatalf("Unexpected error: %v message %s", srvErr.Code(), srvErr.Message())
				}

				// Subsequent request returns the existing volume
				mockCloud.EXPECT().GetVolumeByName(gomock.Eq(ctx), gomock.Eq(extraReq.GetName())).Return(&mockVol, nil)
				mockCloud.EXPECT().CreateVolume(gomock.Eq(ctx), FakeDiskOfferingID, "", gomock.Eq(extraReq.GetName()), gomock.Any()).Times(0)
				resp, err := fakeCs.CreateVolume(ctx, extraReq)
				if err != nil {
					srvErr, ok := status.FromError(err)
					if !ok {
						t.Fatalf("Could not get error status code from error: %v", srvErr)
					}
					t.Fatalf("Unexpected error: %v message %s", srvErr.Code(), srvErr.Message())
				}

				vol := resp.GetVolume()
				if vol == nil {
					t.Fatalf("Expected volume, got nil")
				}

				for expKey, expVal := range expVol.GetVolumeContext() {
					ctx := vol.GetVolumeContext()
					if gotVal, ok := ctx[expKey]; !ok || gotVal != expVal {
						t.Fatalf("Expected volume context for key %v: %v, got: %v", expKey, expVal, gotVal)
					}
				}

				if expVol.GetAccessibleTopology() != nil {
					if !reflect.DeepEqual(expVol.GetAccessibleTopology(), vol.GetAccessibleTopology()) {
						t.Fatalf("Expected AccessibleTopology to be %+v, got: %+v", expVol.GetAccessibleTopology(), vol.GetAccessibleTopology())
					}
				}
			},
		},
		{
			name: "fail with in-flight operation",
			testFunc: func(t *testing.T) {
				t.Helper()
				req := &csi.CreateVolumeRequest{
					Name:               "my-volume",
					VolumeCapabilities: stdVolCap,
					CapacityRange:      stdCapRange,
					Parameters:         stdParams,
				}

				ctx := context.Background()

				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockCloud := cloud.NewMockCloud(mockCtl)

				volumeLocks := util.NewVolumeLocks()
				volumeLocks.TryAcquire(req.GetName())
				defer volumeLocks.Release(req.GetName())

				fakeCs := &ControllerService{
					connector:   mockCloud,
					volumeLocks: volumeLocks,
				}

				_, err := fakeCs.CreateVolume(ctx, req)
				if err == nil {
					t.Fatalf("Expected error, got no error")
				}
				srvErr, ok := status.FromError(err)
				if !ok {
					t.Fatalf("Could not get error status code from error: %v", srvErr)
				}
				if srvErr.Code() != codes.Aborted {
					t.Fatalf("Expected error code %d, got %d message %s", codes.Aborted, srvErr.Code(), srvErr.Message())
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, c.testFunc)
	}
}
