package api_test

import (
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"reflect"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
)

func TestKubernetesCluster(t *testing.T) {
	t.Parallel()
	t.Log("running TestKubernetesCluster test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	defer dbAPI.Close()
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	edgeAuthContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "edge",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	edgeCtx := context.WithValue(context.Background(), base.AuthContextKey, edgeAuthContext)
	defer dbAPI.DeleteTenant(ctx, tenantID, nil)
	t.Run("Create/Get/Delete Kubernetes Clusters", func(t *testing.T) {
		cluster := model.KubernetesCluster{
			BaseModel: model.BaseModel{
				TenantID: tenantID,
			},
			Name:         "test-cluster",
			ChartVersion: "1.0.0",
			KubeVersion:  "1.2.0",
		}
		resp, err := dbAPI.CreateKubernetesCluster(ctx, &cluster, nil)
		require.NoError(t, err)
		cluster.ID = resp.(model.CreateDocumentResponse).ID
		defer dbAPI.DeleteKubernetesCluster(ctx, cluster.ID, nil)
		listResp, err := dbAPI.SelectAllKubernetesClusters(ctx, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		clusters := listResp.KubernetesClustersList
		require.Equal(t, 1, len(clusters), "One must exist")
		t.Logf("Cluster %+v\n", clusters[0])
		cluster.CreatedAt = clusters[0].CreatedAt
		cluster.UpdatedAt = clusters[0].UpdatedAt
		cluster.Version = clusters[0].Version
		require.Equal(t, true, reflect.DeepEqual(&cluster, &clusters[0]), "Unequal output")
		svcListResp, err := dbAPI.SelectAllServiceDomains(ctx, &model.EntitiesQueryParamV1{})
		require.NoError(t, err)
		require.Equal(t, len(svcListResp), 0, "Service Domain count must be 0")
		// GetServiceDomain must return the Kubernetes Cluster if the ID is specified
		svcGetResp, err := dbAPI.GetServiceDomain(ctx, cluster.ID)
		require.NoError(t, err)
		kubernetesCluster := model.KubernetesCluster{}
		kubernetesCluster.FromServiceDomain(&svcGetResp)
		t.Logf("Service Domain %+v\n", kubernetesCluster)
		kubernetesCluster.CreatedAt = cluster.CreatedAt
		kubernetesCluster.UpdatedAt = cluster.UpdatedAt
		kubernetesCluster.Version = cluster.Version
		kubernetesCluster.ChartVersion = cluster.ChartVersion
		kubernetesCluster.KubeVersion = cluster.KubeVersion
		require.Equal(t, true, reflect.DeepEqual(&cluster, &kubernetesCluster), "Unequal output")
		clusterResp, err := dbAPI.GetKubernetesCluster(ctx, cluster.ID)
		require.NoError(t, err)
		t.Logf("Cluster %+v\n", clusterResp)
		require.Equal(t, true, reflect.DeepEqual(&cluster, &clusterResp), "Unequal output")
		cluster.Onboarded = true
		_, err = dbAPI.UpdateKubernetesCluster(ctx, &cluster, nil)
		require.NoError(t, err)
		clusterResp, err = dbAPI.GetKubernetesCluster(ctx, cluster.ID)
		require.NoError(t, err)
		t.Logf("Cluster %+v\n", clusterResp)
		cluster.CreatedAt = clusterResp.CreatedAt
		cluster.UpdatedAt = clusterResp.UpdatedAt
		cluster.Version = clusterResp.Version
		t.Logf("Comparing clusters \n%+v\n%+v\n", cluster, clusterResp)
		require.Equal(t, false, reflect.DeepEqual(&cluster, &clusterResp), "Equal output")
		_, err = dbAPI.UpdateKubernetesCluster(edgeCtx, &cluster, nil)
		require.NoError(t, err)
		clusterResp, err = dbAPI.GetKubernetesCluster(ctx, cluster.ID)
		require.NoError(t, err)
		t.Logf("Cluster %+v\n", clusterResp)
		cluster.CreatedAt = clusterResp.CreatedAt
		cluster.UpdatedAt = clusterResp.UpdatedAt
		cluster.Version = clusterResp.Version
		t.Logf("Comparing clusters \n%+v\n%+v\n", cluster, clusterResp)
		require.Equal(t, true, reflect.DeepEqual(&cluster, &clusterResp), "Unequal output")
		_, err = dbAPI.DeleteKubernetesCluster(ctx, cluster.ID, nil)
		require.NoError(t, err)
		listResp, err = dbAPI.SelectAllKubernetesClusters(ctx, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		clusters = listResp.KubernetesClustersList
		require.Equal(t, 0, len(clusters), "None must exist")
		_, err = dbAPI.GetKubernetesCluster(ctx, cluster.ID)
		require.Error(t, err)
		installer, err := dbAPI.GetKubernetesClusterInstaller(ctx)
		require.NoError(t, err)
		t.Logf("Installer %+v", installer)
		require.NotEmpty(t, installer.ID, "ID is empty")
		require.NotEmpty(t, installer.URL, "URL is empty")
	})
}
