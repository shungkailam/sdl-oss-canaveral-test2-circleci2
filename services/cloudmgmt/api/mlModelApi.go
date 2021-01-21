package api

import (
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/meta"
	"cloudservices/common/model"
	"cloudservices/common/service"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	"google.golang.org/grpc"

	gapi "cloudservices/ai/generated/golang/grpc"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const (
	entityTypeMLModel                = "mlmodel"
	mlVersion                        = "v1"
	modelURLDefaultExpirationMinutes = 15
)

// MLModelDBO is DB object model for ML Model
type MLModelDBO struct {
	model.BaseModelDBO
	Name          string `json:"name" db:"name"`
	Description   string `json:"description" db:"description"`
	FrameworkType string `json:"frameworkType" db:"framework_type"`
	ProjectID     string `json:"projectId" db:"project_id"`
}

type MLModelProjects struct {
	MLModelDBO
	ProjectIDs []string `json:"projectIds" db:"project_ids"`
}

type MLModelVersionDBO struct {
	ID             int64      `json:"id" db:"id"`
	ModelID        string     `json:"modelId" db:"model_id"`
	ModelVersion   int        `json:"modelVersion" db:"model_version"`
	S3Version      string     `json:"s3Version" db:"s3_version"`
	Description    string     `json:"description" db:"description"`
	ModelSizeBytes int64      `json:"modelSizeBytes" db:"model_size_bytes"`
	CreatedAt      *time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt      *time.Time `json:"updatedAt" db:"updated_at"`
}

type mlModelIDsParam struct {
	ModelIDs []string `db:"model_ids"`
}

func init() {
	queryMap["SelectMLModelsByProjectsTemplate"] = `SELECT *, count(*) OVER() as total_count FROM machine_inference_model WHERE tenant_id = :tenant_id AND project_id IN (:project_ids) AND (:id = '' OR id = :id) %s`
	queryMap["SelectMLModelsVersions"] = `SELECT * FROM machine_inference_version_model WHERE model_id IN (:model_ids) ORDER BY id`
	queryMap["CreateMLModel"] = `INSERT INTO machine_inference_model (id, version, tenant_id, name, description, project_id, framework_type, created_at, updated_at) VALUES (:id, :version, :tenant_id, :name, :description, :project_id, :framework_type, :created_at, :updated_at)`
	queryMap["CreateMLModelVersion"] = `INSERT INTO machine_inference_version_model (model_id, model_version, s3_version, description, model_size_bytes, created_at, updated_at) VALUES (:model_id, :model_version, :s3_version, :description, :model_size_bytes, :created_at, :updated_at)`
	queryMap["UpdateMLModel"] = `UPDATE machine_inference_model SET version = :version, description = :description, updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`
	queryMap["UpdateMLModelVersion"] = `UPDATE machine_inference_version_model SET description = :description, updated_at = :updated_at WHERE model_id = :model_id AND model_version = :model_version`
	queryMap["DeleteMLModelVersion"] = `DELETE FROM machine_inference_version_model WHERE model_id = :model_id AND model_version = :model_version`

	orderByHelper.Setup(entityTypeMLModel, []string{"id", "version", "created_at", "updated_at", "name", "description", "framework_type"})
}

// S3 key for the model
// of the form: <version>/<tenant id>/<project id>/<model id>
func getModelS3Key(mdl model.MLModel) *string {
	return base.StringPtr(fmt.Sprintf("%s/%s/%s/%s", mlVersion, mdl.TenantID, mdl.ProjectID, mdl.ID))
}

// minio does not support object versioning, so make version part of key
func getMinioKey(s3key *string, version int) *string {
	return base.StringPtr(fmt.Sprintf("%s/%d", *s3key, version))
}

func getMLModelDBQueryParam(context context.Context, projectID string, id string) (base.InQueryParam, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return base.InQueryParam{}, err
	}
	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID, ID: id}
	param := MLModelDBO{BaseModelDBO: tenantModel}
	var projectIDs []string
	if projectID != "" {
		if !auth.IsProjectMember(projectID, authContext) {
			return base.InQueryParam{}, errcode.NewPermissionDeniedError("RBAC")
		}
		projectIDs = []string{projectID}
	} else {
		projectIDs = auth.GetProjectIDs(authContext)
		if len(projectIDs) == 0 {
			return base.InQueryParam{}, nil
		}
	}
	return base.InQueryParam{
		Param: MLModelProjects{
			MLModelDBO: param,
			ProjectIDs: projectIDs,
		},
		Key:     "SelectMLModelsByProjectsTemplate",
		InQuery: true,
	}, nil
}

func (dbAPI *dbObjectModelAPI) getMLModelsByProjectsForQuery(context context.Context, projectIDs []string, modelID string, entitiesQueryParam *model.EntitiesQueryParam) ([]model.MLModel, int, error) {
	mlModels := []model.MLModel{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return mlModels, 0, err
	}
	tenantID := authContext.TenantID
	mlModelDBOs := []MLModelDBO{}
	query, err := buildLimitQuery(entityTypeMLModel, queryMap["SelectMLModelsByProjectsTemplate"], entitiesQueryParam, orderByNameID)
	if err != nil {
		return mlModels, 0, err
	}
	err = dbAPI.QueryIn(context, &mlModelDBOs, query, tenantIDParam5{TenantID: tenantID, ProjectIDs: projectIDs, ID: modelID})
	if err != nil {
		return mlModels, 0, err
	}
	if len(mlModelDBOs) == 0 {
		return mlModels, 0, nil
	}
	totalCount := 0
	first := true
	for _, mlModelDBO := range mlModelDBOs {
		mlModel := model.MLModel{}
		if first {
			first = false
			if mlModelDBO.TotalCount != nil {
				totalCount = *mlModelDBO.TotalCount
			}
		}
		err := base.Convert(&mlModelDBO, &mlModel)
		if err != nil {
			return []model.MLModel{}, 0, err
		}
		mlModels = append(mlModels, mlModel)
	}
	err = dbAPI.populateMLModelsVersions(context, mlModels)
	return mlModels, totalCount, err
}

func (dbAPI *dbObjectModelAPI) getMLModelsW(context context.Context, projectID string, modelID string, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParam(req)
	mlModels, totalCount, err := dbAPI.getMLModels(context, projectID, modelID, entitiesQueryParam)

	if err != nil {
		return err
	}
	entityListResponsePayload := makeEntityListResponsePayload(entitiesQueryParam, &ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeMLModel})

	if len(modelID) == 0 {
		r := model.MLModelListResponsePayload{
			EntityListResponsePayload: entityListResponsePayload,
			MLModelList:               mlModels,
		}
		return json.NewEncoder(w).Encode(r)
	}
	if len(mlModels) == 0 {
		return errcode.NewRecordNotFoundError(modelID)
	}
	return json.NewEncoder(w).Encode(mlModels[0])
}

func (dbAPI *dbObjectModelAPI) getMLModels(context context.Context, projectID string, modelID string, entitiesQueryParam *model.EntitiesQueryParam) (mlModels []model.MLModel, totalCount int, err error) {
	mlModels = []model.MLModel{}
	dbQueryParam, err := getMLModelDBQueryParam(context, projectID, modelID)
	if err != nil {
		return
	}
	if dbQueryParam.Key == "" {
		return
	}
	projectIDs := dbQueryParam.Param.(MLModelProjects).ProjectIDs
	return dbAPI.getMLModelsByProjectsForQuery(context, projectIDs, modelID, entitiesQueryParam)
}

func deleteS3ModelVersions(mdl model.MLModel, mdlVersions []model.MLModelVersion) error {
	isMinio := isMinio()
	s3Client := s3.New(awsSession)
	key := getModelS3Key(mdl)
	objsToDel := []*s3.ObjectIdentifier{}
	for _, x := range mdlVersions {
		var oid *s3.ObjectIdentifier
		if isMinio {
			oid = &s3.ObjectIdentifier{
				Key: getMinioKey(key, x.ModelVersion),
			}
		} else {
			oid = &s3.ObjectIdentifier{
				Key:       key,
				VersionId: &x.S3Version,
			}
		}
		objsToDel = append(objsToDel, oid)
	}
	dinput := &s3.DeleteObjectsInput{
		Bucket: config.Cfg.MLModelS3Bucket,
		Delete: &s3.Delete{
			Objects: objsToDel,
			Quiet:   aws.Bool(false),
		},
	}
	_, err := s3Client.DeleteObjects(dinput)
	return err
}

func (dbAPI *dbObjectModelAPI) populateMLModelsVersions(context context.Context, models []model.MLModel) error {
	if len(models) == 0 {
		return nil
	}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	tenantID := authContext.TenantID
	modelIDs := []string{}
	for _, mdl := range models {
		modelIDs = append(modelIDs, mdl.ID)
	}
	param := mlModelIDsParam{
		ModelIDs: modelIDs,
	}
	modelVersionDBOs := []MLModelVersionDBO{}
	err = dbAPI.QueryIn(context, &modelVersionDBOs, queryMap["SelectMLModelsVersions"], param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in select ML models versions for tenant ID %s. Error: %s"), tenantID, err.Error())
		return errcode.TranslateDatabaseError("", err)
	}
	modelVersionsMap := map[string][]model.MLModelVersion{}
	for _, modelVersionDBO := range modelVersionDBOs {
		modelVersionsMap[modelVersionDBO.ModelID] = append(modelVersionsMap[modelVersionDBO.ModelID], model.MLModelVersion{
			ModelVersion:   modelVersionDBO.ModelVersion,
			S3Version:      modelVersionDBO.S3Version,
			Description:    modelVersionDBO.Description,
			ModelSizeBytes: modelVersionDBO.ModelSizeBytes,
			CreatedAt:      modelVersionDBO.CreatedAt,
			UpdatedAt:      modelVersionDBO.UpdatedAt,
		})
	}
	for i := range models {
		models[i].ModelVersions = modelVersionsMap[models[i].ID]
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) SelectAllMLModels(context context.Context, entitiesQueryParam *model.EntitiesQueryParam) ([]model.MLModel, error) {
	mlModels, _, err := dbAPI.getMLModels(context, "", "", entitiesQueryParam)
	return mlModels, err
}
func (dbAPI *dbObjectModelAPI) SelectAllMLModelsW(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getMLModelsW(context, "", "", w, req)
}
func (dbAPI *dbObjectModelAPI) SelectAllMLModelsForProject(context context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParam) ([]model.MLModel, error) {
	mlModels, _, err := dbAPI.getMLModels(context, projectID, "", entitiesQueryParam)
	return mlModels, err
}
func (dbAPI *dbObjectModelAPI) SelectAllMLModelsForProjectW(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getMLModelsW(context, projectID, "", w, req)
}
func (dbAPI *dbObjectModelAPI) GetMLModel(context context.Context, modelID string) (model.MLModel, error) {
	if len(modelID) == 0 {
		return model.MLModel{}, errcode.NewBadRequestError("modelID")
	}
	mlModels, _, err := dbAPI.getMLModels(context, "", modelID, nil)
	if err != nil {
		return model.MLModel{}, err
	}
	if len(mlModels) == 0 {
		return model.MLModel{}, errcode.NewRecordNotFoundError(modelID)
	}
	return mlModels[0], nil
}
func (dbAPI *dbObjectModelAPI) GetMLModelW(context context.Context, id string, w io.Writer, req *http.Request) error {
	if len(id) == 0 {
		return errcode.NewBadRequestError("mlModelID")
	}
	return dbAPI.getMLModelsW(context, "", id, w, req)
}
func (dbAPI *dbObjectModelAPI) CreateMLModel(context context.Context, i interface{} /* *model.MLModelMetadata */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponseV2{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.MLModelMetadata)
	if !ok {
		return resp, errcode.NewInternalError("CreateMLModel: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if !base.CheckID(doc.ID) {
		doc.ID = base.GetUUID()
	}
	err = model.ValidateMLModel(&doc)
	if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityMLModel,
		meta.OperationCreate,
		auth.RbacContext{
			ProjectID:  doc.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return resp, err
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	mlModelDBO := MLModelDBO{}

	err = base.Convert(&doc, &mlModelDBO)
	if err != nil {
		return resp, err
	}
	_, err = dbAPI.NamedExec(context, queryMap["CreateMLModel"], &mlModelDBO)

	if err != nil {
		return resp, err
	}

	if callback != nil {
		// go callback(context, doc)
		go func() {
			mdl, err := dbAPI.GetMLModel(context, doc.ID)
			if err == nil {
				callback(context, mdl)
			} else {
				glog.Errorf(base.PrefixRequestID(context, "CreateMLModel: error in callback path: %s"), err.Error())
			}
		}()
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addMLModelAuditLog(dbAPI, context, doc, CREATE)
	return resp, nil
}
func (dbAPI *dbObjectModelAPI) CreateMLModelW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateMLModel, &model.MLModelMetadata{}, w, r, callback)
}
func (dbAPI *dbObjectModelAPI) UpdateMLModel(context context.Context, i interface{} /* *model.MLModel */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponseV2{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.MLModelMetadata)
	if !ok {
		return resp, errcode.NewInternalError("UpdateMLModel: type error")
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
	err = model.ValidateMLModel(&doc)
	if err != nil {
		return resp, err
	}

	// TODO - FIXME - check if anything other than description changed and
	// if so, fail with bad input

	// no need to check projectID change, since our SQL update script only honors description change
	err = auth.CheckRBAC(
		authContext,
		meta.EntityMLModel,
		meta.OperationUpdate,
		auth.RbacContext{
			ProjectID:    doc.ProjectID,
			OldProjectID: doc.ProjectID,
			ProjNameFn:   GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return resp, err
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.UpdatedAt = now
	mlModelDBO := MLModelDBO{}
	err = base.Convert(&doc, &mlModelDBO)
	if err != nil {
		return resp, err
	}
	_, err = dbAPI.NamedExec(context, queryMap["UpdateMLModel"], &mlModelDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating ML model for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	if callback != nil {
		go func() {
			mdl, err := dbAPI.GetMLModel(context, doc.ID)
			if err == nil {
				callback(context, mdl)
			} else {
				glog.Errorf(base.PrefixRequestID(context, "UpdateMLModel: error in callback path: %s"), err.Error())
			}
		}()
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addMLModelAuditLog(dbAPI, context, doc, UPDATE)
	return resp, nil
}
func (dbAPI *dbObjectModelAPI) UpdateMLModelW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateMLModel, &model.MLModelMetadata{}, w, r, callback)
}
func (dbAPI *dbObjectModelAPI) DeleteMLModel(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	mdl, err := dbAPI.GetMLModel(context, id)
	if errcode.IsRecordNotFound(err) {
		return resp, nil
	} else if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityMLModel,
		meta.OperationDelete,
		auth.RbacContext{
			ProjectID:  mdl.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return resp, err
	}
	if len(mdl.ModelVersions) != 0 {
		// [Delete all S3 objects] - best effort, don't error out if delete from S3 fails.
		// This is needed since delete is not transactional, we could have S3 delete
		// succeeds while RDS delete fails. Subsequent attempt to delete S3 object
		// will fail and if we error out we will not be able to delete stale RDS entries.
		// In the worst case this may leave behind some orphaned S3 objects when
		// delete from S3 encounter some temporary error and we proceed to delete the RDS
		// entries successfully.
		// This kind of orphaned S3 objects should be rare and can be periodically
		// cleaned up by some batch job.
		err = deleteS3ModelVersions(mdl, mdl.ModelVersions)
		if err != nil {
			// note: ignore error, just log it
			glog.Errorf(base.PrefixRequestID(context, "Error delete ML model from S3, model id %s. Error: %s"), id, err.Error())
		}
	}

	result, err := DeleteEntity(context, dbAPI, "machine_inference_model", "id", id, mdl, callback)
	if err == nil {
		GetAuditlogHandler().addMLModelAuditLog(dbAPI, context, mdl.MLModelMetadata, DELETE)
	}
	return result, err
}
func (dbAPI *dbObjectModelAPI) DeleteMLModelW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteMLModel), id, w, callback)
}

func doS3Upload(context context.Context, bucket *string, key *string, version int, reader io.Reader, op string) (string, error) {
	isMinio := isMinio()
	bkey := key
	if isMinio {
		bkey = getMinioKey(key, version)
	}
	s3Uploader := s3manager.NewUploader(awsSession)
	result, err := s3Uploader.Upload(&s3manager.UploadInput{
		Bucket: bucket,
		Key:    bkey,
		Body:   reader,
	})

	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "%s: upload error, bucket %s, key %s, version: %d. Error: %s"), op, *bucket, *key, version, err.Error())
		return "", errcode.NewInternalError(fmt.Sprintf("%s: upload error", op))
	}
	if isMinio {
		return fmt.Sprintf("%d", version), nil
	}
	if result.VersionID == nil {
		return "", errcode.NewInternalError(fmt.Sprintf("%s: put object ok, but version id nil???", op))
	}
	return *result.VersionID, nil
}

/*
 validateMLModelBinary will make a RPC request to AI server to
 validate the ML Model version.If the ml model is not valid ,then it throws error.
*/
func validateMLModelBinary(ctx context.Context, mdl model.MLModel, mdlv MLModelVersionDBO) error {
	// Note: We are not validating TF 2.1.0 version models. So skipping it here.
	if mdl.FrameworkType == model.FT_TENSORFLOW_2_1_0 {
		glog.Infof("Not validating the ML model name %s version %f", mdl.Name, mdl.Version)
		return nil
	}
	reqID := base.GetRequestID(ctx)
	s3Client := s3.New(awsSession)
	var input *s3.GetObjectInput
	if isMinio() {
		s3key := getModelS3Key(mdl)
		input = &s3.GetObjectInput{
			Bucket: config.Cfg.MLModelS3Bucket,
			Key:    getMinioKey(s3key, mdlv.ModelVersion),
		}
	} else {
		input = &s3.GetObjectInput{
			Bucket:    config.Cfg.MLModelS3Bucket,
			Key:       getModelS3Key(mdl),
			VersionId: &mdlv.S3Version,
		}
	}
	req, _ := s3Client.GetObjectRequest(input)
	urlStr, err := req.Presign(time.Duration(10) * time.Minute)

	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error getting pre-signed URL for ML model version, model id %s, version %s. Error: %s"), mdl.ID, mdlv.ModelVersion, err.Error())
		return errcode.NewInternalError("Failed to get pre-signed URL for ML model version")
	}
	var mlModelType gapi.MLModelType
	if mdl.FrameworkType == model.FT_TENSORFLOW_DEFAULT {
		mlModelType = gapi.MLModelType_TENSORFLOW_1_13_1
	} else if mdl.FrameworkType == model.FT_OPENVINO_DEFAULT {
		mlModelType = gapi.MLModelType_OPENVINO_2019_R2
	}
	request := &gapi.ValidateRequest{
		Url:              urlStr,
		MlmodelType:      mlModelType,
		MlmodelSizeBytes: mdlv.ModelSizeBytes,
	}

	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewMLModelClient(conn)
		resp, err := client.Validate(ctx, request)
		if err != nil {
			glog.Errorf("Request %s: Error: %s", reqID, err.Error())
			return err
		}
		if !resp.ValidModel {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in validating ML model version, model name %s, version %d. Error: %s"), mdl.Name, mdlv.ModelVersion, resp.ErrorMsg)
			return errors.New(resp.ErrorMsg)
		}

		return nil
	}
	err = service.CallClient(ctx, service.AIService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in validating ML model version, model name %s, version %d. Error: %s"), mdl.Name, mdlv.ModelVersion, err.Error())
	}
	return err
}

func (dbAPI *dbObjectModelAPI) CreateMLModelVersionW(context context.Context, modelID string, w io.Writer, req *http.Request, callback func(context.Context, interface{}) error) error {
	resp := model.CreateDocumentResponseV2{}
	param := model.GetMLModelVersionCreateParam(req)
	modelVersion := param.ModelVersion
	mdl, err := dbAPI.GetMLModel(context, modelID)
	if err != nil {
		return err
	}
	// check we don't already have this version
	exists := false
	for _, mv := range mdl.ModelVersions {
		if mv.ModelVersion == modelVersion {
			exists = true
			break
		}
	}
	if exists {
		return errcode.NewBadRequestError("ModelVersion")
	}

	// we support both regular and multipart POST
	var reader io.Reader
	var mediaType string
	params := make(map[string]string)
	reader = req.Body
	contentType := req.Header.Get("Content-Type")
	if contentType != "" {
		mt, ps, err := mime.ParseMediaType(contentType)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error parsing content type in create ML model version, model id %s. Error: %s"), modelID, err.Error())
			return errcode.NewBadRequestError("Content-Type")
		}
		mediaType = mt
		params = ps
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(req.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				return errcode.NewBadRequestError("Content Not Found")
			}
			if err != nil {
				glog.Errorf(base.PrefixRequestID(context, "Error parsing content in create ML model version, model id %s. Error: %s"), modelID, err.Error())
				return errcode.NewBadRequestError("Content")
			}
			filename := p.FileName()
			if filename == "" {
				glog.Infof(base.PrefixRequestID(context, "create ML model version, skip part w/o filename, name: %s, part: %+v\n"), p.FormName(), *p)
				continue
			}
			glog.Infof(base.PrefixRequestID(context, "create ML model version using multipart content with file name: %s\n"), filename)
			reader = p
			break
		}
	}

	readerWrapper := base.NewReaderWrapper(reader)
	s3Key := getModelS3Key(mdl)
	s3Version, err := doS3Upload(context, config.Cfg.MLModelS3Bucket, s3Key, modelVersion, readerWrapper, "CreateMLModelVersion")
	if err != nil {
		return err
	}

	now := base.RoundedNow()
	mdlv := MLModelVersionDBO{
		ModelID:        modelID,
		ModelVersion:   modelVersion,
		S3Version:      s3Version,
		Description:    param.Description,
		ModelSizeBytes: readerWrapper.Len(),
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	tenantID := authContext.TenantID
	adminCtx := base.GetAdminContextWithTenantID(context, tenantID)
	//We pass newly created context to validate model rpc api - ENG-250555
	//Validate ML Model Binary before storing the entry in database.
	err = validateMLModelBinary(adminCtx, mdl, mdlv)
	if err != nil {
		//If the ML Model is not valid then delete the object from object store.
		mldv := model.MLModelVersion{
			ModelVersion:   modelVersion,
			S3Version:      s3Version,
			Description:    param.Description,
			ModelSizeBytes: readerWrapper.Len(),
			CreatedAt:      &now,
			UpdatedAt:      &now,
		}
		err2 := deleteS3ModelVersions(mdl, []model.MLModelVersion{mldv})
		if err2 != nil {
			glog.Errorf(base.PrefixRequestID(context, "Failed to cleanup ml model: %s version: %d.Error: %s"), modelID, modelVersion, err2)
		}
		return errcode.NewInternalError(err.Error())
	}
	_, err = dbAPI.NamedExec(context, queryMap["CreateMLModelVersion"], &mdlv)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error creating ML model version, model id %s, version %s. Error: %s"), modelID, modelVersion, err.Error())
		return errcode.TranslateDatabaseError(modelID, err)
	}
	// update ML Model version and updatedAt
	_, err = dbAPI.UpdateMLModel(context, &mdl.MLModelMetadata, callback)
	if err != nil {
		// note: ignore error, just log it
		// DB updated successfully, edge sync may be delayed
		glog.Errorf(base.PrefixRequestID(context, "CreateMLModelVersion: error in UpdateMLModel: %s"), err.Error())
	}
	resp.ID = modelID
	return json.NewEncoder(w).Encode(resp)
}
func (dbAPI *dbObjectModelAPI) UpdateMLModelVersionW(context context.Context, modelID string, modelVersion int, w io.Writer, req *http.Request, callback func(context.Context, interface{}) error) error {
	mdl, err := dbAPI.GetMLModel(context, modelID)
	if err != nil {
		return err
	}
	// check we have this version
	var mvp *model.MLModelVersion
	for _, mv := range mdl.ModelVersions {
		if mv.ModelVersion == modelVersion {
			mvp = &mv
			break
		}
	}
	if nil == mvp {
		return errcode.NewBadRequestError("ModelVersion")
	}
	// now get new description from request body
	doc := MLModelVersionDBO{}
	reader := req.Body.(io.Reader)
	err = base.Decode(&reader, &doc)
	if err != nil {
		return errcode.NewBadRequestError("description")
	}
	doc.Description = strings.TrimSpace(doc.Description)
	// update if description changed
	if mvp.Description != doc.Description {
		doc.ModelID = modelID
		doc.ModelVersion = modelVersion
		now := base.RoundedNow()
		doc.UpdatedAt = &now
		_, err = dbAPI.NamedExec(context, queryMap["UpdateMLModelVersion"], &doc)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error updating ML model version description, model id %s, version %s. Error: %s"), modelID, modelVersion, err.Error())
			return errcode.TranslateDatabaseError(modelID, err)
		}
		// update ML Model version and updatedAt, skip callback, ignore error
		// no need to notify since only model version description changed
		_, err = dbAPI.UpdateMLModel(context, &mdl.MLModelMetadata, nil)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "UpdateMLModelVersionW: error in UpdateMLModel: %s"), err.Error())
		}
	}
	return json.NewEncoder(w).Encode(model.UpdateDocumentResponseV2{ID: modelID})
}
func (dbAPI *dbObjectModelAPI) DeleteMLModelVersion(context context.Context, modelID string, modelVersion int, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponseV2{}
	mdl, err := dbAPI.GetMLModel(context, modelID)
	if err != nil {
		return resp, err
	}
	// check we have this version
	var mvp *model.MLModelVersion
	for _, mv := range mdl.ModelVersions {
		if mv.ModelVersion == modelVersion {
			mvp = &mv
			break
		}
	}
	if nil == mvp {
		return resp, errcode.NewBadRequestError("ModelVersion")
	}
	// first delete s3 object
	err = deleteS3ModelVersions(mdl, []model.MLModelVersion{*mvp})
	if err != nil {
		// note: ignore error, just log it
		// see [Delete all S3 objects] comments above
		glog.Errorf(base.PrefixRequestID(context, "Error deleting ML model version, model id %s, version %s. Error: %s"), modelID, modelVersion, err.Error())
	}

	mdlv := MLModelVersionDBO{
		ModelID:      modelID,
		ModelVersion: modelVersion,
	}
	_, err = dbAPI.NamedExec(context, queryMap["DeleteMLModelVersion"], &mdlv)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error deleting ML model version, model id %s, version %s. Error: %s"), modelID, modelVersion, err.Error())
		return resp, errcode.TranslateDatabaseError(modelID, err)
	}
	// update ML Model version and updatedAt
	_, err = dbAPI.UpdateMLModel(context, &mdl.MLModelMetadata, callback)
	if err != nil {
		// note: ignore error, just log it
		// DB updated successfully, edge sync may be delayed
		glog.Errorf(base.PrefixRequestID(context, "DeleteMLModelVersion: error in UpdateMLModel: %s"), err.Error())
	}
	resp.ID = modelID
	return resp, nil
}
func (dbAPI *dbObjectModelAPI) DeleteMLModelVersionW(context context.Context, id string, modelVersion int, w io.Writer, callback func(context.Context, interface{}) error) error {
	resp, err := dbAPI.DeleteMLModelVersion(context, id, modelVersion, callback)
	if err == nil {
		err = json.NewEncoder(w).Encode(resp)
	}
	return err
}
func (dbAPI *dbObjectModelAPI) GetMLModelVersionSignedURL(context context.Context, modelID string, modelVersion int, minutes int) (string, error) {
	mdl, err := dbAPI.GetMLModel(context, modelID)
	if err != nil {
		return "", err
	}
	// check we have this version, need to get s3 version corresponding to model version
	var mvp *model.MLModelVersion
	for _, mv := range mdl.ModelVersions {
		if mv.ModelVersion == modelVersion {
			mvp = &mv
			break
		}
	}
	if nil == mvp {
		return "", errcode.NewBadRequestError("ModelVersion")
	}
	s3Client := s3.New(awsSession)
	var input *s3.GetObjectInput
	if isMinio() {
		s3key := getModelS3Key(mdl)
		input = &s3.GetObjectInput{
			Bucket: config.Cfg.MLModelS3Bucket,
			Key:    getMinioKey(s3key, modelVersion),
		}
	} else {
		input = &s3.GetObjectInput{
			Bucket:    config.Cfg.MLModelS3Bucket,
			Key:       getModelS3Key(mdl),
			VersionId: &mvp.S3Version,
		}
	}
	req, _ := s3Client.GetObjectRequest(input)
	urlStr, err := req.Presign(time.Duration(minutes) * time.Minute)

	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error getting pre-signed URL for ML model version, model id %s, version %s. Error: %s"), modelID, modelVersion, err.Error())
		return "", errcode.NewInternalError("Failed to get pre-signed URL for ML model version")
	}
	return urlStr, nil
}

func (dbAPI *dbObjectModelAPI) GetMLModelVersionSignedURLW(context context.Context, id string, modelVersion int, w io.Writer, req *http.Request) error {
	param := model.GetMLModelVersionGetURLParam(req)
	minutes := param.ExpirationDuration
	if 0 == minutes {
		minutes = modelURLDefaultExpirationMinutes
	}
	url, err := dbAPI.GetMLModelVersionSignedURL(context, id, modelVersion, minutes)
	if err == nil {
		resp := model.MLModelVersionURLGetResponsePayload{
			URL: url,
		}
		err = json.NewEncoder(w).Encode(resp)
	}
	return err
}

func (dbAPI *dbObjectModelAPI) getMLModelsByIDs(ctx context.Context, modelIDs []string) ([]model.MLModel, error) {
	mlModels := []model.MLModel{}
	if len(modelIDs) == 0 {
		return mlModels, nil
	}

	mlModelDBOs := []MLModelDBO{}
	if err := dbAPI.queryEntitiesByTenantAndIds(ctx, &mlModelDBOs, "machine_inference_model", modelIDs); err != nil {
		return nil, err
	}

	for _, mlModelDBO := range mlModelDBOs {
		mlModel := model.MLModel{}
		err := base.Convert(&mlModelDBO, &mlModel)
		if err != nil {
			return []model.MLModel{}, err
		}
		mlModels = append(mlModels, mlModel)
	}
	err := dbAPI.populateMLModelsVersions(ctx, mlModels)
	return mlModels, err

}
