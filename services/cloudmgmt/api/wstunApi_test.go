package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/model"
	"cloudservices/common/utils"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func setupSSHTunneling(t *testing.T, dbAPI api.ObjectModelAPI, ctx1 context.Context, serviceDomainID string) (model.WstunPayload, error) {
	resp := model.WstunPayload{}
	doc := model.WstunRequest{
		ServiceDomainID: serviceDomainID,
	}

	r, err := objToReader(doc)
	require.NoError(t, err)

	// setup ssh tunneling
	var w bytes.Buffer
	err = dbAPI.SetupSSHTunnelingW(ctx1, &w, r, func(ctx context.Context, x interface{}) error {
		t.Logf("callback { x=%+v", x)
		return nil
	})
	if err != nil {
		return resp, err
	}

	err = json.NewDecoder(&w).Decode(&resp)
	if err != nil {
		return resp, err
	}
	if resp.ServiceDomainID != serviceDomainID {
		t.Fatalf("edge id mismatch %s != %s", resp.ServiceDomainID, serviceDomainID)
	}
	if resp.Port < utils.WstunStartPort || resp.Port > utils.WstunEndPort {
		t.Fatalf("port out of range: %d not in %d - %d", resp.Port, utils.WstunStartPort, utils.WstunEndPort)
	}
	t.Logf("setup ssh tenneling successful, %d - %d", resp.Port, resp.Expiration)
	return resp, nil
}

func teardownSSHTunneling(t *testing.T, dbAPI api.ObjectModelAPI, ctx1 context.Context, serviceDomainID string) error {
	doc := model.WstunTeardownRequest{
		ServiceDomainID: serviceDomainID,
		PublicKey:       "",
	}

	r, err := objToReader(doc)
	require.NoError(t, err)

	// teardown ssh tunneling
	var w bytes.Buffer
	err = dbAPI.TeardownSSHTunnelingW(ctx1, &w, r, func(ctx context.Context, x interface{}) error {
		t.Logf("callback { x=%+v", x)
		return nil
	})
	if err != nil {
		return err
	}

	t.Log("teardown ssh tenneling successful")
	return nil
}

// Note: to get this test to work one must set the env var DISABLE_K8S=1
func TestWstun(t *testing.T) {
	t.Parallel()
	t.Log("running TestWstun test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID

	edge := createEdge(t, dbAPI, tenantID)
	edgeID := edge.ID

	edge2 := createEdge(t, dbAPI, tenantID)
	edgeID2 := edge2.ID

	edge3 := createEdge(t, dbAPI, tenantID)
	edgeID3 := edge3.ID

	user := createUser(t, dbAPI, tenantID)
	userId := user.ID
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{userId}, nil)
	projectID := project.ID
	ctx1, ctx2, ctx3 := makeContext(tenantID, []string{projectID})

	// // Teardown
	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteUser(ctx1, userId, nil)
		dbAPI.DeleteEdge(ctx1, edgeID3, nil)
		dbAPI.DeleteEdge(ctx1, edgeID2, nil)
		dbAPI.DeleteEdge(ctx1, edgeID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Setup / Teardown SSHTunnelingW", func(t *testing.T) {
		t.Log("running Setup / Teardown SSHTunnelingW test")

		// Testing ctx1 - infra admin with project access
		r1, err := setupSSHTunneling(t, dbAPI, ctx1, edgeID)
		require.NoError(t, err)
		r2, err := setupSSHTunneling(t, dbAPI, ctx1, edgeID2)
		require.NoError(t, err)
		r3, err := setupSSHTunneling(t, dbAPI, ctx1, edgeID3)
		require.NoError(t, err)
		r1a, err := setupSSHTunneling(t, dbAPI, ctx1, edgeID)
		require.NoError(t, err)
		r2a, err := setupSSHTunneling(t, dbAPI, ctx1, edgeID2)
		require.NoError(t, err)
		r3a, err := setupSSHTunneling(t, dbAPI, ctx1, edgeID3)
		require.NoError(t, err)
		// we now update expiration on each setup, so response will not
		// have the same expiration
		r1a.Expiration = r1.Expiration
		r2a.Expiration = r2.Expiration
		r3a.Expiration = r3.Expiration

		if !reflect.DeepEqual(r1, r1a) {
			t.Fatalf("setup tunnel response not the same: %+v vs %+v", r1, r1a)
		}
		if !reflect.DeepEqual(r2, r2a) {
			t.Fatalf("setup tunnel response not the same: %+v vs %+v", r2, r2a)
		}
		if !reflect.DeepEqual(r3, r3a) {
			t.Fatalf("setup tunnel response not the same: %+v vs %+v", r3, r3a)
		}

		err = teardownSSHTunneling(t, dbAPI, ctx1, edgeID)
		require.NoError(t, err)
		err = teardownSSHTunneling(t, dbAPI, ctx1, edgeID2)
		require.NoError(t, err)
		err = teardownSSHTunneling(t, dbAPI, ctx1, edgeID3)
		require.NoError(t, err)

		// Testing ctx2 - non admin and no project access
		_, err = setupSSHTunneling(t, dbAPI, ctx2, edgeID)
		require.Errorf(t, err, "expected setup ssh for ctx2 to fail for edge %s", edgeID)
		_, err = setupSSHTunneling(t, dbAPI, ctx2, edgeID2)
		require.Errorf(t, err, "expected setup ssh for ctx2 to fail for edge %s", edgeID2)
		_, err = setupSSHTunneling(t, dbAPI, ctx2, edgeID3)
		require.Errorf(t, err, "expected setup ssh for ctx2 to fail for edge %s", edgeID3)
		err = teardownSSHTunneling(t, dbAPI, ctx2, edgeID)
		require.Errorf(t, err, "expected teardown ssh for ctx2 to fail for edge %s", edgeID)
		err = teardownSSHTunneling(t, dbAPI, ctx2, edgeID2)
		require.Errorf(t, err, "expected teardown ssh for ctx2 to fail for edge %s", edgeID2)
		err = teardownSSHTunneling(t, dbAPI, ctx2, edgeID3)
		require.Errorf(t, err, "expected teardown ssh for ctx2 to fail for edge %s", edgeID3)

		// Testing ctx3 - non admin with project access
		_, err = setupSSHTunneling(t, dbAPI, ctx3, edgeID)
		require.Errorf(t, err, "expected setup ssh for ctx3 to fail for edge %s", edgeID)
		_, err = setupSSHTunneling(t, dbAPI, ctx3, edgeID2)
		require.Errorf(t, err, "expected setup ssh for ctx3 to fail for edge %s", edgeID2)
		_, err = setupSSHTunneling(t, dbAPI, ctx3, edgeID3)
		require.Errorf(t, err, "expected setup ssh for ctx3 to fail for edge %s", edgeID3)
		err = teardownSSHTunneling(t, dbAPI, ctx3, edgeID)
		require.Errorf(t, err, "expected teardown ssh for ctx3 to fail for edge %s", edgeID)
		err = teardownSSHTunneling(t, dbAPI, ctx3, edgeID2)
		require.Errorf(t, err, "expected teardown ssh for ctx3 to fail for edge %s", edgeID2)
		err = teardownSSHTunneling(t, dbAPI, ctx3, edgeID3)
		require.Errorf(t, err, "expected teardown ssh for ctx3 to fail for edge %s", edgeID3)
	})
}
