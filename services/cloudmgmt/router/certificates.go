package router

import (
	"cloudservices/cloudmgmt/api"
)

func getCertificatesRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "POST",
			path:   "/v1/certificates",
			// swagger:route POST /v1/certificates CertificatesCreate
			//
			// Create certificates. ntnx:ignore
			//
			// Certificates for devices requiring them.
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
			//       200: Certificates
			//       default: APIError
			handle: makeCreateHandle(dbAPI, dbAPI.CreateCertificatesW, msgSvc, "certificates", NOTIFICATION_NONE),
		},
		{
			method: "POST",
			path:   "/v1.0/certificates",
			// swagger:route POST /v1.0/certificates Certificate CertificatesCreateV2
			//
			// Create certificates.
			//
			// Certificates for devices requiring them.
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
			//       200: Certificates
			//       default: APIError
			handle: makeCreateHandle(dbAPI, dbAPI.CreateCertificatesW, msgSvc, "certificates", NOTIFICATION_NONE),
		},
		{
			method: "POST",
			path:   "/v1.0/certificates/",
			handle: makeCreateHandle(dbAPI, dbAPI.CreateCertificatesW, msgSvc, "certificates", NOTIFICATION_NONE),
		},
	}
}
