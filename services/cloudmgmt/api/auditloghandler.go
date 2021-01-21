package api

import (
	"cloudservices/common/base"
	"sync"

	//"cloudservices/common/errcode"
	"cloudservices/common/model"
	//"github.com/dgrijalva/jwt-go"
	//"google.golang.org/grpc/metadata"
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang/glog"
)

//todo: add more cloudmgmt objects
const (
	PROJECT              = "Project"
	SERVICE_DOMAIN       = "Service Domain"
	APPLICATION          = "Application"
	ML_MODEL             = "ML Model"
	FUNCTION             = "Function"
	DATA_SOURCE          = "Data Source"
	NODE                 = "Node"
	USER                 = "User"
	CATEGORY             = "Category"
	CONTAINER_REGISTRY   = "Container Registry"
	DATA_PIPELINE        = "Data Pipeline"
	CLOUD_PROFILE        = "Cloud Profile"
	API_KEY              = "API Key"
	RUNTIME_ENV          = "Runtime Environment"
	SERVICE_INSTANCE     = "Service Instance"
	SERVICE_BINDING      = "Service Binding"
	KUBERNETES_CLUSTER   = "Kubernetes Cluster"
	LOG_COLLECTOR        = "Log Collector"
	DATA_DRIVER_CLASS    = "Data Driver Class"
	DATA_DRIVER_INSTANCE = "Data Driver Instance"
	DATA_DRIVER_CONFIG   = "Data Driver Config"
	DATA_DRIVER_STREAM   = "Data Driver Stream"

	CREATE = "CREATE"
	UPDATE = "UPDATE"
	DELETE = "DELETE"

	infraRole = "admin"
	// todo: confirm this none value again from Deepak, Shyan
	projectRole  = "none"
	operatorRole = "operator"
)

var (
	handler *AuditlogHandler
	mutex   = &sync.Mutex{}
)

func (handler *AuditlogHandler) readFromQueueGoroutine() {
	for packet := range handler.auditlogQueue {
		response, err := packet.dbAPI.InsertAuditLogV2(packet.context, packet.model)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(packet.context, "Error in grpc call. AuditLog : Insert : Scope : %s : OperationType : %s : Resource : %s : Error: %s"), packet.model.AuditLog.Scope, packet.model.AuditLog.OperationType, packet.model.AuditLog.ResourceType, err.Error())
		} else {
			glog.V(3).Infof("AuditLog : Response : %s", response)
		}
	}
}

func GetAuditlogHandler() *AuditlogHandler {
	if handler == nil {
		mutex.Lock()
		handler = NewAuditLogHandler()
		mutex.Unlock()
		go GetAuditlogHandler().readFromQueueGoroutine()
	}
	return handler
}

func NewAuditLogHandler() *AuditlogHandler {
	return &AuditlogHandler{auditlogQueue: make(chan AuditlogChannelPacket, 100)}
}

type AuditlogHandler struct {
	auditlogQueue chan AuditlogChannelPacket
}

type AuditlogChannelPacket struct {
	dbAPI   *dbObjectModelAPI
	model   model.AuditLogV2InsertRequest
	context context.Context
}

// Wrapper for call to InsertAuditLogV2. Can later write implementation for Redis Pub/Sub inside this method.
func (handler *AuditlogHandler) createAuditLog(objectModelAPI *dbObjectModelAPI, ctx context.Context, resourceType string, operationType string, projectName string, projectID string, resourceName string, resourceID string, serviceDomainNames []string, serviceDomainIDs []string, payload string, scope string) {
	glog.V(3).Infoln("auditloghandler : ctx: ", ctx)
	// this won't give error if the claims object contains at least one key-value pair
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		glog.Error("Error in extracting AuthContext from context : ", ctx, err.Error())
		//return "", err
		return
	}
	claims := authContext.Claims
	err = GetAuditlogHandler().checkClaims(authContext)
	if err != nil {
		glog.Warning("Error in checking auth claims", err.Error())
		return
	}
	auditLogV2 := model.AuditLogV2{
		Timestamp:     time.Now(),
		TenantID:      authContext.TenantID,
		OperationType: operationType,
		ModifierName:  claims["name"].(string),
		ModifierRole:  claims["specialRole"].(string),
		ResourceType:  resourceType,
		ResourceName:  resourceName,
		ResourceID:    resourceID,
		Scope:         scope,
		Payload:       payload, // payload will be empty for now
	}

	if _, ok := claims["id"].(string); ok {
		auditLogV2.ModifierID = claims["id"].(string)
	}

	switch claims["specialRole"].(string) {
	case infraRole:
		auditLogV2.ModifierRole = "Infrastructure Admin"
	case projectRole:
		auditLogV2.ModifierRole = "User"
	}

	// creation of SD cannot have these fields as these fields describe the SD that was *modified*
	if serviceDomainIDs != nil {
		auditLogV2.ServiceDomainName = strings.Join(serviceDomainNames, ",")
		auditLogV2.ServiceDomainID = strings.Join(serviceDomainIDs, ",")
	}

	// any operation related SD or creation of project cannot have these fields as these fields describe the project that was *modified*
	if projectID != "" {
		auditLogV2.ProjectName = projectName
		auditLogV2.ProjectID = projectID
	}

	auditLogV2.Operation = GetAuditlogHandler().getOperationText(auditLogV2.ResourceType, auditLogV2.OperationType, auditLogV2.ResourceName)

	if resourceType == API_KEY {
		auditLogV2.ResourceType = USER
	}
	glog.V(10).Info("oldContext: ", ctx)
	newContext := GetAuditlogHandler().getNewContext(ctx)
	glog.V(10).Info("newContext: ", newContext)
	glog.V(3).Infof("AuditLog : Insert : Scope : %s : OperationType : %s : Resource : %s", scope, operationType, resourceType)

	go func() {
		GetAuditlogHandler().auditlogQueue <- AuditlogChannelPacket{
			dbAPI:   objectModelAPI,
			model:   model.AuditLogV2InsertRequest{AuditLog: auditLogV2},
			context: newContext,
		}
	}()
}

func (handler *AuditlogHandler) InsertInfraScopeAuditLog(objectModelAPI *dbObjectModelAPI, ctx context.Context, resourceType string, operationType string, projectName string, projectID string, resourceName string, resourceID string, serviceDomainNames []string, serviceDomainIDs []string, payload string) {
	GetAuditlogHandler().createAuditLog(objectModelAPI, ctx, resourceType, operationType, projectName, projectID, resourceName, resourceID, serviceDomainNames, serviceDomainIDs, payload, "INFRA")
}

func (handler *AuditlogHandler) InsertProjectScopeAuditLog(objectModelAPI *dbObjectModelAPI, ctx context.Context, resourceType string, operationType string, projectName string, projectID string, resourceName string, resourceID string, serviceDomainNames []string, serviceDomainIDs []string, payload string) {
	GetAuditlogHandler().createAuditLog(objectModelAPI, ctx, resourceType, operationType, projectName, projectID, resourceName, resourceID, serviceDomainNames, serviceDomainIDs, payload, "PROJECT")
}

// Util method to get text for the operation field. Eg. Created new data source "Sony Camera"
func (handler *AuditlogHandler) getOperationText(resourceType string, operationType string, resourceName string) string {
	var resType, opType, resName string

	switch operationType {
	case CREATE:
		opType = "Added new "
	case UPDATE:
		opType = "Updated "
	case DELETE:
		opType = "Removed "
	}

	if resourceType != API_KEY {
		resType = strings.ToLower(resourceType) + " "
	} else {
		resType = resourceType + " "
	}

	resName = "\"" + resourceName + "\". "

	return opType + resType + resName
}

// method to check if claims map contains the necessary key-value pairs.
// doesn't seem efficient or scalable but, at the moment,
// don't know any other way to perform this check.
func (handler *AuditlogHandler) checkClaims(authContext *base.AuthContext) error {
	if _, ok := authContext.Claims["specialRole"].(string); ok {
		if _, ok := authContext.Claims["name"].(string); ok {
			return nil
		}
	}
	return errors.New("AuthContext.Claims map is incomplete ")
}

// getting a new context from context.Background()
// because goroutine receives cancelled context error when calling function returns
// oldContext has some additional request information which is not required at the moment for the call.
// newContext only has the AuthContext struct. Additional fields like requestID can be further added to this context in this method.
func (handler *AuditlogHandler) getNewContext(oldContext context.Context) context.Context {
	newContext, ok := oldContext.Value(base.AuthContextKey).(*base.AuthContext)
	if !ok {
		glog.Error("Cannot extract authContext from context. ")
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, newContext)
	return ctx
}
