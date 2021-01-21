package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/model"
)

func getWstunRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {

	SetupSSHTunnelingHandle := makeCustomMessageHandle(dbAPI, dbAPI.SetupSSHTunnelingW, msgSvc, "setupSSHTunneling", NOTIFICATION_EDGE, func(doc interface{}) *string {
		payload := doc.(*model.WstunPayload)
		return &payload.ServiceDomainID
	})

	TeardownSSHTunnelingHandle := makeCustomMessageHandle(dbAPI, dbAPI.TeardownSSHTunnelingW, msgSvc, "teardownSSHTunneling", NOTIFICATION_EDGE, func(doc interface{}) *string {
		payload := doc.(*model.WstunTeardownRequest)
		return &payload.ServiceDomainID
	})

	return []routeHandle{
		{
			method: "POST",
			path:   "/v1.0/setupsshtunneling",
			// swagger:route POST /v1.0/setupsshtunneling SSH SetupSSHTunneling
			//
			// Configure SSH tunneling to the service domain.
			//
			// Configure SSH tunneling to the service domain.
			// Requirements to use this feature:
			// Minimum service domain version of 1.15.0.
			// Remote SSH tunneling feature and CLI access are enabled per account.
			// Service domain profile has SSH enabled.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: SetupSSHTunnelingResponse
			//       default: APIError
			handle: SetupSSHTunnelingHandle,
		},
		{
			method: "POST",
			path:   "/v1.0/teardownsshtunneling",
			// swagger:route POST /v1.0/teardownsshtunneling SSH TeardownSSHTunneling
			//
			// Disable service domain SSH tunneling.
			//
			// Shut down SSH tunneling to the service domain. Disables SSH tunneling, including current open sessions.
			//
			//     Consumes:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: TeardownSSHTunnelingResponse
			//       default: APIError
			handle: TeardownSSHTunnelingHandle,
		},
	}
}
