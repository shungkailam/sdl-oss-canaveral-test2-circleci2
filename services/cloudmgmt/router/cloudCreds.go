package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"context"
	"net/http"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
)

func getCloudCredsRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/cloudcreds",
			// swagger:route GET /v1/cloudcreds CloudCredsList
			//
			// Get all cloud service profiles. ntnx:ignore
			//
			// Retrieves all cloud service provider profiles.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: CloudCredsListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllCloudCredsW, "/cloudcreds"),
		},
		{
			method: "GET",
			path:   "/v1/cloudcreds/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllCloudCredsW, "/cloudcreds"),
		},
		{
			method: "GET",
			path:   "/v1.0/cloudprofiles",
			// swagger:route GET /v1.0/cloudprofiles Cloud_Profile CloudProfileList
			//
			// Get all cloud service profiles.
			//
			// Retrieves all cloud service provider profiles.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: CloudProfileListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllCloudCredsWV2, "/cloudcreds"),
		},
		{
			method: "GET",
			path:   "/v1.0/cloudprofiles/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllCloudCredsWV2, "/cloudcreds"),
		},
		{
			method: "GET",
			path:   "/v1/projects/:projectId/cloudcreds",
			// swagger:route GET /v1/projects/{projectId}/cloudcreds ProjectGetCloudCreds
			//
			// Get cloud profiles according to project ID. ntnx:ignore
			//
			// Retrieves all cloud service profiles for a project with a given ID {projectId}
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: CloudCredsListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllCloudCredsForProjectW, "/project-cloudcreds", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId/cloudprofiles",
			// swagger:route GET /v1.0/projects/{projectId}/cloudprofiles Cloud_Profile ProjectGetCloudProfiles
			//
			// Get cloud profiles according to project ID.
			//
			// Retrieves all cloud service profiles for a project with a given ID {projectId}
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: CloudProfileListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllCloudCredsForProjectWV2, "/project-cloudcreds", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1/cloudcreds/:id",
			// swagger:route GET /v1/cloudcreds/{id} CloudCredsGet
			//
			// Get a cloud profile according to profile ID. ntnx:ignore
			//
			// Retrieves a cloud service profile with a given ID {id}
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: CloudCredsGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetCloudCredsW, "/cloudcreds/:id", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/cloudprofiles/:id",
			// swagger:route GET /v1.0/cloudprofiles/{id} Cloud_Profile CloudProfileGet
			//
			// Get a cloud profile according to profile ID.
			//
			// Retrieves a cloud service profile with a given ID {id}
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: CloudProfileGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetCloudCredsW, "/cloudcreds/:id", "id"),
		},
		{
			method: "DELETE",

			path: "/v1/cloudcreds/:id",
			// swagger:route DELETE /v1/cloudcreds/{id} CloudCredsDelete
			//
			// Delete a cloud profile by its ID. ntnx:ignore
			//
			// Delete a cloud service profile with the given ID {id}
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteCloudCredsW, msgSvc, "cloudcreds", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "DELETE",

			path: "/v1.0/cloudprofiles/:id",
			// swagger:route DELETE /v1.0/cloudprofiles/{id} Cloud_Profile CloudProfileDelete
			//
			// Delete a cloud profile by its ID.
			//
			// Delete a cloud service profile with the given ID {id}
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteCloudCredsWV2, msgSvc, "cloudcreds", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "POST",
			path:   "/v1/cloudcreds",
			// swagger:route POST /v1/cloudcreds CloudCredsCreate
			//
			// Create a cloud profile. ntnx:ignore
			//
			// Create a cloud service profile.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateCloudCredsW, msgSvc, "cloudcreds", NOTIFICATION_TENANT),
		},
		{
			method: "POST",
			path:   "/v1.0/cloudprofiles",
			// swagger:route POST /v1.0/cloudprofiles Cloud_Profile CloudProfileCreate
			//
			// Create a cloud profile.
			//
			// Create a cloud service profile.
			//
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateCloudCredsWV2, msgSvc, "cloudcreds", NOTIFICATION_TENANT),
		},
		{
			method: "PUT",
			path:   "/v1/cloudcreds",
			// swagger:route PUT /v1/cloudcreds CloudCredsUpdate
			//
			// Update a cloud profile. ntnx:ignore
			//
			// Update an existing cloud profile.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateCloudCredsW, msgSvc, "cloudcreds", NOTIFICATION_TENANT, ""),
		},
		{
			method: "PUT",
			path:   "/v1/cloudcreds/:id",
			// swagger:route PUT /v1/cloudcreds/{id} CloudCredsUpdateV2
			//
			// Update a cloud profile by its ID. ntnx:ignore
			//
			// Update an existing cloud profile with a given ID {id}
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateCloudCredsW, msgSvc, "cloudcreds", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "PUT",
			path:   "/v1.0/cloudprofiles/:id",
			// swagger:route PUT /v1.0/cloudprofiles/{id} Cloud_Profile CloudProfileUpdate
			//
			// Update a cloud profile by its ID.
			//
			// Update an existing cloud profile with a given ID {id}
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateCloudCredsWV2, msgSvc, "cloudcreds", NOTIFICATION_TENANT, "id"),
		},
		// private API to encrypt all cloud profiles
		{
			method: "POST",
			path:   "/v1/encryptcloudcreds",
			handle: getContext(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
				// This is called by wget. There is this alpine busybox wget issue.
				// https://svn.dd-wrt.com//ticket/5771
				// The received context is cancelled when the write socket is closed.
				// So, create a new independent context.
				ctx := context.WithValue(context.Background(), base.RequestIDKey, base.GetRequestID(r.Context()))
				glog.Info(base.PrefixRequestID(ctx, "POST /v1/encryptcloudcreds"))
				err := dbAPI.EncryptAllCloudCredsW(ctx, r.Body)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
				} else {
					w.WriteHeader(http.StatusOK)
				}
			}),
		},
	}
}
