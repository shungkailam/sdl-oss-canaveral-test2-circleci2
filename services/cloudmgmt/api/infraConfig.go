package api

import (
	"cloudservices/cloudmgmt/kubeprovider"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/golang/glog"
)

func init() {
	queryMap["SelectOnboardedEdgeDevices"] = `SELECT * FROM edge_device_model WHERE  ((tenant_id = :tenant_id) AND (edge_cluster_id = :edge_cluster_id) AND is_onboarded = true) ORDER BY created_at`
}

func generateInfraConfig(edgeDevices []model.EdgeDevice) (infraConfig model.InfraConfig, err error) {
	// This function is guaranteed to be passed at least one onboarded device in cluster.
	// If multiple onboarded devices are passed they are pre-sorted by creation time to reduce role churn.
	// Example:
	// Say initally devices 1,2,3 are master and devices 4,5 are workers.
	// If device 3 is removed then dev4 gets promoted to master and rest are unchanged.
	glog.V(4).Infof("Generate RKE config for %d nodes", len(edgeDevices))
	k8sProviderConfig := kubeprovider.NewRKEProvider(edgeDevices[0].ClusterID)

	if len(edgeDevices) < 1 {
		return infraConfig, errcode.NewInternalError("Need at least one device to generate RKE config")
	}

	if len(edgeDevices) == 1 {
		// If cluster has one onboarded device then make it master and worker
		ipAddress := edgeDevices[0].EdgeDeviceCore.IPAddress
		k8sProviderConfig.AddNode(ipAddress, "admin", []string{"controlplane", "etcd", "worker"})
	} else {
		const maxK8Masters = 3
		// For cluster with multiple devices make first 3 master + worker and rest workers
		for i := 0; i < len(edgeDevices); i++ {
			ipAddress := edgeDevices[i].EdgeDeviceCore.IPAddress
			if i < maxK8Masters {
				k8sProviderConfig.AddNode(ipAddress, "admin", []string{"controlplane", "etcd", "worker"})
			} else {
				k8sProviderConfig.AddNode(ipAddress, "admin", []string{"worker"})
			}
		}
	}

	infraConfig.ClusterConfig.FloatingIP = "1.1.1.1"
	infraConfig.K8sConfig.ProviderType = "RKE"
	infraConfig.K8sConfig.ProviderConfig, err = k8sProviderConfig.GetConf()
	if err != nil {
		return infraConfig, errcode.NewInternalError(err.Error())
	}
	return infraConfig, nil
}

// GetInfraConfig get a config for infra id
func (dbAPI *dbObjectModelAPI) GetInfraConfig(context context.Context, clusterID string) (infraConfig model.InfraConfig, err error) {
	if len(clusterID) == 0 {
		return infraConfig, errcode.NewBadRequestError("infraConfigID")
	}

	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return infraConfig, errcode.NewInternalError(err.Error())
	}

	edgeDeviceDBOs := []EdgeDeviceDBO{}
	tenantID := authContext.TenantID
	clusterModel := model.ClusterEntityModelDBO{BaseModelDBO: model.BaseModelDBO{TenantID: tenantID}, ClusterID: clusterID}
	param := EdgeDeviceDBO{ClusterEntityModelDBO: clusterModel}
	err = dbAPI.Query(context, &edgeDeviceDBOs, queryMap["SelectOnboardedEdgeDevices"], param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "GetInfraConfig: DB select failed: %s\n"), err.Error())
		return infraConfig, errcode.NewInternalError(err.Error())
	}

	if len(edgeDeviceDBOs) == 0 {
		// For empty cluster - ensure edge handles this response gracefully.
		return infraConfig, nil
	}
	numDevices := len(edgeDeviceDBOs)
	glog.V(4).Infof("Found %d onboarded devices in cluster %s", numDevices, clusterID)
	edgeDevices := []model.EdgeDevice{}
	for i := 0; i < len(edgeDeviceDBOs); i++ {
		edgeDevice := model.EdgeDevice{}
		err = base.Convert(&edgeDeviceDBOs[i], &edgeDevice)
		if err != nil {
			return infraConfig, errcode.NewInternalError(err.Error())
		}
		edgeDevices = append(edgeDevices, edgeDevice)
	}
	infraConfig, err = generateInfraConfig(edgeDevices)
	return infraConfig, nil
}

// GetInfraConfig get a config for infra id and write output
func (dbAPI *dbObjectModelAPI) GetInfraConfigW(context context.Context, id string, w io.Writer, r *http.Request) error {
	config, err := dbAPI.GetInfraConfig(context, id)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(config)
}
