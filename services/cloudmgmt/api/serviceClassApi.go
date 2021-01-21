package api

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"cloudservices/common/schema"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/jmoiron/sqlx/types"
)

const (
	entityTypeServiceClass = "serviceclass"
)

var (
	// reservedServiceClassTags defines the reserved tag key to a slice of allowed values.
	// If the allowed values slice is emtpty, any value is accepted. Only one value is accepted
	reservedServiceClassTags = map[string]*TagValueProperty{
		"essential": &TagValueProperty{
			IsOptional:    true,
			AllowedValues: []string{"yes", "no"},
		},
		"category": &TagValueProperty{
			IsOptional: false,
		},
	}
)

func init() {
	queryMap["SelectServiceClasses"] = `SELECT *, count(*) OVER() as total_count from service_class_model WHERE (:id = '' OR id = :id) AND (:scope = '' OR scope = :scope) AND (:type = '' OR type = :type) AND (:svc_version = '' OR svc_version = :svc_version) AND (json_array_length(:tags) = 0 OR tags::::jsonb @> :tags)`
	queryMap["CreateServiceClass"] = `INSERT INTO service_class_model(id, name, description, type, svc_version, scope, state, min_svc_domain_version, bindable, svc_instance_create_schema, svc_instance_update_schema, svc_binding_create_schema, tags, version, created_at, updated_at) VALUES (:id, :name, :description, :type, :svc_version, :scope, :state, :min_svc_domain_version, :bindable, :svc_instance_create_schema, :svc_instance_update_schema, :svc_binding_create_schema, :tags, :version, :created_at, :updated_at)`
	queryMap["UpdateServiceClass"] = `UPDATE service_class_model SET name = :name, description = :description, type = :type, svc_version = :svc_version, scope = :scope, state = :state, min_svc_domain_version = :min_svc_domain_version, bindable = :bindable, svc_instance_create_schema = :svc_instance_create_schema, svc_instance_update_schema = :svc_instance_update_schema, svc_binding_create_schema = :svc_binding_create_schema, tags = :tags, version = :version, updated_at = :updated_at WHERE id = :id`
	queryMap["DeleteServiceClass"] = `DELETE from service_class_model WHERE id = :id`

	orderByHelper.Setup(entityTypeServiceClass, []string{"id", "name", "type", "svc_version", "scope", "state", "min_svc_domain_version", "bindable", "version", "created_at", "updated_at"})
}

// TagValueProperty defines the property of a tag value
type TagValueProperty struct {
	IsOptional    bool
	AllowedValues []string // empty means anything, only one out of values
}

// ServiceClassCommonDBO is the shared common DB model
type ServiceClassCommonDBO struct {
	Type                string `json:"type" db:"type"`
	SvcVersion          string `json:"svcVersion" db:"svc_version"`
	Scope               string `json:"scope" db:"scope"`
	MinSvcDomainVersion string `json:"minSvcDomainVersion" db:"min_svc_domain_version"`
}

// ServiceClassDBO is the DB model for Service Class
type ServiceClassDBO struct {
	ServiceClassCommonDBO
	ServiceClassSchemasDBO
	ID          string          `json:"id" db:"id"`
	Name        string          `json:"name" db:"name"`
	Description string          `json:"description" db:"description"`
	State       string          `json:"state" db:"state"`
	Bindable    bool            `json:"bindable,omitempty" db:"bindable"`
	Version     float64         `json:"version" db:"version"`
	CreatedAt   time.Time       `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time       `json:"updatedAt" db:"updated_at"`
	TotalCount  *int            `json:"totalCount,omitempty" db:"total_count"`
	Tags        *types.JSONText `json:"tags,omitempty" db:"tags"`
}

// ServiceClassSchemasDBO is the placeholder for facilitating the data conversion
type ServiceClassSchemasDBO struct {
	SvcInstanceCreateSchema *types.JSONText `json:"svcInstanceCreateSchema,omitempty" db:"svc_instance_create_schema"`
	SvcInstanceUpdateSchema *types.JSONText `json:"svcInstanceUpdateSchema,omitempty" db:"svc_instance_update_schema"`
	SvcBindingCreateSchema  *types.JSONText `json:"svcBindingCreateSchema,omitempty" db:"svc_binding_create_schema"`
}

// ServiceClassSchemas is the placeholder for facilitating the data conversion
type ServiceClassSchemas struct {
	SvcInstanceCreateSchema map[string]interface{} `json:"svcInstanceCreateSchema"`
	SvcInstanceUpdateSchema map[string]interface{} `json:"svcInstanceUpdateSchema"`
	SvcBindingCreateSchema  map[string]interface{} `json:"svcBindingCreateSchema"`
}

func (dbAPI *dbObjectModelAPI) ValidateServiceClassCommon(ctx context.Context, svcClass *model.ServiceClass) error {
	err := validateServiceClassSchemas(ctx, svcClass)
	if err != nil {
		return err
	}
	err = base.ValidateVersion(svcClass.MinSvcDomainVersion)
	if err != nil {
		return err
	}
	if svcClass.Scope != model.ServiceClassServiceDomainScope &&
		svcClass.Scope != model.ServiceClassProjectScope {
		return errcode.NewBadRequestError("scope")
	}
	if svcClass.State != model.ServiceClassFinalState &&
		svcClass.State != model.ServiceClassDraftState &&
		svcClass.State != model.ServiceClassDeprecatedState {
		return errcode.NewBadRequestError("state")
	}
	if len(svcClass.Type) == 0 {
		return errcode.NewBadRequestError("type")
	}
	if len(svcClass.SvcVersion) == 0 {
		return errcode.NewBadRequestError("svcVersion")
	}
	tagsMap := map[string][]string{}
	for _, tag := range svcClass.Tags {
		tagName := strings.ToLower(tag.Name)
		tagsMap[tagName] = append(tagsMap[tagName], tag.Value)
	}
loop:
	for tagName, tagValueProp := range reservedServiceClassTags {
		values, ok := tagsMap[tagName]
		if !ok {
			if !tagValueProp.IsOptional {
				return errcode.NewBadRequestExError("tag", fmt.Sprintf("Tag %s is required", tagName))
			}
		}
		if len(tagValueProp.AllowedValues) > 0 {
			if len(values) != 1 {
				return errcode.NewBadRequestExError("tag", fmt.Sprintf("Tag %s must have values one of %s", tagName, strings.Join(tagValueProp.AllowedValues, ", ")))
			}
			value := values[0]
			for _, allowedValue := range tagValueProp.AllowedValues {
				if strings.ToLower(value) == strings.ToLower(allowedValue) {
					continue loop
				}
			}
			return errcode.NewBadRequestExError("tag", fmt.Sprintf("Tag %s must have values one of %s", tagName, strings.Join(tagValueProp.AllowedValues, ", ")))
		}
	}

	return nil
}

func (dbAPI *dbObjectModelAPI) ValidateCreateServiceClass(ctx context.Context, svcClass *model.ServiceClass) error {
	err := dbAPI.ValidateServiceClassCommon(ctx, svcClass)
	if err != nil {
		return err
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) ValidateUpdateServiceClass(ctx context.Context, existingSvcClass *model.ServiceClass, svcClass *model.ServiceClass) error {
	err := dbAPI.ValidateServiceClassCommon(ctx, svcClass)
	if err != nil {
		return err
	}
	if existingSvcClass.MinSvcDomainVersion != svcClass.MinSvcDomainVersion {
		// Instances can break.
		// Delete and add in case it is required.
		// DB constraint will prevent if there are references
		return errcode.NewBadRequestExError("minSvcDomainVersion", "Minimum Service Domain version cannot be changed")
	}
	if existingSvcClass.Scope != svcClass.Scope {
		// It can break existing Service Instance and Class.
		// Delete and add in case it is required.
		// DB constraint will prevent if there are references
		return errcode.NewBadRequestExError("scope", "Scope cannot be updated")
	}
	if existingSvcClass.Type != svcClass.Type {
		// Type cannot be updated.
		// Delete and add in case it is required.
		// DB constraint will prevent if there are references
		return errcode.NewBadRequestExError("type", "Type cannot be updated")
	}

	return nil
}

func validateServiceClassSchemas(ctx context.Context, svcClass *model.ServiceClass) error {
	createSvcInstanceSchema := svcClass.Schemas.SvcInstance.Create.Parameters
	updateSvcInstanceSchema := svcClass.Schemas.SvcInstance.Update.Parameters
	createSvcBindingSchema := svcClass.Schemas.SvcBinding.Create.Parameters
	if createSvcInstanceSchema != nil && len(createSvcInstanceSchema) > 0 {
		err := schema.ValidateSpecMap(ctx, createSvcInstanceSchema)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Invalid Service Instance CREATE schema. Error: %s"), err.Error())
			return err
		}
	}
	if updateSvcInstanceSchema != nil && len(updateSvcInstanceSchema) > 0 {
		err := schema.ValidateSpecMap(ctx, updateSvcInstanceSchema)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Invalid Service Instance UPDATE schema. Error: %s"), err.Error())
			return err
		}
	}
	if createSvcBindingSchema != nil && len(createSvcBindingSchema) > 0 {
		err := schema.ValidateSpecMap(ctx, createSvcBindingSchema)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Invalid Service Binding CREATE schema. Error: %s"), err.Error())
			return err
		}
	}
	return nil
}

func setSchemaFieldsDBO(svcClass *model.ServiceClass, svcClassDBO *ServiceClassDBO) error {
	schemas := ServiceClassSchemas{
		SvcInstanceCreateSchema: svcClass.Schemas.SvcInstance.Create.Parameters,
		SvcInstanceUpdateSchema: svcClass.Schemas.SvcInstance.Update.Parameters,
		SvcBindingCreateSchema:  svcClass.Schemas.SvcBinding.Create.Parameters,
	}

	schemasDBO := ServiceClassSchemasDBO{}
	err := base.Convert(&schemas, &schemasDBO)
	if err != nil {
		return err
	}
	svcClassDBO.ServiceClassSchemasDBO = schemasDBO
	return nil
}

func setSchemaFields(svcClassDBO *ServiceClassDBO, svcClass *model.ServiceClass) error {
	schemas := ServiceClassSchemas{}
	err := base.Convert(svcClassDBO, &schemas)
	if err != nil {
		return err
	}
	svcClass.Schemas.SvcInstance.Create.Parameters = schemas.SvcInstanceCreateSchema
	svcClass.Schemas.SvcInstance.Update.Parameters = schemas.SvcInstanceUpdateSchema
	svcClass.Schemas.SvcBinding.Create.Parameters = schemas.SvcBindingCreateSchema
	return nil
}

func convertServiceClassDBO(svcClass *model.ServiceClass, svcClassDBO *ServiceClassDBO) error {
	err := base.Convert(svcClass, svcClassDBO)
	if err != nil {
		return err
	}
	err = setSchemaFieldsDBO(svcClass, svcClassDBO)
	if err != nil {
		return err
	}
	return nil
}

func convertServiceClass(svcClassDBO *ServiceClassDBO, svcClass *model.ServiceClass) error {
	err := base.Convert(svcClassDBO, svcClass)
	if err != nil {
		return err
	}
	err = setSchemaFields(svcClassDBO, svcClass)
	if err != nil {
		return err
	}
	return nil
}

func convertServiceClassQueryParam(queryParam *model.ServiceClassQueryParam, param *ServiceClassDBO) error {
	err := base.Convert(queryParam, param)
	if err != nil {
		return err
	}
	svcClassTags, err := queryParam.ParseTags()
	if err != nil {
		return err
	}
	jsonData, err := base.ConvertToJSON(svcClassTags)
	if err != nil {
		return err
	}
	jsonText := types.JSONText(jsonData)
	param.Tags = &jsonText
	return nil
}

// CreateServiceClass creates a Service Class in the DB
func (dbAPI *dbObjectModelAPI) CreateServiceClass(ctx context.Context, i interface{} /* *model.ServiceClass */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponseV2{}
	p, ok := i.(*model.ServiceClass)
	if !ok {
		return resp, errcode.NewInternalError("CreateServiceClass: type error")
	}
	doc := *p
	if !base.CheckID(doc.ID) {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(ctx, "CreateServiceClass doc.ID was invalid, update it to %s"), doc.ID)
	}

	err := dbAPI.ValidateCreateServiceClass(ctx, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in validation of the Service Class %s. Error: %s"), doc.ID, err.Error())
		return resp, err
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now

	svcClassDBO := ServiceClassDBO{}
	err = convertServiceClassDBO(&doc, &svcClassDBO)
	if err != nil {
		return resp, err
	}
	_, err = dbAPI.NamedExec(ctx, queryMap["CreateServiceClass"], &svcClassDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in creating Service Class %+v. Error: %s"), svcClassDBO, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	if callback != nil {
		go callback(ctx, doc)
	}
	resp.ID = doc.ID
	return resp, nil
}

// CreateServiceClassW creates a Service Class in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateServiceClassW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, dbAPI.CreateServiceClass, &model.ServiceClass{}, w, r, callback)
}

// UpdateServiceClass updates the Service Class
func (dbAPI *dbObjectModelAPI) UpdateServiceClass(ctx context.Context, i interface{} /* *model.ServiceClass */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponseV2{}
	p, ok := i.(*model.ServiceClass)
	if !ok {
		return resp, errcode.NewInternalError("UpdateServiceClass: type error")
	}
	doc := *p
	if len(doc.ID) == 0 {
		return resp, errcode.NewBadRequestError("svcClassId")
	}
	existingSvcClass, err := dbAPI.GetServiceClass(ctx, doc.ID)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in fetching existing Service Class %s. Error: %s"), doc.ID, err.Error())
		return resp, err
	}
	err = dbAPI.ValidateUpdateServiceClass(ctx, &existingSvcClass, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in validation of the Service Class %s. Error: %s"), doc.ID, err.Error())
		return resp, err
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now

	svcClassDBO := ServiceClassDBO{}
	err = convertServiceClassDBO(&doc, &svcClassDBO)
	if err != nil {
		return resp, err
	}
	_, err = dbAPI.NamedExec(ctx, queryMap["UpdateServiceClass"], &svcClassDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in updating Service Class %+v. Error: %s"), svcClassDBO, err.Error())
		return resp, err
	}
	if callback != nil {
		go callback(ctx, doc)
	}
	resp.ID = doc.ID
	return resp, nil
}

// UpdateServiceClassW updated the Service Class in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateServiceClassW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, dbAPI.UpdateServiceClass, &model.ServiceClass{}, w, r, callback)
}

// SelectAllServiceClasses returns all the Service Classes matching the query filters
func (dbAPI *dbObjectModelAPI) SelectAllServiceClasses(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParam, queryParam *model.ServiceClassQueryParam) (model.ServiceClassListPayload, error) {
	resp := model.ServiceClassListPayload{}
	query, err := buildQuery(entityTypeServiceClass, queryMap["SelectServiceClasses"], entitiesQueryParam, orderByNameID)
	if err != nil {
		return resp, err
	}
	totalCount := 0
	param := ServiceClassDBO{}
	err = convertServiceClassQueryParam(queryParam, &param)
	if err != nil {
		return resp, err
	}
	svcClasses := []model.ServiceClass{}
	_, err = dbAPI.NotPagedQuery(ctx, base.StartPageToken, base.MaxRowsLimit, func(dbObjPtr interface{}) error {
		svcClassDBO := dbObjPtr.(*ServiceClassDBO)
		svcClass := model.ServiceClass{}
		err = convertServiceClass(svcClassDBO, &svcClass)
		if err != nil {
			return err
		}
		if svcClassDBO.TotalCount != nil && totalCount == 0 {
			totalCount = *svcClassDBO.TotalCount
		}
		svcClasses = append(svcClasses, svcClass)
		return nil
	}, query, param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in fetching Service Classes. Error: %s"), err.Error())
		return resp, err
	}
	entityListResponsePayload := makeEntityListResponsePayload(entitiesQueryParam, &ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeServiceClass})
	resp.EntityListResponsePayload = entityListResponsePayload
	resp.SvcClassList = svcClasses
	return resp, nil
}

// SelectAllServiceClassesW returns all the Service Classes matching the query filters, writing the output to the writer
func (dbAPI *dbObjectModelAPI) SelectAllServiceClassesW(ctx context.Context, w io.Writer, r *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParam(r)
	queryParam := &model.ServiceClassQueryParam{}
	err := base.GetHTTPQueryParams(r, queryParam)
	if err != nil {
		return err
	}
	response, err := dbAPI.SelectAllServiceClasses(ctx, entitiesQueryParam, queryParam)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(response)
}

// GetServiceClass returns the Service Class with the given ID
func (dbAPI *dbObjectModelAPI) GetServiceClass(ctx context.Context, id string) (model.ServiceClass, error) {
	resp := model.ServiceClass{}
	if id == "" {
		return resp, errcode.NewBadRequestError("id")
	}
	tags := types.JSONText("[]")
	param := ServiceClassDBO{ID: id, Tags: &tags}
	svcClassDBOs := []ServiceClassDBO{}
	err := dbAPI.Query(ctx, &svcClassDBOs, queryMap["SelectServiceClasses"], param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting Service Class %s. Error: %s"), id, err.Error())
		return resp, err
	}
	if len(svcClassDBOs) != 1 {
		return resp, errcode.NewRecordNotFoundError("id")
	}
	svcClass := model.ServiceClass{}
	err = convertServiceClass(&svcClassDBOs[0], &svcClass)
	if err != nil {
		return resp, err
	}
	return svcClass, nil
}

// GetServiceClass returns the Service Class with the given ID, writing the output to the writer
func (dbAPI *dbObjectModelAPI) GetServiceClassW(ctx context.Context, id string, w io.Writer, req *http.Request) error {
	svcClass, err := dbAPI.GetServiceClass(ctx, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, svcClass)
}

// DeleteServiceClass delete the Service Class with the given ID
func (dbAPI *dbObjectModelAPI) DeleteServiceClass(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponseV2{}
	if id == "" {
		return resp, errcode.NewBadRequestError("id")
	}
	result, err := dbAPI.NamedExec(ctx, queryMap["DeleteServiceClass"], &ServiceClassDBO{ID: id})
	if err != nil {
		return resp, err
	}
	if base.IsDeleteSuccessful(result) {
		resp.ID = id
		if callback != nil {
			go callback(ctx, resp)
		}
	}
	return resp, nil
}

// DeleteServiceClassW deletes the Service Class with the given ID, write the response to the writer
func (dbAPI *dbObjectModelAPI) DeleteServiceClassW(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(ctx, dbAPI.DeleteServiceClass, id, w, callback)
}
