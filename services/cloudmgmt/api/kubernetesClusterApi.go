package api

import (
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"cloudservices/common/service"
	gapi "cloudservices/operator/generated/grpc"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
	funk "github.com/thoas/go-funk"
	"google.golang.org/grpc"
)

const (
	// entityTypeKubernetesCluster is the entity type
	entityTypeKubernetesCluster = "kubernetesCluster"

	kubeVersionURL = "https://kubernetes.default.svc/version"
)

func init() {
	queryMap["SelectKubernetesClusterInfo"] = `SELECT * FROM kubernetes_cluster_info_model WHERE tenant_id = :tenant_id AND id IN (:ids)`
	queryMap["InsertKubernetesClusterInfo"] = `INSERT INTO kubernetes_cluster_info_model (id, tenant_id, chart_version, kube_version, onboarded) VALUES (:id, :tenant_id, :chart_version, :kube_version, :onboarded)`
	queryMap["UpdateKubernetesClusterInfo"] = `UPDATE kubernetes_cluster_info_model SET onboarded = :onboarded WHERE tenant_id = :tenant_id AND id = :id`
	queryMap["UpdateKubernetesClusterInfoKubeVersion"] = `UPDATE kubernetes_cluster_info_model SET kube_version = :kube_version WHERE tenant_id = :tenant_id AND id = :id`
	queryMap["UpdateServiceDomainUpdatedAt"] = `UPDATE edge_cluster_model SET updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`

	orderByHelper.Setup(entityTypeKubernetesCluster, []string{"id", "version", "created_at", "updated_at", "name", "description"})
}

// KubernetesClusterInfoDBO is the DB model
type KubernetesClusterInfoDBO struct {
	ID           string `json:"id" db:"id"`
	TenantID     string `json:"tenantId" db:"tenant_id"`
	KubeVersion  string `json:"kubeVersion" db:"kube_version"`
	ChartVersion string `json:"chartVersion" db:"chart_version"`
	Onboarded    bool   `json:"onboarded" db:"onboarded"`
}

// KubernetesClusterQueryParam is for querying the DB
type KubernetesClusterQueryParam struct {
	TenantID string   `json:"tenantId" db:"tenant_id"`
	IDs      []string `json:"ids" db:"ids"`
}

// KubeVersionResponse maps to the response from kubeVersionURL
type KubeVersionResponse struct {
	Major      string `json:"major"`
	Minor      string `json:"minor"`
	GitVersion string `json:"gitVersion"`
}

func (dbAPI *dbObjectModelAPI) CreateKubernetesCluster(ctx context.Context, i interface{} /* *model.KubernetesCluster */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	p, ok := i.(*model.KubernetesCluster)
	if !ok {
		return resp, errcode.NewInternalError("CreateKubernetesCluster: type error")
	}
	svcDomain := p.ToServiceDomain()
	return dbAPI.createServiceDomainWithTxnCallback(ctx, svcDomain, func(tx *base.WrappedTx, svcDomain *model.ServiceDomain) error {
		p.ID = svcDomain.ID
		p.TenantID = svcDomain.TenantID
		clusterDBO := &KubernetesClusterInfoDBO{
			ID:           p.ID,
			TenantID:     p.TenantID,
			ChartVersion: p.ChartVersion,
			KubeVersion:  p.KubeVersion,
			Onboarded:    p.Onboarded,
		}
		_, err := tx.NamedExec(ctx, queryMap["InsertKubernetesClusterInfo"], clusterDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in creating kubernetes cluster %s. Error: %s"), p.ID, err.Error())
		}
		return err
	}, callback)
}

func (dbAPI *dbObjectModelAPI) CreateKubernetesClusterW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, model.ToCreateV2(dbAPI.CreateKubernetesCluster), &model.KubernetesCluster{}, w, r, callback)
}

func (dbAPI *dbObjectModelAPI) SelectAllKubernetesClusters(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParam) (model.KubernetesClustersListResponsePayload, error) {
	resp := model.KubernetesClustersListResponsePayload{}
	svcDomainIDs, svcDomainIDsInPage, err := dbAPI.getServiceDomainIDsInPage(ctx, "", entitiesQueryParam, model.KubernetesClusterTargetType)
	if err != nil {
		return resp, err
	}
	svcDomains, err := dbAPI.getServiceDomainsCore(ctx, svcDomainIDsInPage, entitiesQueryParam)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in listing all kubernetes clusters. Error: %s"), err.Error())
		return resp, err
	}
	kubernetesClusters := make([]model.KubernetesCluster, 0, len(svcDomains))
	for i := range svcDomains {
		svcDomain := &svcDomains[i]
		kubernetesCluster := model.KubernetesCluster{}
		kubernetesCluster.FromServiceDomain(svcDomain)
		kubernetesClusters = append(kubernetesClusters, kubernetesCluster)
	}
	err = dbAPI.populateKubernetesClusterInfos(ctx, kubernetesClusters)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in listing all kubernetes clusters. Error: %s"), err.Error())
		return resp, err
	}
	entityListResponsePayload := makeEntityListResponsePayload(entitiesQueryParam, &ListQueryInfo{TotalCount: len(svcDomainIDs), EntityType: entityTypeKubernetesCluster})
	resp.EntityListResponsePayload = entityListResponsePayload
	resp.KubernetesClustersList = kubernetesClusters
	return resp, nil
}

func (dbAPI *dbObjectModelAPI) SelectAllKubernetesClustersW(ctx context.Context, w io.Writer, r *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParam(r)
	resp, err := dbAPI.SelectAllKubernetesClusters(ctx, entitiesQueryParam)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(resp)
}

func (dbAPI *dbObjectModelAPI) GetKubernetesCluster(ctx context.Context, id string) (model.KubernetesCluster, error) {
	kubernetesCluster := model.KubernetesCluster{}
	// No type check
	svcDomain, err := dbAPI.GetServiceDomain(ctx, id)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting the kubernetes cluster %s. Error: %s"), id, err.Error())
		return kubernetesCluster, err
	}
	kubernetesCluster.FromServiceDomain(&svcDomain)
	kubernetesClusters := []model.KubernetesCluster{kubernetesCluster}
	err = dbAPI.populateKubernetesClusterInfos(ctx, kubernetesClusters)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting the kubernetes cluster %s. Error: %s"), id, err.Error())
		return kubernetesCluster, err
	}
	return kubernetesClusters[0], nil
}

func (dbAPI *dbObjectModelAPI) GetKubernetesClusterW(ctx context.Context, id string, w io.Writer, req *http.Request) error {
	kubernetesCluster, err := dbAPI.GetKubernetesCluster(ctx, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, kubernetesCluster)
}

func (dbAPI *dbObjectModelAPI) UpdateKubernetesCluster(ctx context.Context, i interface{} /* *model.KubernetesCluster*/, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.KubernetesCluster)
	if !ok {
		return resp, errcode.NewInternalError("UpdateKubernetesCluster: type error")
	}
	if authContext.ID != "" {
		p.ID = authContext.ID
	}
	if p.ID == "" {
		return resp, errcode.NewBadRequestError("ID")
	}
	p.TenantID = authContext.TenantID
	svcDomain := p.ToServiceDomain()
	edgeRole := auth.IsEdgeRole(authContext)
	if edgeRole {
		adminCtx := base.GetAdminContextWithTenantID(ctx, authContext.TenantID)
		kubernetesCluster, err := dbAPI.GetKubernetesCluster(adminCtx, p.ID)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in updating kubernetes cluster %s. Error: %s"), p.ID, err.Error())
			return resp, err
		}
		err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
			clusterDBO := &KubernetesClusterInfoDBO{TenantID: kubernetesCluster.TenantID, ID: kubernetesCluster.ID, Onboarded: p.Onboarded}
			_, err = tx.NamedExec(ctx, queryMap["UpdateKubernetesClusterInfo"], clusterDBO)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in updating kubernetes cluster %s. Error: %s"), p.ID, err.Error())
				return err
			}
			_, err = tx.NamedExec(ctx, queryMap["UpdateServiceDomainUpdatedAt"], model.BaseModelDBO{
				TenantID:  clusterDBO.TenantID,
				ID:        clusterDBO.ID,
				UpdatedAt: base.RoundedNow(),
			})
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in updating kubernetes cluster %s. Error: %s"), p.ID, err.Error())
				return err
			}
			resp.ID = kubernetesCluster.ID
			return nil
		})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in updating kubernetes cluster %s. Error: %s"), p.ID, err.Error())
			return resp, err
		}
	} else {
		svcResp, err := dbAPI.updateServiceDomainWithTxnCallback(ctx, svcDomain, nil, callback)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in updating kubernetes cluster %s. Error: %s"), p.ID, err.Error())
			return resp, err
		}
		// There are no kubernetes cluster specific fields to be updated for now
		resp.ID = svcResp.(model.UpdateDocumentResponse).ID
	}
	return resp, nil
}

func (dbAPI *dbObjectModelAPI) UpdateKubernetesClusterW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(ctx, model.ToUpdateV2(dbAPI.UpdateKubernetesCluster), &model.KubernetesCluster{}, w, r, callback)
}

func (dbAPI *dbObjectModelAPI) DeleteKubernetesCluster(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	return dbAPI.DeleteServiceDomain(ctx, id, callback)
}

func (dbAPI *dbObjectModelAPI) DeleteKubernetesClusterW(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(ctx, model.ToDeleteV2(dbAPI.DeleteKubernetesCluster), id, w, callback)
}

func (dbAPI *dbObjectModelAPI) GetKubernetesClusterHandle(ctx context.Context, kubernetestClusterID string, payload model.GetHandlePayload) (model.KubernetesClusterCert, error) {
	kubernetesClusterCert := model.KubernetesClusterCert{}
	edgeCert, err := dbAPI.GetServiceDomainHandle(ctx, kubernetestClusterID, payload)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting handle for kubernetes cluster %s. Error: %s"), kubernetestClusterID, err.Error())
		return kubernetesClusterCert, err
	}
	kubernetesClusterCert.FromEdgeCert(&edgeCert)
	return kubernetesClusterCert, nil
}

func (dbAPI *dbObjectModelAPI) GetKubernetesClusterHandleW(ctx context.Context, kubernetestClusterID string, w io.Writer, req *http.Request) error {
	doc := model.GetHandlePayload{}
	var r io.Reader = req.Body
	err := base.Decode(&r, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error decoding into GetHandlePayload. Error: %s"), err.Error())
		return errcode.NewBadRequestError("GetHandlePayload")
	}
	resp, err := dbAPI.GetKubernetesClusterHandle(ctx, kubernetestClusterID, doc)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, resp)
}

func (dbAPI *dbObjectModelAPI) GetKubernetesClusterInstaller(ctx context.Context) (model.KubernetesClusterInstaller, error) {
	resp := model.KubernetesClusterInstaller{}
	gRequest := &gapi.GetReleaseHelmChartRequest{}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		gResponse, err := client.GetReleaseHelmChart(ctx, gRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get kubernetes cluster installer. Error: %s"), err.Error())
			return err
		}
		resp.ID = gResponse.Release.Id
		resp.URL = gResponse.Release.Url
		return nil
	}
	err := service.CallClient(ctx, service.OperatorService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in operator service call. Error: %s"), err.Error())
	}
	return resp, err
}

func (dbAPI *dbObjectModelAPI) GetKubernetesClusterInstallerW(ctx context.Context, w io.Writer, req *http.Request) error {
	resp, err := dbAPI.GetKubernetesClusterInstaller(ctx)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, resp)
}

func (dbAPI *dbObjectModelAPI) UpdateKubernetesClusterKubeVersion(ctx context.Context, kubernetestClusterID string) error {
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("GET", kubeVersionURL, nil)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in creating kube version request for cluster %s. Error: %s"), kubernetestClusterID, err.Error())
		return err
	}

	resp, err := wsMsgService.SendHTTPRequest(ctx, authContext.TenantID, kubernetestClusterID, req, kubeVersionURL)
	if err != nil {
		glog.Warningf(base.PrefixRequestID(ctx, "Error in sending request to %s to get kube version. Error: %s"), kubernetestClusterID, err.Error())
		return err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	kubeVersion := KubeVersionResponse{}
	err = base.ConvertFromJSON(body, &kubeVersion)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in converting kube version response for %s. Error: %s"), kubernetestClusterID, err.Error())
		return err
	}

	return dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		_, err := tx.NamedExec(ctx, queryMap["UpdateKubernetesClusterInfoKubeVersion"], KubernetesClusterInfoDBO{
			TenantID:    authContext.TenantID,
			ID:          kubernetestClusterID,
			KubeVersion: kubeVersion.GitVersion,
		})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in updating kube version % for cluster %s. Error: %s"), kubeVersion, kubernetestClusterID, err.Error())
			return err
		}
		_, err = tx.NamedExec(ctx, queryMap["UpdateServiceDomainUpdatedAt"], model.BaseModelDBO{
			TenantID:  authContext.TenantID,
			ID:        kubernetestClusterID,
			UpdatedAt: base.RoundedNow(),
		})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in updating kube version % for cluster %s. Error: %s"), kubeVersion, kubernetestClusterID, err.Error())
			return err
		}
		return nil
	})
}

func (dbAPI *dbObjectModelAPI) populateKubernetesClusterInfos(ctx context.Context, kubernetesClusters []model.KubernetesCluster) error {
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	if len(kubernetesClusters) == 0 {
		return nil
	}
	ids := funk.Map(kubernetesClusters, func(x model.KubernetesCluster) string { return x.ID }).([]string)
	clusterDBOs := []KubernetesClusterInfoDBO{}
	err = dbAPI.QueryIn(ctx, &clusterDBOs, queryMap["SelectKubernetesClusterInfo"], KubernetesClusterQueryParam{TenantID: authContext.TenantID, IDs: ids})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in fetching kubernetes cluster info. Error: %s"), err.Error())
		return err
	}
	copyClusterInfos(kubernetesClusters, clusterDBOs)
	return nil
}

func copyClusterInfos(kubernetesClusters []model.KubernetesCluster, clusterDBOs []KubernetesClusterInfoDBO) {
	clusterDBOsMap := map[string]*KubernetesClusterInfoDBO{}
	for i := range clusterDBOs {
		clusterDBO := &clusterDBOs[i]
		clusterDBOsMap[clusterDBO.ID] = clusterDBO
	}
	for i := range kubernetesClusters {
		kubernetesCluster := &kubernetesClusters[i]
		clusterDBO, ok := clusterDBOsMap[kubernetesCluster.ID]
		if !ok {
			// Must not happen as there is FK
			continue
		}
		kubernetesCluster.ChartVersion = clusterDBO.ChartVersion
		kubernetesCluster.KubeVersion = clusterDBO.KubeVersion
		kubernetesCluster.Onboarded = clusterDBO.Onboarded
	}
}
