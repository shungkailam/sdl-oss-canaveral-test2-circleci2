package api_test

import (
	"cloudservices/common/model"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestCertificates(t *testing.T) {
	t.Parallel()
	t.Log("running TestCertificates test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	project := createEmptyCategoryProject(t, dbAPI, tenantID)
	projectID := project.ID
	authContext, _, _ := makeContext(tenantID, []string{projectID})

	// Teardown
	defer func() {
		dbAPI.DeleteProject(authContext, projectID, nil)
		dbAPI.DeleteTenant(authContext, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create Certificates", func(t *testing.T) {
		t.Log("running Create Certificates test")

		// Certificate is already there as a part of edge creation
		i, err := dbAPI.CreateCertificates(authContext, nil)
		require.NoError(t, err)
		cert, ok := i.(model.Certificates)
		if !ok {
			t.Fatalf("'Certificates' object not returned by CreateCertificates call")
		}
		t.Logf("CreateCertificates call successful, %+v", cert)

		// Get the tenant_rootca_model object corresponding to the tenantID and
		// compare with the root CA returned by CreateCertificates.
		rootCA, err := dbAPI.GetTenantRootCA(tenantID)
		require.NoErrorf(t, err, "Failed to get root CA for tenantID '%s' with error: %s", tenantID, err)
		if strings.Compare(rootCA, cert.CACertificate) != 0 {
			t.Errorf("Root CA returned by 'CreateCertificates' is different from root CA for tenant in DB")
			t.Fatalf("Root CA from 'CreateCertificates': %s \n Root CA for tenant in DB: %s", cert.CACertificate, rootCA)
		}
	})
}
