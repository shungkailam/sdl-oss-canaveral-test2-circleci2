package testhelper

// This is a helper file for tests
import (
	"cloudservices/common/base"
	cloudmgmtmodel "cloudservices/common/model"
	"cloudservices/tenantpool/core"
	"cloudservices/tenantpool/model"
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	TestProjectID   = base.GetUUID()
	DefaultDeadline = time.Second * 30
	DefaultInterval = time.Millisecond * 250
	TimedOutErr     = errors.New("Timed out")
)

// TestEdgeProvisioner  is the mock EdgeProvisioner
type TestEdgeProvisioner struct {
	edgeInfoMap           map[string]*model.EdgeInfo
	postDeleteEdgesCounts map[string]int
	mutex                 *sync.Mutex
}

// NewTestEdgeProvisioner creates the instance of TestEdgeProvisioner
func NewTestEdgeProvisioner() *TestEdgeProvisioner {
	return &TestEdgeProvisioner{edgeInfoMap: map[string]*model.EdgeInfo{}, postDeleteEdgesCounts: map[string]int{}, mutex: &sync.Mutex{}}
}

// CreateEdge creates an edge
func (edgeProvisioner *TestEdgeProvisioner) CreateEdge(ctx context.Context, config *model.CreateEdgeConfig) (*model.EdgeInfo, error) {
	var regID string
	edgeInfo := &model.EdgeInfo{ContextID: base.GetUUID(), State: core.Creating}
	for _, tag := range config.Tags {
		if strings.HasPrefix(tag, "registration=") {
			regID = strings.Split(tag, "=")[1]
			break
		}
	}
	mapKey := edgeProvisioner.getMapKey(regID, config.TenantID, edgeInfo.ContextID)
	edgeProvisioner.mutex.Lock()
	defer edgeProvisioner.mutex.Unlock()
	edgeProvisioner.edgeInfoMap[mapKey] = edgeInfo
	return edgeInfo, nil
}

// SetEdgeStatusByCount sets status of the edge(s) by count and registration ID
func (edgeProvisioner *TestEdgeProvisioner) SetEdgeStatusByCount(ctx context.Context, edgeCount int, regID, state string) {
	edgeProvisioner.mutex.Lock()
	defer edgeProvisioner.mutex.Unlock()
	idx := 0
	for key, edgeInfo := range edgeProvisioner.edgeInfoMap {
		if edgeCount > 0 && idx == edgeCount {
			break
		}
		if edgeInfo.State == core.Deleted {
			continue
		}
		thisRegID := edgeProvisioner.getRegistrationID(key)
		if thisRegID != regID {
			continue
		}
		if state == core.Created || state == core.Failed {
			if edgeInfo.State != core.Creating {
				continue
			}
		}
		edgeInfo.State = state
		if state == core.Created {
			edgeInfo.Edge = &cloudmgmtmodel.Edge{}
			edgeInfo.Edge.ID = edgeProvisioner.getEdgeID(key)
			edgeInfo.Resources = map[string]*model.Resource{}
			edgeInfo.Resources[TestProjectID] = &model.Resource{
				Type: model.ProjectResourceType,
				Name: "Test Project",
				ID:   TestProjectID,
			}
		}
		idx++
	}
}

// GetEdgeStatus returns the edge status
func (edgeProvisioner *TestEdgeProvisioner) GetEdgeStatus(ctx context.Context, tenantID, edgeContextID string) (*model.EdgeInfo, error) {
	edgeProvisioner.mutex.Lock()
	defer edgeProvisioner.mutex.Unlock()
	for key, edgeInfo := range edgeProvisioner.edgeInfoMap {
		if strings.HasSuffix(key, fmt.Sprintf("/%s/%s", tenantID, edgeContextID)) {
			if edgeInfo.State == core.Deleting {
				edgeInfo.State = core.Deleted
			}
			return edgeInfo, nil
		}
	}
	return nil, fmt.Errorf("Not found - tenant %s and edge context %s", tenantID, edgeContextID)
}

// GetEdgeCount returns the total number of edges for the registration ID
func (edgeProvisioner *TestEdgeProvisioner) GetEdgeCount(regID string) int {
	edgeProvisioner.mutex.Lock()
	defer edgeProvisioner.mutex.Unlock()
	count := 0
	for key, edgeInfo := range edgeProvisioner.edgeInfoMap {
		if edgeInfo.State == core.Deleted {
			continue
		}
		thisRegID := edgeProvisioner.getRegistrationID(key)
		if thisRegID == regID {
			count++
		}
	}
	return count
}

// DeleteEdge deletes an edge
func (edgeProvisioner *TestEdgeProvisioner) DeleteEdge(ctx context.Context, tenantID, edgeContextID string) (*model.EdgeInfo, error) {
	edgeProvisioner.mutex.Lock()
	defer edgeProvisioner.mutex.Unlock()
	for key, edgeInfo := range edgeProvisioner.edgeInfoMap {
		if edgeInfo.State == core.Deleted {
			continue
		}
		if strings.HasSuffix(key, fmt.Sprintf("/%s/%s", tenantID, edgeContextID)) {
			// Do not delete as other concurrent builds may assume it is deleted
			edgeInfo.State = core.Deleting
			return edgeInfo, nil
		}
	}
	return nil, fmt.Errorf("Not found - tenant %s and edge context %s", tenantID, edgeContextID)
}

// PostDeleteEdges does cleanup
func (edgeProvisioner *TestEdgeProvisioner) PostDeleteEdges(ctx context.Context, tenantID string) error {
	edgeProvisioner.mutex.Lock()
	defer edgeProvisioner.mutex.Unlock()
	for key := range edgeProvisioner.edgeInfoMap {
		if strings.HasSuffix(key, fmt.Sprintf("/%s/", tenantID)) {
			panic("PostDeleteEdges called before all edges are deleted")
		}
	}
	edgeProvisioner.postDeleteEdgesCounts[tenantID] = edgeProvisioner.postDeleteEdgesCounts[tenantID] + 1
	return nil
}

// DescribeEdge returns empty map
func (edgeProvisioner *TestEdgeProvisioner) DescribeEdge(ctx context.Context, tenantID, appID string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

// GetPostDeleteEdgesCount returns the count of PostDeleteEdgesCount calls
func (edgeProvisioner *TestEdgeProvisioner) GetPostDeleteEdgesCount(tenantID string) int {
	edgeProvisioner.mutex.Lock()
	defer edgeProvisioner.mutex.Unlock()
	return edgeProvisioner.postDeleteEdgesCounts[tenantID]
}

func (edgeProvisioner *TestEdgeProvisioner) getMapKey(regID, tenantID, edgeContextID string) string {
	return fmt.Sprintf("%s/%s/%s", regID, tenantID, edgeContextID)
}

func (edgeProvisioner *TestEdgeProvisioner) getRegistrationID(mapKey string) string {
	return strings.Split(mapKey, "/")[0]
}

func (edgeProvisioner *TestEdgeProvisioner) getTenantID(mapKey string) string {
	return strings.Split(mapKey, "/")[1]
}

func (edgeProvisioner *TestEdgeProvisioner) getEdgeID(mapKey string) string {
	return strings.Split(mapKey, "/")[2]
}

// WaitForState waits for the numbers of rows matching the reqID and states to be within the minCount and maxCount
func WaitForState(t *testing.T, bookKeeper *core.BookKeeper, regID string, states []string, minCount int, maxCount int) {
	idx := 0
	err := DoWithDeadline(DefaultDeadline, DefaultInterval, func() (bool, error) {
		idx = 0
		pageResponse, err := bookKeeper.ScanTenantClaims(context.Background(), regID, "", states, nil, func(registration *model.Registration, tenantClaim *model.TenantClaim) error {
			idx++
			return nil
		})
		if err != nil {
			return true, err
		}
		if pageResponse.TotalCount != idx {
			return true, fmt.Errorf("Unmatched total count %d, %d", pageResponse.TotalCount, idx)
		}
		if idx > maxCount {
			return false, nil
		}
		if idx < minCount {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Logf("Final count for registration %s and states %+v is %d", regID, states, idx)
		debug.PrintStack()
		t.Fatal(err)
	}
}

// DoWithDeadline calls the callback until it returns exit to true or the retry times out
func DoWithDeadline(deadline time.Duration, interval time.Duration, callback func() (bool, error)) error {
	start := time.Now()
	for {
		exit, err := callback()
		if exit {
			return err
		}
		elapsed := time.Since(start)
		if elapsed > deadline {
			return TimedOutErr
		} else {
			time.Sleep(interval)
		}
	}
}
