package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"net/http"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
)

func getDockerProfileRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/dockerprofiles",
			// swagger:route GET /v1/dockerprofiles DockerProfileList
			//
			// Get DockerProfiles. ntnx:ignore
			//
			// Retrieves all DockerProfiles for a tenant.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: DockerProfileListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllDockerProfilesW, "/dockerprofiles"),
		},
		{
			method: "GET",
			path:   "/v1/dockerprofiles/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllDockerProfilesW, "/dockerprofiles"),
		},
		{
			method: "GET",
			path:   "/v1.0/dockerprofiles",
			// swagger:route GET /v1.0/dockerprofiles Container_Registry DockerProfileListV2
			//
			// Get DockerProfiles. ntnx:ignore
			//
			// Retrieves all DockerProfiles.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: DockerProfileListResponseV2
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllDockerProfilesWV2, "/dockerprofiles"),
		},
		{
			method: "GET",
			path:   "/v1.0/dockerprofiles/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllDockerProfilesWV2, "/dockerprofiles"),
		},
		{
			method: "GET",
			path:   "/v1/projects/:projectId/dockerprofiles",
			// swagger:route GET /v1/projects/{projectId}/dockerprofiles ProjectGetDockerProfiles
			//
			// Get project DockerProfiles. ntnx:ignore
			//
			// Retrieves all DockerProfiles for a project by project ID {projectId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DockerProfileListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllDockerProfilesForProjectW, "/project-dockerprofiles", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId/dockerprofiles",
			// swagger:route GET /v1.0/projects/{projectId}/dockerprofiles Container_Registry ProjectGetDockerProfilesV2
			//
			// Get project DockerProfiles. ntnx:ignore
			//
			// Retrieves all DockerProfiles for a project by project ID {projectId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DockerProfileListResponseV2
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllDockerProfilesForProjectWV2, "/project-dockerprofiles", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1/dockerprofiles/:id",
			// swagger:route GET /v1/dockerprofiles/{id} DockerProfileGet
			//
			// Get dockerprofiles. ntnx:ignore
			//
			// Retrieves dockerProfiles with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DockerProfileGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetDockerProfileW, "/dockerprofiles/:id", "id"),
		},
		{
			method: "DELETE",
			path:   "/v1/dockerprofiles/:id",
			// swagger:route DELETE /v1/dockerprofiles/{id} DockerProfileDelete
			//
			// Delete dockerprofiles. ntnx:ignore
			//
			// Deletes the dockerprofiles with the given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteDockerProfileW, msgSvc, "dockerprofile", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "POST",
			path:   "/v1/dockerprofiles",
			// swagger:route POST /v1/dockerprofiles DockerProfileCreate
			//
			// Create dockerprofiles. ntnx:ignore
			//
			// Creates a dockerprofile.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateDockerProfileW, msgSvc, "dockerprofile", NOTIFICATION_TENANT),
		},
		{
			method: "PUT",
			path:   "/v1/dockerprofiles",
			// swagger:route PUT /v1/dockerprofiles DockerProfileUpdate
			//
			// Update dockerprofile. ntnx:ignore
			//
			// Update a dockerprofiles.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateDockerProfileW, msgSvc, "dockerprofile", NOTIFICATION_TENANT, ""),
		},
		{
			method: "PUT",
			path:   "/v1/dockerprofiles/:id",
			// swagger:route PUT /v1/dockerprofiles/{id} DockerProfileUpdateV2
			//
			// Update dockerprofiles. ntnx:ignore
			//
			// Update a dockerprofiles by ID {id}.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateDockerProfileW, msgSvc, "dockerprofile", NOTIFICATION_TENANT, "id"),
		},
		// private API to encrypt all docker profiles
		{
			method: "POST",
			path:   "/v1/encryptdockerprofiles",
			handle: getContext(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
				reqID := r.Context().Value(base.RequestIDKey).(string)
				glog.Infof("POST /v1/encryptdockerprofiles Request %s", reqID)
				err := dbAPI.EncryptAllDockerProfilesW(r.Context(), r.Body)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
				} else {
					w.WriteHeader(http.StatusOK)
				}
			}),
		},
	}
}
