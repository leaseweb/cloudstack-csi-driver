//go:build sanity

package sanity

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubernetes-csi/csi-test/v5/pkg/sanity"
	"k8s.io/klog/v2"

	"github.com/leaseweb/cloudstack-csi-driver/pkg/cloud/fake"
	"github.com/leaseweb/cloudstack-csi-driver/pkg/driver"
	"github.com/leaseweb/cloudstack-csi-driver/pkg/mount"
)

func TestSanity(t *testing.T) {
	// Setup driver
	dir, err := ioutil.TempDir("", "sanity-cloudstack-csi")
	if err != nil {
		t.Fatalf("error creating directory: %v", err)
	}
	defer os.RemoveAll(dir)

	targetPath := filepath.Join(dir, "target")
	stagingPath := filepath.Join(dir, "staging")
	endpoint := "unix://" + filepath.Join(dir, "csi.sock")

	config := sanity.NewTestConfig()
	config.TargetPath = targetPath
	config.StagingPath = stagingPath
	config.Address = endpoint
	config.TestVolumeParameters = map[string]string{
		driver.DiskOfferingKey: "9743fd77-0f5d-4ef9-b2f8-f194235c769c",
	}
	config.IdempotentCount = 5
	config.TestNodeVolumeAttachLimit = true

	logger := klog.Background()
	ctx := klog.NewContext(context.Background(), logger)

	options := driver.Options{
		Mode:              driver.AllMode,
		Endpoint:          endpoint,
		NodeName:          "node",
		VolumeAttachLimit: 16,
	}
	csiDriver, err := driver.NewDriver(ctx, fake.New(), &options, mount.NewFake())
	if err != nil {
		t.Fatalf("error creating driver: %v", err)
	}
	go func() {
		csiDriver.Run(ctx)
	}()

	sanity.Test(t, config)
}
