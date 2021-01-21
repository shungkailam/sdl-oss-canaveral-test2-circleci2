package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"github.com/go-openapi/errors"
	"github.com/stretchr/testify/require"
	"github.com/thoas/go-funk"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func createCloudCreds(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string) model.CloudCreds {
	// create cloud creds
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	cloudCredsName := "aws-cloud-creds-name-" + base.GetUUID()
	cloudCredsDesc := "aws-cloud-creds-desc"

	// CloudCreds object, leave ID blank and let create generate it
	cc := model.CloudCreds{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		Name:        cloudCredsName,
		Type:        "AWS",
		Description: cloudCredsDesc,
		AWSCredential: &model.AWSCredential{
			AccessKey: "foo",
			Secret:    "bar",
		},
		GCPCredential: nil,
	}
	// create CloudCreds
	resp, err := dbAPI.CreateCloudCreds(ctx, &cc, nil)
	require.NoError(t, err)
	t.Logf("create CloudCreds successful, %s", resp)

	cc.ID = resp.(model.CreateDocumentResponse).ID
	cloudCreds, err := dbAPI.GetCloudCreds(ctx, cc.ID)
	require.NoError(t, err)
	return cloudCreds
}

func TestCloudCreds(t *testing.T) {
	t.Parallel()
	t.Log("running TestCloudCreds test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	edge := createEdge(t, dbAPI, tenantID)
	edgeID := edge.ID
	edge2 := createEdge(t, dbAPI, tenantID)
	edgeID2 := edge2.ID
	cc := createCloudCreds(t, dbAPI, tenantID)
	cloudCredsID := cc.ID
	dp := createAWSDockerProfile(t, dbAPI, tenantID, cloudCredsID)
	dockerProfileId := dp.ID
	project := createExplicitProjectCommon(t, dbAPI, tenantID, []string{cloudCredsID}, []string{dockerProfileId}, []string{}, []string{edgeID})
	projectID := project.ID
	project2 := createExplicitProjectCommon(t, dbAPI, tenantID, []string{cloudCredsID}, []string{dockerProfileId}, []string{}, []string{edgeID2})
	projectID2 := project2.ID
	ctx1, ctx2, ctx3 := makeContext(tenantID, []string{projectID})

	// Teardown
	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteProject(ctx1, projectID2, nil)
		dbAPI.DeleteCloudCreds(ctx1, cloudCredsID, nil)
		dbAPI.DeleteDockerProfile(ctx1, dockerProfileId, nil)
		dbAPI.DeleteEdge(ctx1, edgeID, nil)
		dbAPI.DeleteEdge(ctx1, edgeID2, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete CloudCreds", func(t *testing.T) {
		t.Log("running Create/Get/Delete CloudCreds test")

		cloudCredsDesc := "aws-cloud-creds-desc"
		cloudCredsNameUpdated := "aws-cloud-creds-name-updated"

		// get CloudCreds
		cloudCreds, err := dbAPI.GetCloudCreds(ctx1, cloudCredsID)
		require.NoError(t, err)
		t.Logf("get CloudCreds before update successful, %+v", cloudCreds)

		// update CloudCreds
		cc.Name = cloudCredsNameUpdated
		upResp, err := dbAPI.UpdateCloudCreds(ctx1, &cc, nil)
		require.NoError(t, err)
		t.Logf("update CloudCreds successful, %+v", upResp)

		// test SelectAllCloudCreds
		cloudCredss, err := dbAPI.SelectAllCloudCreds(ctx1, nil)
		require.NoError(t, err)
		require.Len(t, cloudCredss, 1)

		cloudCredss, err = dbAPI.SelectAllCloudCreds(ctx2, nil)
		require.NoError(t, err)
		require.Len(t, cloudCredss, 0)

		cloudCreds, err = dbAPI.GetCloudCreds(ctx2, cloudCredsID)
		require.Error(t, err, "Expected not found error")

		cloudCredss, err = dbAPI.SelectAllCloudCreds(ctx3, nil)
		require.NoError(t, err)
		require.Len(t, cloudCredss, 1)

		// test SelectAllCloudCredsForProject
		authContext1 := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		newCtx := context.WithValue(context.Background(), base.AuthContextKey, authContext1)
		cloudCredss, err = dbAPI.SelectAllCloudCredsForProject(newCtx, projectID, nil)
		require.Error(t, err, "expect select all cloud profiles to fail for auth 1")

		cloudCredss, err = dbAPI.SelectAllCloudCredsForProject(ctx2, projectID, nil)
		require.Error(t, err, "expect select all cloud profiles to fail for auth 2")

		cloudCredss, err = dbAPI.SelectAllCloudCredsForProject(ctx3, projectID, nil)
		require.NoError(t, err)
		require.Len(t, cloudCredss, 1)

		// select all vs select all W
		var w bytes.Buffer
		ccs1, err := dbAPI.SelectAllCloudCreds(ctx1, nil)
		require.NoError(t, err)
		ccs2 := &[]model.CloudCreds{}
		err = selectAllConverter(ctx1, dbAPI.SelectAllCloudCredsW, ccs2, &w)
		require.NoError(t, err)
		sort.Sort(model.CloudCredssByID(ccs1))
		// SelectW is masked, Select is not, so mask ccs1 here
		model.MaskCloudCreds(ccs1)
		sort.Sort(model.CloudCredssByID(*ccs2))
		if !reflect.DeepEqual(&ccs1, ccs2) {
			t.Fatalf("expect select cloud creds and select cloud creds w results to be equal %+v vs %+v", ccs1, *ccs2)
		}

		// test GetCloudCreds
		// get CloudCreds
		cloudCreds, err = dbAPI.GetCloudCreds(ctx1, cloudCredsID)
		require.NoError(t, err)
		t.Logf("get CloudCreds successful, %+v", cloudCreds)

		if cloudCreds.ID != cloudCredsID || cloudCreds.Name != cloudCredsNameUpdated || cloudCreds.Description != cloudCredsDesc {
			t.Fatal("CloudCreds data mismatch")
		}
		cloudCreds, err = dbAPI.GetCloudCreds(ctx2, cloudCredsID)
		require.Error(t, err, "Expected not found error")

		cloudCreds, err = dbAPI.GetCloudCreds(ctx3, cloudCredsID)
		require.NoError(t, err, "Unexpected GetCloudCreds error")

		edgeIDs, err := dbAPI.GetAllCloudCredsEdges(ctx1, cloudCredsID)
		require.NoError(t, err, "Unexpected GetAllCloudCredsEdges error")
		require.Len(t, edgeIDs, 2)

		sort.Strings(edgeIDs)
		edgeIDs2 := []string{edgeID, edgeID2}
		sort.Strings(edgeIDs2)
		require.Equal(t, edgeIDs, edgeIDs2)
	})

	// select all CloudCreds
	t.Run("SelectAllCloudCreds", func(t *testing.T) {
		t.Log("running SelectAllCloudCreds test")
		cloudCreds, err := dbAPI.SelectAllCloudCreds(ctx1, nil)
		require.NoError(t, err)
		for _, cloudCred := range cloudCreds {
			testForMarshallability(t, cloudCred)
		}
	})

	t.Run("CloudCredsConversion", func(t *testing.T) {
		t.Log("running CloudCredsConversion test")
		var awsCredential = model.AWSCredential{
			AccessKey: "foo",
			Secret:    "bar",
		}
		var gcpCredential = model.GCPCredential{
			Type:                    "type-val",
			ProjectID:               "project-id-val",
			PrivateKeyID:            "private-key-id-val",
			PrivateKey:              "private-key-val",
			ClientEmail:             "client-email-val",
			ClientID:                "client-id-val",
			AuthURI:                 "auth-uri-val",
			TokenURI:                "token-uri-val",
			AuthProviderX509CertURL: "auth-provider-x509-cert-url-val",
			ClientX509CertURL:       "client-x509-cert-url",
		}
		now, _ := time.Parse(time.RFC3339, "2018-01-01T01:01:01Z")
		cloudCredsList := []model.CloudCreds{
			{
				BaseModel: model.BaseModel{
					ID:        "aws-cloud-creds-id",
					TenantID:  tenantID,
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:          "aws-cloud-creds-name",
				Type:          "AWS",
				Description:   "aws-cloud-creds-desc",
				AWSCredential: &awsCredential,
				GCPCredential: nil,
			},
			{
				BaseModel: model.BaseModel{
					ID:        "gcp-cloud-creds-id",
					TenantID:  tenantID,
					Version:   0,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:          "gcp-cloud-creds-name",
				Type:          "GCP",
				Description:   "aws-cloud-creds-desc",
				AWSCredential: nil,
				GCPCredential: &gcpCredential,
			},
		}
		for _, app := range cloudCredsList {
			appDBO := api.CloudCredsDBO{}
			app2 := model.CloudCreds{}
			err := base.Convert(&app, &appDBO)
			require.NoError(t, err)
			err = base.Convert(&appDBO, &app2)
			require.NoError(t, err)
			if !reflect.DeepEqual(app, app2) {
				t.Fatalf("deep equal failed: %+v vs. %+v", app, app2)
			}
		}
	})

	t.Run("Bad input", func(t *testing.T) {
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		cloudCredsName := "aws-cloud-creds-name-" + base.GetUUID()
		cloudCredsDesc := "aws-cloud-creds-desc"

		// CloudCreds object, leave ID blank and let create generate it
		cc := model.CloudCreds{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			Name:        cloudCredsName,
			Type:        "GCP",
			Description: cloudCredsDesc,
			AWSCredential: &model.AWSCredential{
				AccessKey: "foo",
				Secret:    "bar",
			},
			GCPCredential: nil,
		}
		// create CloudCreds
		_, err := dbAPI.CreateCloudCreds(ctx, &cc, nil)
		require.Error(t, err, "we can create a malformed cloud credentials")
	})

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		return dbAPI.CreateCloudCreds(ctx1, &model.CloudCreds{
			BaseModel: model.BaseModel{
				ID:       id,
				TenantID: tenantID,
				Version:  5,
			},
			Name:        "cloud-test-" + funk.RandomString(10),
			Type:        "AWS",
			Description: "",
			AWSCredential: &model.AWSCredential{
				AccessKey: "foo",
				Secret:    "bar",
			},
			GCPCredential: nil,
		}, nil)
	}, func(id string) (interface{}, error) {
		if id == "" {
			return nil, errors.NotFound(id)
		}
		return dbAPI.GetCloudCreds(ctx1, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteCloudCreds(ctx1, id, nil)
	}))
}

func TestCloudCredsWaldot(t *testing.T) {
	t.Parallel()
	t.Log("running TestCloudCredsWaldot test")
	dbAPI := newObjectModelAPI(t)

	defer dbAPI.Close()

	t.Run("SelectAllCloudCredsWaldot", func(t *testing.T) {
		t.Log("running SelectAllCloudCredsWaldot test")
		ctx, _, _ := makeContext("tenant-id-waldot", []string{"bcc246fb71fac047a6985247b956bc89"})
		cloudCreds, err := dbAPI.SelectAllCloudCreds(ctx, nil)
		require.NoError(t, err)
		for _, cloudCred := range cloudCreds {
			testForMarshallability(t, cloudCred)
		}
	})
}
