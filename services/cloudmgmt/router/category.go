package router

import (
	"cloudservices/cloudmgmt/api"
)

func getCategoryRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/categories",
			// swagger:route GET /v1/categories CategoryList
			//
			// Get all categories. ntnx:ignore
			//
			// Retrieves a list of all categories.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: CategoryListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllCategoriesW, "/categories"),
		},
		{
			method: "GET",
			path:   "/v1/categories/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllCategoriesW, "/categories"),
		},
		{
			method: "GET",
			path:   "/v1.0/categories",
			// swagger:route GET /v1.0/categories Category CategoryListV2
			//
			// Get all categories.
			//
			// Retrieves a list of all categories.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: CategoryListResponseV2
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllCategoriesWV2, "/categories"),
		},
		{
			method: "GET",
			path:   "/v1.0/categories/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllCategoriesWV2, "/categories"),
		},
		{
			method: "GET",
			path:   "/v1/categories/:id",
			// swagger:route GET /v1/categories/{id} CategoryGet
			//
			// Get a category by its ID. ntnx:ignore
			//
			// Retrieves a category with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: CategoryGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetCategoryW, "/categories/:id", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/categories/:id",
			// swagger:route GET /v1.0/categories/{id} Category CategoryGetV2
			//
			// Get a category by its ID.
			//
			// Retrieves a category with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: CategoryGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetCategoryW, "/categories/:id", "id"),
		},
		{
			method: "DELETE",
			path:   "/v1/categories/:id",
			// swagger:route DELETE /v1/categories/{id} CategoryDelete
			//
			// Delete a category by its ID. ntnx:ignore
			//
			// Delete a category with the given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteCategoryW, msgSvc, "category", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/categories/:id",
			// swagger:route DELETE /v1.0/categories/{id} Category CategoryDeleteV2
			//
			// Delete category.
			//
			// Delete the category with the given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteCategoryWV2, msgSvc, "category", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "POST",
			path:   "/v1/categories",
			// swagger:route POST /v1/categories CategoryCreate
			//
			// Create a category. ntnx:ignore
			//
			// Create a category.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateCategoryW, msgSvc, "category", NOTIFICATION_TENANT),
		},
		{
			method: "POST",
			path:   "/v1.0/categories",
			// swagger:route POST /v1.0/categories Category CategoryCreateV2
			//
			// Create a category.
			//
			// Create a category.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateCategoryWV2, msgSvc, "category", NOTIFICATION_TENANT),
		},
		{
			method: "PUT",
			path:   "/v1/categories",
			// swagger:route PUT /v1/categories CategoryUpdate
			//
			// Update a category. ntnx:ignore
			//
			// Update a category.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateCategoryW, msgSvc, "category", NOTIFICATION_TENANT, ""),
		},
		{
			method: "PUT",
			path:   "/v1/categories/:id",
			// swagger:route PUT /v1/categories/{id} CategoryUpdateV2
			//
			// Update category. ntnx:ignore
			//
			// Update a category.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateCategoryW, msgSvc, "category", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "PUT",
			path:   "/v1.0/categories/:id",
			// swagger:route PUT /v1.0/categories/{id} Category CategoryUpdateV3
			//
			// Update a category by its ID.
			//
			// Update a category with the given ID {id}.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateCategoryWV2, msgSvc, "category", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/categoriesusage",
			// swagger:route GET /v1.0/categoriesusage Category CategoryUsageList
			//
			// Get all categories usage. ntnx:ignore
			//
			// Retrieves a list of all categories usage.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: CategoryUsageListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllCategoriesUsageInfoW, "/categoriesusage"),
		},
		{
			method: "GET",
			path:   "/v1.0/categoriesusage/:id",
			// swagger:route GET /v1.0/categoriesusage/{id} Category CategoryUsageGet
			//
			// Get detailed usage of a category by its ID. ntnx:ignore
			//
			// Retrieves detailed usage of a category with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: CategoryUsageGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetCategoryDetailUsageInfoW, "/categoriesusage/:id", "id"),
		},
	}
}
