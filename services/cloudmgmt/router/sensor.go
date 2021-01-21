package router

import (
	"cloudservices/cloudmgmt/api"
)

func getSensorRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/sensors",
			// swagger:route GET /v1/sensors SensorList
			//
			// Get sensors. ntnx:ignore
			//
			// Retrieves all sensors for a tenant.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: SensorListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllSensorsW, "/sensors"),
		},
		{
			method: "GET",
			path:   "/v1/sensors/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllSensorsW, "/sensors"),
		},
		{
			method: "GET",
			path:   "/v1.0/sensors",
			// swagger:route GET /v1.0/sensors Sensor SensorListV2
			//
			// Get sensors. ntnx:ignore
			//
			// Retrieves all sensors for a tenant.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: SensorListResponseV2
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllSensorsWV2, "/sensors"),
		},
		{
			method: "GET",
			path:   "/v1.0/sensors/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllSensorsWV2, "/sensors"),
		},
		{
			method: "GET",
			path:   "/v1/edges/:edgeId/sensors",
			// swagger:route GET /v1/edges/{edgeId}/sensors EdgeGetSensors
			//
			// Get edge sensors by edge ID. ntnx:ignore
			//
			// Retrieves all sensors for an edge by edge ID {edgeId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: SensorListResponse
			//       default: APIError
			handle: makeEdgeGetAllHandle(dbAPI, dbAPI.SelectAllSensorsForEdgeW, "/edge-sensors", "edgeId"),
		},
		{
			method: "GET",
			path:   "/v1.0/edges/:edgeId/sensors",
			// swagger:route GET /v1.0/edges/{edgeId}/sensors Sensor EdgeGetSensorsV2
			//
			// Get edge sensors by edge ID. ntnx:ignore
			//
			// Retrieves all sensors for an edge by edge ID {edgeId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: SensorListResponseV2
			//       default: APIError
			handle: makeEdgeGetAllHandle(dbAPI, dbAPI.SelectAllSensorsForEdgeWV2, "/edge-sensors", "edgeId"),
		},
		{
			method: "GET",
			path:   "/v1/sensors/:id",
			// swagger:route GET /v1/sensors/{id} SensorGet
			//
			// Get sensor by ID. ntnx:ignore
			//
			// Retrieves the sensor with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: SensorGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetSensorW, "/sensors/:id", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/sensors/:id",
			// swagger:route GET /v1.0/sensors/{id} Sensor SensorGetV2
			//
			// Get sensor by ID. ntnx:ignore
			//
			// Retrieves the sensor with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: SensorGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetSensorW, "/sensors/:id", "id"),
		},
		{
			method: "DELETE",
			path:   "/v1/sensors/:id",
			// swagger:route DELETE /v1/sensors/{id} SensorDelete
			//
			// Delete sensor by ID. ntnx:ignore
			//
			// Deletes the sensor with the given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteSensorW, msgSvc, "sensor", NOTIFICATION_NONE, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/sensors/:id",
			// swagger:route DELETE /v1.0/sensors/{id} Sensor SensorDeleteV2
			//
			// Delete sensor. ntnx:ignore
			//
			// Deletes the sensor with the given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteSensorWV2, msgSvc, "sensor", NOTIFICATION_NONE, "id"),
		},
		{
			method: "POST",
			path:   "/v1/sensors",
			// swagger:route POST /v1/sensors SensorCreate
			//
			// Create sensor. ntnx:ignore
			//
			// Creates a sensor.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateSensorW, msgSvc, "sensor", NOTIFICATION_NONE),
		},
		{
			method: "POST",
			path:   "/v1.0/sensors",
			// swagger:route POST /v1.0/sensors Sensor SensorCreateV2
			//
			// Create sensor. ntnx:ignore
			//
			// Creates a sensor.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateSensorWV2, msgSvc, "sensor", NOTIFICATION_NONE),
		},
		{
			method: "PUT",
			path:   "/v1/sensors",
			// swagger:route PUT /v1/sensors SensorUpdate
			//
			// Update sensor. ntnx:ignore
			//
			// Updates a sensor.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateSensorW, msgSvc, "sensor", NOTIFICATION_NONE, ""),
		},
		{
			method: "PUT",
			path:   "/v1/sensors/:id",
			// swagger:route PUT /v1/sensors/{id} SensorUpdateV2
			//
			// Update sensor. ntnx:ignore
			//
			// Updates a sensor.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateSensorW, msgSvc, "sensor", NOTIFICATION_NONE, "id"),
		},
		{
			method: "PUT",
			path:   "/v1.0/sensors/:id",
			// swagger:route PUT /v1.0/sensors/{id} Sensor SensorUpdateV3
			//
			// Update a sensor by ID. ntnx:ignore
			//
			// Updates a sensor by ID {id}.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateSensorWV2, msgSvc, "sensor", NOTIFICATION_NONE, "id"),
		},
	}
}
