package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"context"
	"net/http"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
)

func getContainerRegistryProfileRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/containerregistries",
			// swagger:route GET /v1/containerregistries ContainerRegistryList
			//
			// Get container registry profiles. ntnx:ignore
			//
			// Retrieves a list of all container registry profiles.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ContainerRegistryListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllContainerRegistriesW, "/containerregistries"),
		},
		{
			method: "GET",
			path:   "/v1/containerregistries/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllContainerRegistriesW, "/containerregistries"),
		},
		{
			method: "GET",
			path:   "/v1.0/containerregistries",
			// swagger:route GET /v1.0/containerregistries Container_Registry ContainerRegistryListV2
			//
			// Get container registry profiles.
			//
			// Retrieves a list of all container registry profiles.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ContainerRegistryListResponseV2
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllContainerRegistriesWV2, "/containerregistries"),
		},
		{
			method: "GET",
			path:   "/v1.0/containerregistries/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllContainerRegistriesWV2, "/containerregistries"),
		},
		{
			method: "GET",
			path:   "/v1/projects/:projectId/containerregistries",
			// swagger:route GET /v1/projects/{projectId}/containerregistries ProjectGetContainerRegistries
			//
			// Get container registry profiles by project ID. ntnx:ignore
			//
			// Retrieves a list of all container registry profiles with a given ID {projectId}
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ContainerRegistryListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllContainerRegistriesForProjectW, "/project-containerregistries", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId/containerregistries",
			// swagger:route GET /v1.0/projects/{projectId}/containerregistries Container_Registry ProjectGetContainerRegistriesV2
			//
			// Get container registry profiles by project ID.
			//
			// Retrieves a list of all container registry profiles with a given ID {projectId}
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ContainerRegistryListResponseV2
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllContainerRegistriesForProjectWV2, "/project-containerregistries", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1/containerregistries/:id",
			// swagger:route GET /v1/containerregistries/{id} ContainerRegistryGet
			//
			// Get a container registry profile by profile ID. ntnx:ignore
			//
			// Retrieves a container registry profile with a given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ContainerRegistryGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetContainerRegistryW, "/containerregistries/:id", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/containerregistries/:id",
			// swagger:route GET /v1.0/containerregistries/{id} Container_Registry ContainerRegistryGetV2
			//
			// Get a container registry profile by profile ID.
			//
			// Retrieves a container registry profile with a given ID {id}.
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ContainerRegistryGetResponseV2
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetContainerRegistryWV2, "/containerregistries/:id", "id"),
		},
		{
			method: "DELETE",
			path:   "/v1/containerregistries/:id",
			// swagger:route DELETE /v1/containerregistries/{id} ContainerRegistryDelete
			//
			// Delete a container registry profile by profile ID. ntnx:ignore
			//
			// Deletes a container registry profile with a given ID {id}.
			//
			// This will delete the containerregistries with the given id.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DeleteDocumentResponse
			//       default: APIError
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteContainerRegistryW, msgSvc, "dockerprofile", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/containerregistries/:id",
			// swagger:route DELETE /v1.0/containerregistries/{id} Container_Registry ContainerRegistryDeleteV2
			//
			// Delete a container registry profile by profile ID.
			//
			// Deletes a container registry profile with a given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteContainerRegistryWV2, msgSvc, "dockerprofile", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "POST",
			path:   "/v1/containerregistries",
			// swagger:route POST /v1/containerregistries ContainerRegistryCreate
			//
			// Create a container registry profile. ntnx:ignore
			//
			// Creates a container registry profile.
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
			//       200: CreateDocumentResponse
			//       default: APIError
			handle: makeCreateHandle(dbAPI, dbAPI.CreateContainerRegistryW, msgSvc, "dockerprofile", NOTIFICATION_TENANT),
		},
		{
			method: "POST",
			path:   "/v1.0/containerregistries",
			// swagger:route POST /v1.0/containerregistries Container_Registry ContainerRegistryCreateV2
			//
			// Create a container registry profile.
			//
			// Creates a container registry profile.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateContainerRegistryWV2, msgSvc, "dockerprofile", NOTIFICATION_TENANT),
		},
		{
			method: "PUT",
			path:   "/v1/containerregistries/:id",
			// swagger:route PUT /v1/containerregistries/{id} ContainerRegistryUpdate
			//
			// Update a container registry profile. ntnx:ignore
			//
			// Updates a container registry profile.
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
			//       200: UpdateDocumentResponse
			//       default: APIError
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateContainerRegistryW, msgSvc, "dockerprofile", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "PUT",
			path:   "/v1.0/containerregistries/:id",
			// swagger:route PUT /v1.0/containerregistries/{id} Container_Registry ContainerRegistryUpdateV2
			//
			// Update a container registry profile.
			//
			// Updates a container registry profile.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateContainerRegistryWV2, msgSvc, "dockerprofile", NOTIFICATION_TENANT, "id"),
		},
		// private API to encrypt all docker profiles
		{
			method: "POST",
			path:   "/v1/encryptcontainerregistries",
			handle: getContext(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
				// This is called by wget. There is this alpine busybox wget issue.
				// https://svn.dd-wrt.com//ticket/5771
				// The received context is cancelled when the write socket is closed.
				// So, create a new independent context.
				ctx := context.WithValue(context.Background(), base.RequestIDKey, base.GetRequestID(r.Context()))
				glog.Info(base.PrefixRequestID(ctx, "POST /v1/encryptcontainerregistries"))
				err := dbAPI.EncryptAllContainerRegistriesW(ctx, r.Body)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
				} else {
					w.WriteHeader(http.StatusOK)
				}
			}),
		},
	}
}
