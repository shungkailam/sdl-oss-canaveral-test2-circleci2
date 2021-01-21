package api

import (
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/meta"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/golang/glog"
	funk "github.com/thoas/go-funk"
)

const entityTypeCategory = "category"

// CategoryDBO is the DB model for Category
type CategoryDBO struct {
	// required: true
	model.BaseModelDBO
	Name    string `json:"name" db:"name"`
	Purpose string `json:"purpose" db:"purpose"`
}

// CategoryValueDBO is the DB model
type CategoryValueDBO struct {
	ID         int64  `json:"id" db:"id"`
	CategoryID string `json:"categoryId" db:"category_id"`
	Value      string `json:"value" db:"value"`
}
type CategoryIdsParam struct {
	CategoryIDs []string `json:"categoryIds" db:"category_ids"`
}

type CategoryUseDBO struct {
	ID         string `db:"id"`
	CategoryID string `db:"category_id"`
}

type categoryValueUseDBO struct {
	ID    string `db:"id"`
	Value string `db:"value"`
}

func init() {
	queryMap["SelectCategoriesTemplate"] = `SELECT * FROM category_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) %s`
	queryMap["SelectCategoriesByIDsTemplate"] = `SELECT *, count(*) OVER() as total_count FROM category_model WHERE tenant_id = :tenant_id %s`
	queryMap["SelectCategoryValues"] = `SELECT * FROM category_value_model WHERE category_id = :category_id AND (:value = '' OR value = :value) ORDER BY ID`
	queryMap["SelectCategoriesValues"] = `SELECT * FROM category_value_model WHERE category_id IN (:category_ids) ORDER BY ID`

	queryMap["CreateCategory"] = `INSERT INTO category_model (id, version, tenant_id, name, purpose, created_at, updated_at) VALUES (:id, :version, :tenant_id, :name, :purpose, :created_at, :updated_at)`
	queryMap["CreateCategoryValue"] = `INSERT INTO category_value_model (category_id, value) VALUES (:category_id, :value)`
	queryMap["UpdateCategory"] = `UPDATE category_model SET version = :version, name = :name, purpose = :purpose, updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`

	queryMap["SelectCategoriesIDsTemplate"] = `SELECT id from category_model WHERE tenant_id = '%s'`
	queryMap["SelectCategoriesEdgesTemplate"] = `SELECT DISTINCT e.edge_id AS id, c.category_id FROM edge_label_model AS e INNER JOIN category_value_model AS c ON e.category_value_id = c.id WHERE c.category_id IN (SELECT id FROM category_model WHERE tenant_id = '%s')`
	queryMap["SelectCategoriesProjectsTemplate"] = `SELECT DISTINCT e.project_id AS id, c.category_id FROM project_edge_selector_model AS e INNER JOIN category_value_model AS c ON e.category_value_id = c.id WHERE c.category_id IN (SELECT id FROM category_model WHERE tenant_id = '%s')`
	queryMap["SelectCategoriesApplicationsTemplate"] = `SELECT DISTINCT e.application_id AS id, c.category_id FROM application_edge_selector_model AS e INNER JOIN category_value_model AS c ON e.category_value_id = c.id WHERE c.category_id IN (SELECT id FROM category_model WHERE tenant_id = '%s')`
	queryMap["SelectCategoriesApplicationsOriginTemplate"] = `SELECT DISTINCT e.application_id AS id, c.category_id FROM application_origin_model AS e INNER JOIN category_value_model AS c ON e.category_value_id = c.id WHERE c.category_id IN (SELECT id FROM category_model WHERE tenant_id = '%s')`
	queryMap["SelectCategoriesDataPipelinesTemplate"] = `SELECT DISTINCT e.data_stream_id AS id, c.category_id FROM data_stream_origin_model AS e INNER JOIN category_value_model AS c ON e.category_value_id = c.id WHERE c.category_id IN (SELECT id FROM category_model WHERE tenant_id = '%s')`
	queryMap["SelectCategoriesDataSourcesTemplate"] = `SELECT DISTINCT e.data_source_id AS id, c.category_id FROM data_source_field_selector_model AS e INNER JOIN category_value_model AS c ON e.category_value_id = c.id WHERE c.category_id IN (SELECT id FROM category_model WHERE tenant_id = '%s')`

	queryMap["SelectCategoryValueEdgesTemplate"] = `SELECT e.edge_id AS id, c.value FROM edge_label_model AS e INNER JOIN category_value_model AS c ON e.category_value_id = c.id WHERE c.category_id = '%s'`
	queryMap["SelectCategoryValueProjectsTemplate"] = `SELECT e.project_id AS id, c.value FROM project_edge_selector_model AS e INNER JOIN category_value_model AS c ON e.category_value_id = c.id WHERE c.category_id = '%s'`
	queryMap["SelectCategoryValueApplicationsTemplate"] = `SELECT e.application_id AS id, c.value FROM application_edge_selector_model AS e INNER JOIN category_value_model AS c ON e.category_value_id = c.id WHERE c.category_id = '%s'`
	queryMap["SelectCategoryValueApplicationsOriginTemplate"] = `SELECT e.application_id AS id, c.value FROM application_origin_model AS e INNER JOIN category_value_model AS c ON e.category_value_id = c.id WHERE c.category_id = '%s'`
	queryMap["SelectCategoryValueDataPipelinesTemplate"] = `SELECT e.data_stream_id AS id, c.value FROM data_stream_origin_model AS e INNER JOIN category_value_model AS c ON e.category_value_id = c.id WHERE c.category_id = '%s'`
	// DISTINCT needed here for there are multiple fields in a data source
	queryMap["SelectCategoryValueDataSourcesTemplate"] = `SELECT DISTINCT e.data_source_id AS id, c.value FROM data_source_field_selector_model AS e INNER JOIN category_value_model AS c ON e.category_value_id = c.id WHERE c.category_id = '%s'`

	orderByHelper.Setup(entityTypeCategory, []string{"id", "version", "created_at", "updated_at", "name", "purpose"})
}

func (dbAPI *dbObjectModelAPI) populateCategoriesValues(ctx context.Context, categories []model.Category) error {
	if len(categories) == 0 {
		return nil
	}
	categoryValueDBOs := []CategoryValueDBO{}
	categoryIDs := funk.Map(categories, func(category model.Category) string { return category.ID }).([]string)
	err := dbAPI.QueryIn(ctx, &categoryValueDBOs, queryMap["SelectCategoriesValues"], CategoryIdsParam{
		CategoryIDs: categoryIDs,
	})
	if err != nil {
		return err
	}
	categoryValuesMap := map[string]([]string){}
	for _, categoryValueDBO := range categoryValueDBOs {
		categoryValuesMap[categoryValueDBO.CategoryID] = append(categoryValuesMap[categoryValueDBO.CategoryID], categoryValueDBO.Value)
	}
	for i := 0; i < len(categories); i++ {
		categories[i].Values = categoryValuesMap[categories[i].ID]
	}
	return nil
}

/*** Start of common shared private methods for categories ***/

func (dbAPI *dbObjectModelAPI) getCategoriesEtag(ctx context.Context, etag string, categoryID string, queryParam *model.EntitiesQueryParamV1) ([]model.Category, error) {
	categories := []model.Category{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return categories, err
	}
	tenantID := authContext.TenantID
	categoryDBOs := []CategoryDBO{}
	baseModel := model.BaseModelDBO{TenantID: tenantID, ID: categoryID}
	param := CategoryDBO{BaseModelDBO: baseModel}
	query, err := buildQuery(entityTypeCategory, queryMap["SelectCategoriesTemplate"], queryParam, orderByNameID)
	if err != nil {
		return categories, err
	}
	err = dbAPI.Query(ctx, &categoryDBOs, query, param)
	if err != nil {
		return categories, err
	}
	// if len(etag) > 0 {
	// 	if handled, err := handleEtag(w, etag, categories); handled {
	// 		return categories, err
	// 	}
	// }
	for _, categoryDBO := range categoryDBOs {
		category := model.Category{}
		err = base.Convert(&categoryDBO, &category)
		if err != nil {
			return categories, err
		}
		categories = append(categories, category)
	}
	err = dbAPI.populateCategoriesValues(ctx, categories)
	return categories, err
}

// internal API used by SelectAllCategoriesWV2
func (dbAPI *dbObjectModelAPI) getCategoriesForQuery(context context.Context, entitiesQueryParam *model.EntitiesQueryParam) ([]model.Category, int, error) {
	categories := []model.Category{}
	categoryDBOs := []CategoryDBO{}
	err := dbAPI.getEntities(context, entityTypeCategory, queryMap["SelectCategoriesByIDsTemplate"], entitiesQueryParam, &categoryDBOs)
	if err != nil {
		return categories, 0, err
	}
	if len(categoryDBOs) == 0 {
		return categories, 0, nil
	}

	first := true
	totalCount := 0
	for _, categoryDBO := range categoryDBOs {
		if first {
			first = false
			if categoryDBO.TotalCount != nil {
				totalCount = *categoryDBO.TotalCount
			}
		}
		category := model.Category{}
		err := base.Convert(&categoryDBO, &category)
		if err != nil {
			return []model.Category{}, 0, err
		}
		categories = append(categories, category)
	}
	err = dbAPI.populateCategoriesValues(context, categories)
	return categories, totalCount, err
}

// internal api for old public W apis
func (dbAPI *dbObjectModelAPI) getCategoriesW(context context.Context, categoryID string, w io.Writer, req *http.Request) error {
	etag := getEtag(req)
	queryParam := model.GetEntitiesQueryParamV1(req)
	categories, err := dbAPI.getCategoriesEtag(context, etag, categoryID, queryParam)
	if err != nil {
		return err
	}
	if categoryID != "" {
		if len(categories) == 0 {
			return errcode.NewRecordNotFoundError(categoryID)
		}
		return json.NewEncoder(w).Encode(categories[0])
	}
	return base.DispatchPayload(w, categories)
}

func (dbAPI *dbObjectModelAPI) getCategoryValueDBOs(ctx context.Context, param CategoryValueDBO) ([]CategoryValueDBO, error) {
	categoryValueDBOs := []CategoryValueDBO{}
	err := dbAPI.Query(ctx, &categoryValueDBOs, queryMap["SelectCategoryValues"], param)
	return categoryValueDBOs, err
}

func (dbAPI *dbObjectModelAPI) getCategoryValueDBOsByCategoryIds(ctx context.Context, ids []string) ([]CategoryValueDBO, error) {
	categoryValueDBOs := []CategoryValueDBO{}
	err := dbAPI.QueryIn(ctx, &categoryValueDBOs, queryMap["SelectCategoriesValues"], CategoryIdsParam{CategoryIDs: ids})
	if err != nil {
		return nil, err
	}
	return categoryValueDBOs, nil
}

/*** End of common shared private methods for categories ***/

/** Start of public APIs */

// SelectAllCategories returns all the categories for the tenant ID
func (dbAPI *dbObjectModelAPI) SelectAllCategories(context context.Context, queryParam *model.EntitiesQueryParamV1) ([]model.Category, error) {
	return dbAPI.getCategoriesEtag(context, "", "", queryParam)
}

// SelectAllCategoriesW select all categories for the tenant in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllCategoriesW(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getCategoriesW(context, "", w, req)
}

// SelectAllCategoriesWV2 select all categories for the tenant in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllCategoriesWV2(context context.Context, w io.Writer, req *http.Request) error {
	queryParam := model.GetEntitiesQueryParam(req)
	categories, totalCount, err := dbAPI.getCategoriesForQuery(context, queryParam)
	if err != nil {
		return err
	}
	queryInfo := ListQueryInfo{EntityType: entityTypeCategory, TotalCount: totalCount}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &queryInfo)
	r := model.CategoryListResponsePayload{
		EntityListResponsePayload: entityListResponsePayload,
		CategoryList:              categories,
	}
	return json.NewEncoder(w).Encode(r)
}

// GetCategory get a category object for the tenant in the DB
func (dbAPI *dbObjectModelAPI) GetCategory(context context.Context, categoryID string) (model.Category, error) {
	if len(categoryID) == 0 {
		return model.Category{}, errcode.NewBadRequestError("categoryID")
	}
	categories, err := dbAPI.getCategoriesEtag(context, "", categoryID, nil)

	if err != nil {
		return model.Category{}, err
	}
	if len(categories) == 0 {
		return model.Category{}, errcode.NewRecordNotFoundError(categoryID)
	}
	return categories[0], err
}

// GetCategoryW get a category object for the tenant in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetCategoryW(context context.Context, categoryID string, w io.Writer, req *http.Request) error {
	if len(categoryID) == 0 {
		return errcode.NewBadRequestError("categoryID")
	}
	return dbAPI.getCategoriesW(context, categoryID, w, req)
}

// CreateCategory creates a category object for the tenant in the DB
func (dbAPI *dbObjectModelAPI) CreateCategory(context context.Context, i interface{} /* *model.Category */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.Category)
	if !ok {
		return resp, errcode.NewInternalError("CreateCategory: type error")
	}
	doc := *p

	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if base.CheckID(doc.ID) {
		glog.Infof(base.PrefixRequestID(context, "CreateCategory doc.ID was %s\n"), doc.ID)
	} else {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateCategory doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityCategory,
		meta.OperationCreate,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}

	now := base.RoundedNow()
	doc.Version = float64(now.UnixNano())
	doc.CreatedAt = now
	doc.UpdatedAt = now

	err = model.ValidateCategory(&doc)
	if err != nil {
		return resp, err
	}

	categoryDBO := CategoryDBO{}
	err = base.Convert(&doc, &categoryDBO)
	if err != nil {
		return resp, err
	}
	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		_, err := tx.NamedExec(context, queryMap["CreateCategory"], &categoryDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error creating category %+v. Error: %s"), categoryDBO, err.Error())
			return errcode.TranslateDatabaseError(categoryDBO.ID, err)
		}
		for _, value := range doc.Values {
			// The DB ID is generated
			categoryValueDBO := CategoryValueDBO{CategoryID: doc.ID, Value: value}
			_, err = tx.NamedExec(context, queryMap["CreateCategoryValue"], &categoryValueDBO)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(context, "Error creating category value %+v. Error: %s"), categoryValueDBO, err.Error())
				return errcode.TranslateDatabaseError(categoryDBO.ID, err)
			}
		}
		return nil
	})
	if err != nil {
		return resp, err
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addCategoryAuditLog(dbAPI, context, doc, CREATE)
	return resp, nil
}

// CreateCategoryW creates a category object for the tenant in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateCategoryW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateCategory, &model.Category{}, w, r, callback)
}

// CreateCategoryWV2 creates a category object for the tenant in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateCategoryWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateCategory), &model.Category{}, w, r, callback)
}

// UpdateCategory updates a category object for the tenant in the DB. It allows delete/insert of non-referred category values.
func (dbAPI *dbObjectModelAPI) UpdateCategory(context context.Context, i interface{} /* *model.Category */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.Category)
	if !ok {
		return resp, errcode.NewInternalError("UpdateCategory: type error")
	}
	if authContext.ID != "" {
		p.ID = authContext.ID
	}
	if p.ID == "" {
		return resp, errcode.NewBadRequestError("ID")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	err = auth.CheckRBAC(
		authContext,
		meta.EntityCategory,
		meta.OperationUpdate,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}
	now := base.RoundedNow()
	doc.Version = float64(now.UnixNano())
	doc.UpdatedAt = now

	err = model.ValidateCategory(&doc)
	if err != nil {
		return resp, err
	}

	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		categoryDBO := CategoryDBO{}
		err := base.Convert(&doc, &categoryDBO)
		if err != nil {
			return err
		}
		categoryValuesMap := map[string]CategoryValueDBO{}
		categoryValueDBOs := []CategoryValueDBO{}
		err = base.QueryTxn(context, tx, &categoryValueDBOs, queryMap["SelectCategoryValues"], CategoryValueDBO{CategoryID: doc.ID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error quering category values for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		_, err = tx.NamedExec(context, queryMap["UpdateCategory"], &categoryDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in updating category for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		for _, categoryValueDBO := range categoryValueDBOs {
			categoryValuesMap[categoryValueDBO.Value] = categoryValueDBO
		}
		for _, value := range doc.Values {
			_, ok := categoryValuesMap[value]
			if ok {
				delete(categoryValuesMap, value)
			} else {
				// Insert
				categoryValueDBO := CategoryValueDBO{CategoryID: doc.ID, Value: value}
				_, err = tx.NamedExec(context, queryMap["CreateCategoryValue"], &categoryValueDBO)
				if err != nil {
					glog.Errorf(base.PrefixRequestID(context, "Error in creating category values for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
					return errcode.TranslateDatabaseError(doc.ID, err)
				}
			}
		}
		// Delete the remaining
		for _, categoryValueDBO := range categoryValuesMap {
			_, err := base.DeleteTxn(context, tx, "category_value_model", map[string]interface{}{"id": categoryValueDBO.ID})
			if err != nil {
				glog.Errorf(base.PrefixRequestID(context, "Error in deleting category values for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
				return err
			}
		}
		return nil
	})
	if err != nil {
		return resp, err
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addCategoryAuditLog(dbAPI, context, doc, UPDATE)
	return resp, nil
}

// UpdateCategoryW updates a category object for the tenant in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateCategoryW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateCategory, &model.Category{}, w, r, callback)
}

// UpdateCategoryWV2 updates a category object for the tenant in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) UpdateCategoryWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateCategory), &model.Category{}, w, r, callback)
}

// DeleteCategory delete a category object for the tenant in the DB
func (dbAPI *dbObjectModelAPI) DeleteCategory(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	category, errGetCategory := dbAPI.GetCategory(context, id)
	err = auth.CheckRBAC(
		authContext,
		meta.EntityCategory,
		meta.OperationDelete,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}
	doc := model.Category{
		BaseModel: model.BaseModel{
			TenantID: authContext.TenantID,
			ID:       id,
		},
	}
	result, err := DeleteEntity(context, dbAPI, "category_model", "id", id, doc, callback)
	if err == nil {
		if errGetCategory != nil {
			glog.Error("Error in getting category : ", errGetCategory.Error())
		} else {
			GetAuditlogHandler().addCategoryAuditLog(dbAPI, context, category, DELETE)
		}
	}
	return result, err
}

// DeleteCategoryW delete a category object for the tenant in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteCategoryW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteCategory, id, w, callback)
}

// DeleteCategoryWV2 delete a category object for the tenant in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteCategoryWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteCategory), id, w, callback)
}

// SelectAllCategoriesUsageInfoW select all categories usage info for the tenant in the DB
func (dbAPI *dbObjectModelAPI) SelectAllCategoriesUsageInfo(ctx context.Context) ([]model.CategoryUsageInfo, error) {
	result := []model.CategoryUsageInfo{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return result, err
	}
	if !auth.IsInfraAdminRole(authContext) {
		return result, errcode.NewPermissionDeniedError("RBAC")
	}
	tenantID := authContext.TenantID
	allCatMap := map[string]struct{}{}
	categoryMap := map[string]struct{}{}
	categoryEdgesMap := map[string][]string{}
	categoryProjectsMap := map[string][]string{}
	catAppMap := map[string]map[string]struct{}{}
	categoryApplicationsMap := map[string][]string{}
	categoryDataPipelinesMap := map[string][]string{}
	categoryDataSourcesMap := map[string][]string{}

	categoryUseDBOs := []CategoryUseDBO{}
	catQuery := fmt.Sprintf(queryMap["SelectCategoriesIDsTemplate"], tenantID)
	err = dbAPI.Query(ctx, &categoryUseDBOs, catQuery, struct{}{})
	if err != nil {
		return result, err
	}
	for _, categoryUseDBO := range categoryUseDBOs {
		allCatMap[categoryUseDBO.ID] = struct{}{}
	}

	categoryUseDBOs = []CategoryUseDBO{}
	edgeQuery := fmt.Sprintf(queryMap["SelectCategoriesEdgesTemplate"], tenantID)
	err = dbAPI.Query(ctx, &categoryUseDBOs, edgeQuery, struct{}{})
	if err != nil {
		return result, err
	}
	for _, categoryUseDBO := range categoryUseDBOs {
		categoryMap[categoryUseDBO.CategoryID] = struct{}{}
		categoryEdgesMap[categoryUseDBO.CategoryID] = append(categoryEdgesMap[categoryUseDBO.CategoryID], categoryUseDBO.ID)
	}
	categoryUseDBOs = []CategoryUseDBO{}
	projectQuery := fmt.Sprintf(queryMap["SelectCategoriesProjectsTemplate"], tenantID)
	err = dbAPI.Query(ctx, &categoryUseDBOs, projectQuery, struct{}{})
	if err != nil {
		return result, err
	}
	for _, categoryUseDBO := range categoryUseDBOs {
		categoryMap[categoryUseDBO.CategoryID] = struct{}{}
		categoryProjectsMap[categoryUseDBO.CategoryID] = append(categoryProjectsMap[categoryUseDBO.CategoryID], categoryUseDBO.ID)
	}
	categoryUseDBOs = []CategoryUseDBO{}
	applicationQuery := fmt.Sprintf(queryMap["SelectCategoriesApplicationsTemplate"], tenantID)
	err = dbAPI.Query(ctx, &categoryUseDBOs, applicationQuery, struct{}{})
	if err != nil {
		return result, err
	}
	for _, categoryUseDBO := range categoryUseDBOs {
		categoryMap[categoryUseDBO.CategoryID] = struct{}{}
		categoryApplicationsMap[categoryUseDBO.CategoryID] = append(categoryApplicationsMap[categoryUseDBO.CategoryID], categoryUseDBO.ID)
		m := catAppMap[categoryUseDBO.CategoryID]
		if m == nil {
			m = map[string]struct{}{}
			catAppMap[categoryUseDBO.CategoryID] = m
		}
		m[categoryUseDBO.ID] = struct{}{}
	}
	if *config.Cfg.EnableAppOriginSelectors {
		categoryUseDBOs = []CategoryUseDBO{}
		applicationOriginQuery := fmt.Sprintf(queryMap["SelectCategoriesApplicationsOriginTemplate"], tenantID)
		err = dbAPI.Query(ctx, &categoryUseDBOs, applicationOriginQuery, struct{}{})
		if err != nil {
			return result, err
		}
		for _, categoryUseDBO := range categoryUseDBOs {
			categoryMap[categoryUseDBO.CategoryID] = struct{}{}
			if _, b := catAppMap[categoryUseDBO.CategoryID][categoryUseDBO.ID]; !b {
				categoryApplicationsMap[categoryUseDBO.CategoryID] = append(categoryApplicationsMap[categoryUseDBO.CategoryID], categoryUseDBO.ID)
			}
		}
	}

	categoryUseDBOs = []CategoryUseDBO{}
	dataPipelineQuery := fmt.Sprintf(queryMap["SelectCategoriesDataPipelinesTemplate"], tenantID)
	err = dbAPI.Query(ctx, &categoryUseDBOs, dataPipelineQuery, struct{}{})
	if err != nil {
		return result, err
	}
	for _, categoryUseDBO := range categoryUseDBOs {
		categoryMap[categoryUseDBO.CategoryID] = struct{}{}
		categoryDataPipelinesMap[categoryUseDBO.CategoryID] = append(categoryDataPipelinesMap[categoryUseDBO.CategoryID], categoryUseDBO.ID)
	}
	categoryUseDBOs = []CategoryUseDBO{}
	dataSourceQuery := fmt.Sprintf(queryMap["SelectCategoriesDataSourcesTemplate"], tenantID)
	err = dbAPI.Query(ctx, &categoryUseDBOs, dataSourceQuery, struct{}{})
	if err != nil {
		return result, err
	}
	for _, categoryUseDBO := range categoryUseDBOs {
		categoryMap[categoryUseDBO.CategoryID] = struct{}{}
		categoryDataSourcesMap[categoryUseDBO.CategoryID] = append(categoryDataSourcesMap[categoryUseDBO.CategoryID], categoryUseDBO.ID)
	}

	result = make([]model.CategoryUsageInfo, 0, len(allCatMap))
	for id := range categoryMap {
		delete(allCatMap, id)
		usage := model.CategoryUsageInfo{
			ID: id,
			CategoryUsage: model.CategoryUsage{
				EdgeIDs:         NilToEmptyStrings(categoryEdgesMap[id]),
				ProjectIDs:      NilToEmptyStrings(categoryProjectsMap[id]),
				ApplicationIDs:  NilToEmptyStrings(categoryApplicationsMap[id]),
				DataPipelineIDs: NilToEmptyStrings(categoryDataPipelinesMap[id]),
				DataSourceIDs:   NilToEmptyStrings(categoryDataSourcesMap[id]),
			},
		}
		result = append(result, usage)
	}
	for id := range allCatMap {
		usage := model.CategoryUsageInfo{
			ID:            id,
			CategoryUsage: *model.NewEmptyCategoryUsage(),
		}
		result = append(result, usage)
	}
	return result, nil
}

// SelectAllCategoriesUsageInfoW select all categories usage info for the tenant in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllCategoriesUsageInfoW(context context.Context, w io.Writer, req *http.Request) error {
	results, err := dbAPI.SelectAllCategoriesUsageInfo(context)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, results)
}

// GetCategoryDetailUsageInfo get detail usage info for category with the given id
func (dbAPI *dbObjectModelAPI) GetCategoryDetailUsageInfo(ctx context.Context, categoryID string) (model.CategoryDetailUsageInfo, error) {
	result := model.CategoryDetailUsageInfo{
		ID:       categoryID,
		UsageMap: map[string]*model.CategoryUsage{},
	}
	category, err := dbAPI.GetCategory(ctx, categoryID)
	if err != nil {
		return result, err
	}
	if len(category.Values) == 0 {
		return result, nil
	}
	for _, value := range category.Values {
		result.UsageMap[value] = nil
	}

	var categoryValueUseDBOs []categoryValueUseDBO
	var query string

	categoryValueUseDBOs = []categoryValueUseDBO{}
	query = fmt.Sprintf(queryMap["SelectCategoryValueEdgesTemplate"], categoryID)
	err = dbAPI.Query(ctx, &categoryValueUseDBOs, query, struct{}{})
	if err != nil {
		return result, err
	}
	for _, categoryValueUse := range categoryValueUseDBOs {
		pcu := result.UsageMap[categoryValueUse.Value]
		if pcu == nil {
			pcu = model.NewEmptyCategoryUsage()
			result.UsageMap[categoryValueUse.Value] = pcu
		}
		pcu.EdgeIDs = append(pcu.EdgeIDs, categoryValueUse.ID)
	}

	categoryValueUseDBOs = []categoryValueUseDBO{}
	query = fmt.Sprintf(queryMap["SelectCategoryValueProjectsTemplate"], categoryID)
	err = dbAPI.Query(ctx, &categoryValueUseDBOs, query, struct{}{})
	if err != nil {
		return result, err
	}
	for _, categoryValueUse := range categoryValueUseDBOs {
		pcu := result.UsageMap[categoryValueUse.Value]
		if pcu == nil {
			pcu = model.NewEmptyCategoryUsage()
			result.UsageMap[categoryValueUse.Value] = pcu
		}
		pcu.ProjectIDs = append(pcu.ProjectIDs, categoryValueUse.ID)
	}

	categoryValueUseDBOs = []categoryValueUseDBO{}
	query = fmt.Sprintf(queryMap["SelectCategoryValueApplicationsTemplate"], categoryID)
	err = dbAPI.Query(ctx, &categoryValueUseDBOs, query, struct{}{})
	if err != nil {
		return result, err
	}
	for _, categoryValueUse := range categoryValueUseDBOs {
		pcu := result.UsageMap[categoryValueUse.Value]
		if pcu == nil {
			pcu = model.NewEmptyCategoryUsage()
			result.UsageMap[categoryValueUse.Value] = pcu
		}
		pcu.ApplicationIDs = append(pcu.ApplicationIDs, categoryValueUse.ID)
	}

	if *config.Cfg.EnableAppOriginSelectors {
		categoryValueUseDBOs = []categoryValueUseDBO{}
		query = fmt.Sprintf(queryMap["SelectCategoryValueApplicationsOriginTemplate"], categoryID)
		err = dbAPI.Query(ctx, &categoryValueUseDBOs, query, struct{}{})
		if err != nil {
			return result, err
		}
		for _, categoryValueUse := range categoryValueUseDBOs {
			pcu := result.UsageMap[categoryValueUse.Value]
			if pcu == nil {
				pcu = model.NewEmptyCategoryUsage()
				result.UsageMap[categoryValueUse.Value] = pcu
			}
			if !funk.Contains(pcu.ApplicationIDs, categoryValueUse.ID) {
				pcu.ApplicationIDs = append(pcu.ApplicationIDs, categoryValueUse.ID)
			}
		}
	}

	categoryValueUseDBOs = []categoryValueUseDBO{}
	query = fmt.Sprintf(queryMap["SelectCategoryValueDataPipelinesTemplate"], categoryID)
	err = dbAPI.Query(ctx, &categoryValueUseDBOs, query, struct{}{})
	if err != nil {
		return result, err
	}
	for _, categoryValueUse := range categoryValueUseDBOs {
		pcu := result.UsageMap[categoryValueUse.Value]
		if pcu == nil {
			pcu = model.NewEmptyCategoryUsage()
			result.UsageMap[categoryValueUse.Value] = pcu
		}
		pcu.DataPipelineIDs = append(pcu.DataPipelineIDs, categoryValueUse.ID)
	}

	categoryValueUseDBOs = []categoryValueUseDBO{}
	query = fmt.Sprintf(queryMap["SelectCategoryValueDataSourcesTemplate"], categoryID)
	err = dbAPI.Query(ctx, &categoryValueUseDBOs, query, struct{}{})
	if err != nil {
		return result, err
	}
	for _, categoryValueUse := range categoryValueUseDBOs {
		pcu := result.UsageMap[categoryValueUse.Value]
		if pcu == nil {
			pcu = model.NewEmptyCategoryUsage()
			result.UsageMap[categoryValueUse.Value] = pcu
		}
		pcu.DataSourceIDs = append(pcu.DataSourceIDs, categoryValueUse.ID)
	}

	return result, nil
}

// GetCategoryDetailUsageInfo get detail usage info for category with the given id, write output into writer
func (dbAPI *dbObjectModelAPI) GetCategoryDetailUsageInfoW(ctx context.Context, categoryID string, w io.Writer, req *http.Request) error {
	usageInfo, err := dbAPI.GetCategoryDetailUsageInfo(ctx, categoryID)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, usageInfo)
}

// GetBuiltinCategoryID creates the builtin category ID.
// TODO replace with MD5 hash later
func GetBuiltinCategoryID(tenantID string, suffix string) string {
	return fmt.Sprintf("%s-%s", tenantID, suffix)
}

func createBuiltinCategories(ctx context.Context, tx *base.WrappedTx, tenantID string) error {
	now := base.RoundedNow()
	for _, category := range config.BuiltinCategories {
		category.ID = GetBuiltinCategoryID(tenantID, category.ID)
		category.TenantID = tenantID
		category.Version = float64(now.UnixNano())
		category.CreatedAt = now
		category.UpdatedAt = now
		categoryDBO := CategoryDBO{}
		err := base.Convert(&category, &categoryDBO)
		if err != nil {
			return err
		}
		_, err = tx.NamedExec(ctx, queryMap["CreateCategory"], &categoryDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error creating category %+v. Error: %s"), categoryDBO, err.Error())
			return errcode.TranslateDatabaseError(categoryDBO.ID, err)
		}
		for _, value := range category.Values {
			// The DB ID is generated
			categoryValueDBO := CategoryValueDBO{CategoryID: category.ID, Value: value}
			_, err = tx.NamedExec(ctx, queryMap["CreateCategoryValue"], &categoryValueDBO)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error creating category value %+v. Error: %s"), categoryValueDBO, err.Error())
				return errcode.TranslateDatabaseError(categoryDBO.ID, err)
			}
		}
	}
	return nil
}

func deleteBuiltinCategories(ctx context.Context, tx *base.WrappedTx, tenantID string) error {
	for _, category := range config.BuiltinCategories {
		id := GetBuiltinCategoryID(tenantID, category.ID)
		_, err := base.DeleteTxn(ctx, tx, "category_model", map[string]interface{}{"id": id})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Deleting builtIn category %+v with id %s. Error: %s"), category, id, err.Error())
			return err
		}
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) getCategoriesByIDs(ctx context.Context, categoryIDs []string) ([]model.Category, error) {
	categories := []model.Category{}
	if len(categoryIDs) == 0 {
		return categories, nil
	}

	categoryDBOs := []CategoryDBO{}
	if err := dbAPI.queryEntitiesByTenantAndIds(ctx, &categoryDBOs, "category_model", categoryIDs); err != nil {
		return nil, err
	}

	for _, categoryDBO := range categoryDBOs {
		category := model.Category{}
		if err := base.Convert(&categoryDBO, &category); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	err := dbAPI.populateCategoriesValues(ctx, categories)
	return categories, err
}

func (dbAPI *dbObjectModelAPI) GetCategoryNamesByIDs(ctx context.Context, categoryIDs []string) (map[string]string, error) {
	return dbAPI.getNamesByIDs(ctx, "category_model", categoryIDs)
}
