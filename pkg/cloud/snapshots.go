package cloud

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	snapshotReadyStatus = "BackedUp"
	snapReadyDuration   = 1 * time.Second
	snapReadyFactor     = 1.2
	snapReadySteps      = 10
	asyncBackup         = true
)

// CreateSnapshot issues a request to take a Snapshot of the specified Volume with the corresponding ID and
// returns the resultant CloudStack Snapshot Item upon success
func (c *client) CreateSnapshot(ctx context.Context, name, volumeID string) (string, error) {
	// Input validation
	if name == "" || volumeID == "" {
		return "", status.Errorf(codes.Internal, "Snapshotname %s; requested for volume %s", name, volumeID)
	}

	// Pre-API Call Checks
	// Retrieve volume information to ensure it exists and is in a valid state
	volume, _, err := c.Volume.GetVolumeByID(volumeID)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve volume '%s': %v", volumeID, err)
	}

	// Check if the volume is in a 'Ready' state for snapshot creation
	if volume.State != "Ready" {
		return "", fmt.Errorf("volume '%s' is not in a 'Ready' state for snapshot creation", volumeID)
	}

	// Preparing snapshot creation parameters
	p := c.Snapshot.NewCreateSnapshotParams(volumeID)
	p.SetName(name)
	p.SetVolumeid(volumeID)
	p.SetAsyncbackup(asyncBackup)
	ctxzap.Extract(ctx).Sugar().Infow("CloudStack API call", "command", "CreateSnapshot", "params", map[string]string{
		"name":        name,
		"volumeid":    volumeID,
		"asyncbackup": strconv.FormatBool(asyncBackup),
	})
	snapshot, err := c.Snapshot.CreateSnapshot(p)
	if err != nil {
		return "", fmt.Errorf("failed to create snapshot '%s' for volume '%s': %v", name, volumeID, err)
	}

	return snapshot.Id, nil
}

// ListSnapshots retrieves a list of active snapshots from CloudStack for the corresponding Domain.  We also
// provide the ability to provide limit to enable the consumer to provide accurate pagination.
// In addition the filters argument provides a mechanism for passing in valid filter strings to the list
// operation.  Valid filter keys are:  Name, VolumeID, Limit, Marker (DomainID has no effect)

func (c *client) ListSnapshots(ctx context.Context, filters map[string]string) ([]Snapshot, string, error) {
	var snapshotList []Snapshot
	var nextPageToken string

	// Initialize parameters for CloudStack API call
	p := c.Snapshot.NewListSnapshotsParams()

	// Declare pointer variables for pageSize and currentPage
	pageSize := 20   // Default pageSize
	currentPage := 1 // Default to the first page
	var err error

	// Apply filters and pagination parameters
	for key, value := range filters {
		switch key {
		case "Name":
			p.SetName(value)
		case "VolumeID":
			p.SetVolumeid(value)
		case "Marker":
			// Try to parse the page number from the filters
			currentPage, err = strconv.Atoi(value)
			if err != nil {
				fmt.Printf("Invalid format for Marker: %s, using default page\n", value)
				currentPage = 1 // Reassign to default if parsing fails
			}
			p.SetPage(currentPage)
		case "Limit":
			// Try to parse the page size from the filters
			pageSize, err = strconv.Atoi(value)
			if err != nil {
				fmt.Printf("Invalid or unsupported format for Limit: %s, using default limit\n", value)
				pageSize = 20 // Reassign to default if parsing fails or unsupported value
			}
			p.SetPagesize(pageSize)
		default:
			fmt.Printf("Not a valid filter key %s\n", key)
		}
	}

	// Log the final pagination settings
	fmt.Printf("Fetching snapshots with Page: %d, PageSize: %d\n", currentPage, pageSize)

	// Call CloudStack API to list snapshots
	resp, err := c.Snapshot.ListSnapshots(p)
	if err != nil {
		return nil, "", fmt.Errorf("error listing snapshots from CloudStack: %w", err)
	}

	// Convert the response to your []Snapshot type
	for _, apiSnapshot := range resp.Snapshots {
		snapshot := convertAPISnapshotToSnapshot(apiSnapshot)
		snapshotList = append(snapshotList, snapshot)
	}

	// Determine the nextPageToken
	if morePagesExist(resp, pageSize, currentPage) {
		// Set nextPageToken to the next page
		nextPageToken = strconv.Itoa(currentPage + 1)
		if nextPageToken != "" {
			fmt.Println("The nextPageToken is set, more than one page is available.")

		}

	}

	return snapshotList, nextPageToken, nil
}

// DeleteSnapshot issues a request to delete the Snapshot with the specified ID from the backend
func (c *client) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	p := c.Snapshot.NewDeleteSnapshotParams(snapshotID)
	ctxzap.Extract(ctx).Sugar().Infow("CloudStack API call", "command", "DeleteSnapshot", "params", map[string]string{
		"id": snapshotID,
	})
	_, err := c.Snapshot.DeleteSnapshot(p)
	if err != nil && strings.Contains(err.Error(), "4350") {
		// CloudStack error InvalidParameterValueException
		return ErrNotFound
	}
	return err
}

// GetSnapshotByID returns snapshot details by id
func (c *client) GetSnapshotByID(ctx context.Context, snapshotID string) (*Snapshot, error) {
	p := c.Snapshot.NewListSnapshotsParams()
	p.SetId(snapshotID)
	ctxzap.Extract(ctx).Sugar().Infow("CloudStack API call", "command", "ListSnapshots", "params", map[string]string{
		"id": snapshotID,
	})
	l, err := c.Snapshot.ListSnapshots(p)
	if err != nil {
		return nil, err
	}
	if l.Count == 0 {
		return nil, ErrNotFound
	}
	if l.Count > 1 {
		return nil, ErrTooManyResults
	}
	snap := l.Snapshots[0]

	v := Snapshot{
		ID:          snap.Id,
		Name:        snap.Name,
		VirtualSize: int(snap.Virtualsize),
		Created:     snap.Created,
		ZoneID:      snap.Zoneid,
		VolumeID:    snap.Volumeid,
		State:       snap.State,
	}
	return &v, nil
}

// GetSnapshotByName returns snapshot details by name
func (c *client) GetSnapshotByName(ctx context.Context, name string) (*Snapshot, error) {
	p := c.Snapshot.NewListSnapshotsParams()
	p.SetName(name)
	ctxzap.Extract(ctx).Sugar().Infow("CloudStack API call", "command", "ListSnapshots", "params", map[string]string{
		"name": name,
	})
	l, err := c.Snapshot.ListSnapshots(p)
	if err != nil {
		return nil, err
	}
	if l.Count == 0 {
		return nil, ErrNotFound
	}
	if l.Count > 1 {
		return nil, ErrTooManyResults
	}
	snap := l.Snapshots[0]

	v := Snapshot{
		ID:          snap.Id,
		Name:        snap.Name,
		VirtualSize: int(snap.Virtualsize),
		Created:     snap.Created,
		ZoneID:      snap.Zoneid,
		VolumeID:    snap.Volumeid,
		State:       snap.State,
	}
	return &v, nil
}

// WaitSnapshotReady waits till snapshot is ready
func (c *client) WaitSnapshotReady(ctx context.Context, snapshotID string) error {
	backoff := wait.Backoff{
		Duration: snapReadyDuration,
		Factor:   snapReadyFactor,
		Steps:    snapReadySteps,
	}

	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		ready, err := c.snapshotIsReady(ctx, snapshotID)
		if err != nil {
			return false, err
		}
		return ready, nil
	})

	if wait.Interrupted(err) {
		err = fmt.Errorf("timeout, snapshot  %s is still not Ready %v", snapshotID, err.Error())
	}

	return err
}

func (c *client) snapshotIsReady(ctx context.Context, snapshotID string) (bool, error) {
	snap, err := c.GetSnapshotByID(ctx, snapshotID)
	if err != nil {
		return false, fmt.Errorf("Snapshot is not ready: %w", err)
	}

	return snap.State == snapshotReadyStatus, nil
}

// convertAPISnapshotToSnapshot converts an API snapshot object to your application's Snapshot type.
func convertAPISnapshotToSnapshot(apiSnapshot *cloudstack.Snapshot) Snapshot {
	// Conversion logic here
	return Snapshot{
		ID:          apiSnapshot.Id,
		Created:     apiSnapshot.Created,
		Name:        apiSnapshot.Name,
		VolumeID:    apiSnapshot.Volumeid,
		ZoneID:      apiSnapshot.Zoneid,
		State:       apiSnapshot.State,
		VirtualSize: int(apiSnapshot.Virtualsize),
	}
}

// morePagesExist determines if there are more pages of results based on the API response.
func morePagesExist(resp *cloudstack.ListSnapshotsResponse, pageSize int, currentPage int) bool {
	if resp == nil || pageSize <= 0 {
		return false
	}

	totalSnapshots := resp.Count
	totalPages := totalSnapshots / pageSize
	// Check for any remaining snapshots not fitting in a full page
	if totalSnapshots%pageSize != 0 {
		totalPages++
	}
	fmt.Printf("Total Page is %d", totalPages)

	return currentPage < totalPages
}
