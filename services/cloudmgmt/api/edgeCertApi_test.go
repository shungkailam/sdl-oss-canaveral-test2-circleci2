package api_test

import (
	"cloudservices/common/model"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func generateEdgeCert(edgeCertId string, tenantID string, edgeID string) model.EdgeCert {
	return model.EdgeCert{
		EdgeBaseModel: model.EdgeBaseModel{
			BaseModel: model.BaseModel{
				ID:       edgeCertId,
				TenantID: tenantID,
				Version:  0,
			},
			EdgeID: edgeID,
		},
		EdgeCertCore: model.EdgeCertCore{
			Certificate:       "edge-cert",
			PrivateKey:        "edge-key",
			ClientCertificate: "client-cert",
			ClientPrivateKey:  "client-key",
			EdgeCertificate:   "edge-cert-new",
			EdgePrivateKey:    "edge-key-new",
			Locked:            true,
		},
	}
}

func TestEdgeCert(t *testing.T) {
	t.Parallel()
	t.Log("running TestEdgeCert test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	project := createEmptyCategoryProject(t, dbAPI, tenantID)
	projectID := project.ID
	ctx, ctx2, ctx3 := makeContext(tenantID, []string{projectID})
	edge := createEdge(t, dbAPI, tenantID)
	edgeID := edge.ID
	edge2 := createEdge(t, dbAPI, tenantID)
	edge2ID := edge2.ID
	edgeCtx := makeEdgeContext(tenantID, edgeID, nil)
	edge2Ctx := makeEdgeContext(tenantID, edge2ID, nil)

	// Teardown
	defer func() {
		// dbAPI.DeleteApplication(ctx, appID, nil)
		dbAPI.DeleteEdge(ctx, edgeID, nil)
		dbAPI.DeleteEdge(ctx, edge2ID, nil)
		dbAPI.DeleteProject(ctx, projectID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete EdgeCert", func(t *testing.T) {
		t.Log("running Create/Get/Delete EdgeCert test")

		// Certificate is already there as a part of edge creation
		edgeCert, err := dbAPI.GetEdgeCertByEdgeID(ctx, edgeID)
		require.NoError(t, err)
		t.Logf("get edge cert by edge ID before update successful, %+v", edgeCert)

		edgeCertId := edgeCert.ID

		// update edge cert
		doc := generateEdgeCert(edgeCertId, tenantID, edgeID)
		upResp, err := dbAPI.UpdateEdgeCert(ctx, &doc, nil)
		require.NoError(t, err)
		t.Logf("update edge cert successful, %+v", upResp)

		// get edge cert
		edgeCert, err = dbAPI.GetEdgeCert(ctx, edgeCertId)
		require.NoError(t, err)
		t.Logf("get edge cert successful, %+v", edgeCert)

		if edgeCert.ID != edgeCertId || edgeCert.TenantID != tenantID || edgeCert.EdgeID != edgeID || edgeCert.Locked != true {
			t.Fatal("edgeCert data mismatch")
		}

		// test SetEdgeCertLock
		ba := []bool{false, true}
		for _, b := range ba {
			err = dbAPI.SetEdgeCertLock(ctx, edgeCert.EdgeID, b)
			require.NoError(t, err)
			edgeCert, err = dbAPI.GetEdgeCert(ctx, edgeCertId)
			require.NoError(t, err)
			if edgeCert.Locked != b {
				t.Fatal("edgeCert data mismatch")
			}
		}
		// ctx2 and ctx3 should not be allowed to unlock edge cert
		err = dbAPI.SetEdgeCertLock(ctx2, edgeCert.EdgeID, false)
		require.Error(t, err, "unlock cert should be forbidden for non infra admin")
		err = dbAPI.SetEdgeCertLock(ctx3, edgeCert.EdgeID, false)
		require.Error(t, err, "unlock cert should be forbidden for non infra admin")
		// edgeCtx can unlock own edge cert
		err = dbAPI.SetEdgeCertLock(edgeCtx, edgeCert.EdgeID, false)
		require.NoError(t, err)
		// edge2Ctx can't unlock other edge cert
		err = dbAPI.SetEdgeCertLock(edge2Ctx, edgeCert.EdgeID, false)
		require.Error(t, err, "unlock edge cert from another edge should be forbidden")

		// test SetEdgeCertLockW with auto lock
		nSecs := 3
		obj := model.EdgeCertLockParam{
			EdgeClusterID:   edgeCert.EdgeID,
			Locked:          false,
			DurationSeconds: nSecs,
		}
		r, err := objToReader(obj)
		require.NoError(t, err)
		err = dbAPI.SetEdgeCertLockW(ctx, nil, r, nil)
		require.NoError(t, err)
		edgeCert, err = dbAPI.GetEdgeCert(ctx, edgeCertId)
		require.NoError(t, err)
		if edgeCert.Locked != false {
			t.Fatal("edgeCert data mismatch")
		}
		// wait nSecs + 1 seconds
		time.Sleep(time.Duration(nSecs+1) * time.Second)
		edgeCert, err = dbAPI.GetEdgeCert(ctx, edgeCertId)
		require.NoError(t, err)
		if edgeCert.Locked != true {
			t.Fatal("edgeCert data mismatch")
		}

		// delete edge cert
		delResp, err := dbAPI.DeleteEdgeCert(ctx, edgeCertId, nil)
		require.NoError(t, err)
		t.Logf("delete edge cert successful, %v", delResp)

	})

	// select all edge certs
	t.Run("SelectAllEdgeCerts", func(t *testing.T) {
		t.Log("running SelectAllEdgeCerts test")
		edgeCerts, err := dbAPI.SelectAllEdgeCerts(ctx)
		require.NoError(t, err)
		for _, edgeCert := range edgeCerts {
			testForMarshallability(t, edgeCert)
		}
	})

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		doc := generateEdgeCert(id, tenantID, edgeID)
		doc.ID = id
		return dbAPI.CreateEdgeCert(ctx, &doc, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetEdgeCert(ctx, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteEdgeCert(ctx, id, nil)
	}))
}
