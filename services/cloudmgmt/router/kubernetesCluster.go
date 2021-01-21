package router

import (
	"cloudservices/cloudmgmt/api"
)

func getKubernetesClusterRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/kubernetesclusters",
			// swagger:route GET /v1.0/kubernetesclusters Kubernetes_Cluster KubernetesClustersList
			//
			// Get all kubernetes clusters.
			//
			// Retrieves a list of all kubernetes clusters.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: KubernetesClustersListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllKubernetesClustersW, "/v1.0/kubernetesclusters"),
		},
		{
			method: "GET",
			path:   "/v1.0/kubernetesclusters/:id",
			// swagger:route GET /v1.0/kubernetesclusters/{id} Kubernetes_Cluster KubernetesClustersGet
			//
			// Get single kubernetes cluster.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: KubernetesClustersGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetKubernetesClusterW, "/v1.0/kubernetesclusters/:id", "id"),
		},
		{
			method: "POST",
			path:   "/v1.0/kubernetesclusters",
			// swagger:route POST /v1.0/kubernetesclusters Kubernetes_Cluster KubernetesClustersCreate
			//
			// Create a kubernetes cluster.
			//
			// Create a kubernetes cluster.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: CreateDocumentResponseV2
			//       default: APIError
			handle: makeCreateHandle(dbAPI, dbAPI.CreateKubernetesClusterW, msgSvc, "servicedomain", NOTIFICATION_NONE),
		},
		{
			method: "PUT",
			path:   "/v1.0/kubernetesclusters/:id",
			// swagger:route PUT /v1.0/kubernetesclusters/{id} Kubernetes_Cluster KubernetesClustersUpdate
			//
			// Update a kubernetes cluster by its ID.
			//
			// Updates a kubernetes cluster by its ID {id}.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: UpdateDocumentResponseV2
			//       default: APIError
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateKubernetesClusterW, msgSvc, "servicedomain", NOTIFICATION_EDGE, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/kubernetesclusters/:id",
			// swagger:route DELETE /v1.0/kubernetesclusters/{id} Kubernetes_Cluster KubernetesClustersDelete
			//
			// Delete a kubernetes cluster as specified by its ID.
			//
			// Deletes a kubernetes cluster by its ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DeleteDocumentResponseV2
			//       default: APIError
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteKubernetesClusterW, msgSvc, "servicedomain", NOTIFICATION_EDGE, "id"),
		},
		{
			method: "POST",
			path:   "/v1.0/kubernetesclusters/:id/handle",
			// swagger:route POST /v1.0/kubernetesclusters/{id}/handle Kubernetes_Cluster KubernetesClustersHandle
			//
			// Retrieves the certificate and private key for the kubernetes cluster by its given ID. ntnx:ignore
			//
			// Retrieves the certificate and private key for the kubernetes cluster by its given ID.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: KubernetesClustersHandleResponse
			//       default: APIError
			handle: makePostHandleNoAuth2(dbAPI, dbAPI.GetKubernetesClusterHandleW, "/v1.0/kubernetesclusters/:id/handle", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/kubernetescluster-installer",
			// swagger:route GET /v1.0/kubernetescluster-installer Kubernetes_Cluster KubernetesClusterInstaller
			//
			// Get the kubernetes clusters helm installer.
			//
			//
			// Gets the kubernetes cluster helm installer.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: KubernetesClusterInstallerResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.GetKubernetesClusterInstallerW, "/v1.0/kubernetescluster-installer"),
		},
	}
}
