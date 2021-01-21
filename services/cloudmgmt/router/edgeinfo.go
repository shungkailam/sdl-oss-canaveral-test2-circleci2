package router

import (
	"cloudservices/cloudmgmt/api"
)

func getEdgeInfoRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/edgesInfo",
			// swagger:route GET /v1/edgesInfo EdgeInfoList
			//
			// Get edge resource, build, and version details. ntnx:ignore
			//
			// Retrieves all edge resource, build, and version details.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: EdgeInfoListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllEdgesInfoW, "/edgesInfo"),
		},
		{
			method: "GET",
			path:   "/v1/edgesInfo/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllEdgesInfoW, "/edgesInfo"),
		},
		{
			method: "GET",
			path:   "/v1.0/edgesinfo",
			// swagger:route GET /v1.0/edgesinfo Edge_Info EdgeInfoListV2
			//
			// Get edge resource, build, and version details. ntnx:ignore
			//
			// Retrieves all edge resource, build, and version details.
			//
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: EdgeInfoListResponseV2
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllEdgesInfoWV2, "/edgesinfo"),
		},
		{
			method: "GET",
			path:   "/v1.0/edgesinfo/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllEdgesInfoWV2, "/edgesinfo"),
		},
		{
			method: "GET",
			path:   "/v1/projects/:projectId/edgesinfo",
			// swagger:route GET /v1/projects/{projectId}/edgesinfo ProjectGetEdgesInfo
			//
			// Get all edge information for a project by ID. ntnx:ignore
			//
			// Retrieves all edge resource, build, and version details by project ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: EdgeInfoListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllEdgesInfoForProjectW, "/project-edgesinfo", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId/edgesinfo",
			// swagger:route GET /v1.0/projects/{projectId}/edgesinfo Edge_Info ProjectGetEdgesInfoV2
			//
			// Get all edge information for a project by ID. ntnx:ignore
			//
			// Retrieves all edge resource, build, and version details by project ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: EdgeInfoListResponseV2
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllEdgesInfoForProjectWV2, "/project-edgesinfo", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1/edges/:edgeId/info",
			// swagger:route GET /v1/edges/{edgeId}/info EdgeInfoGet
			//
			// Get all edge information by edge ID. ntnx:ignore
			//
			// Retrieves all edge resource, build, and version details for a given edge ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: EdgeInfoGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetEdgeInfoW, "/edgesInfo", "edgeId"),
		},
		{
			method: "GET",
			path:   "/v1.0/edges/:edgeId/info",
			// swagger:route GET /v1.0/edges/{edgeId}/info Edge_Info EdgeInfoGetV2
			//
			// Get all edge (service domain) information by edge ID. ntnx:ignore
			//
			// Once installed, the Karbon Platform Services Service Domain software provides the service domain infrastructure.
			//
			// Retrieves all resource, build, and version details for a given service domain by ID.
			// The ID is the service domain serial number used when you added the service domain.
			//
			// This request also requires an Authorization header which specifies your API key.
			// This example Python request is for a service domain with serial number 5f31b963-acac-4368-a157-0c7dc0d32000,
			// where bearer_api_key is your actual API key.
			//
			// import http.client
			//
			// conn = http.client.HTTPSConnection("karbon.nutanix.com")
			// headers = { 'authorization': "bearer_api_key" }
			// conn.request("GET", "//v1.0/edges/5f31b963-acac-4368-a157-0c7dc0d32000/info", headers=headers)
			// res = conn.getresponse()
			// data = res.read()
			// print(data.decode("utf-8"))
			//
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: EdgeInfoGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetEdgeInfoW, "/edgesInfo", "edgeId"),
		},
		{
			method: "PUT",
			path:   "/v1/edges/:id/info",
			// swagger:route PUT /v1/edges/{id}/info EdgeInfoUpdate
			//
			// Update edge information by edge ID. ntnx:ignore
			//
			// Update edge resource, build, and version details for a given edge ID.
			//
			// Once installed, the Karbon Platform Services Service Domain software provides the service domain infrastructure.
			//
			// Update resource, build, and version details for a given service domain by ID.
			// The ID is the service domain serial number used when you added the service domain.
			//
			// This request also requires an Authorization header which specifies your API key.
			//
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: UpdateDocumentResponse
			//       default: APIError
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateEdgeInfoW, msgSvc, "edgesInfo", NOTIFICATION_NONE, "id"),
		},
		{
			method: "PUT",
			path:   "/v1.0/edges/:id/info",
			// swagger:route PUT /v1.0/edges/{id}/info Edge_Info EdgeInfoUpdateV2
			//
			// Update edge resource, build, and version details for a given edge ID. ntnx:ignore
			//
			// Once installed, the Xi IoT edge software provides the service domain infrastructure.
			//
			// Update resource, build, and version details for a given service domain VM by ID.
			// The ID is the service domain serial number used when you added the service domain.
			//
			// This request also requires an Authorization header which specifies your API key.
			// It returns the updated service domain information in JSON format.
			//
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: UpdateDocumentResponseV2
			//       default: APIError
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateEdgeInfoWV2, msgSvc, "edgesInfo", NOTIFICATION_NONE, "id"),
		},
	}
}
