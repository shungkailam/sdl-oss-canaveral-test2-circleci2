package base

import (
	"cloudservices/common/errcode"
	"strings"
	//"cloudservices/common/model"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
)

// ContextKey is the type for keys in go context
type ContextKey string

const (
	// MachineTenantID is used to perform special actions initiated in the system
	// It can be used to differentiate user requests from such internal requests
	MachineTenantID       = "tid-machine-tenant"
	OperatorTenantID      = "tid-sherlock-operator" // Same value is used in nodejs script
	RequestIDKey          = ContextKey("request-id")
	AuthContextKey        = ContextKey("auth-context")
	HTTPRequestContextKey = ContextKey("http-request-context")
	AuditLogTableNameKey  = ContextKey("audit-log-table-name")
)

// AuthContext carries the auth information and is passed accross micro-services.
type AuthContext struct {
	TenantID string
	// optional ID of context entity - e.g., in PUT request this would be ID of object to update
	ID     string
	Claims jwt.MapClaims
}

type HTTPRequestContext struct {
	URI    string
	Method string
	Params string
}

func (authContext *AuthContext) GetUserID() string {
	userID, ok := authContext.Claims["id"].(string)
	if ok {
		return userID
	}
	return ""
}

func GetDateString() string {
	now := time.Now()
	return fmt.Sprintf("%d%02d%02d", now.Year(), now.Month(), now.Day())
}

// GetDateStart get today start string for DB query use
func GetDateStart() string {
	now := time.Now()
	return fmt.Sprintf("%d-%02d-%02d", now.Year(), now.Month(), now.Day())
}

// GetDateEnd get today end string for DB query use
func GetDateEnd() string {
	now := time.Now()
	return fmt.Sprintf("%d-%02d-%02d 23:59:59", now.Year(), now.Month(), now.Day())
}

// GetRequestID retrieves request ID from the go context.
func GetRequestID(ctx context.Context) string {
	reqID, ok := ctx.Value(RequestIDKey).(string)
	if !ok {
		return GetUUID()
	}
	return reqID
}

// GetAuditLogTableName retrieves audit log table name from the go context.
func GetAuditLogTableName(ctx context.Context) string {
	tableName, ok := ctx.Value(AuditLogTableNameKey).(string)
	if !ok {
		return ""
	}
	return tableName
}

// GetAuthContext retrieves the AuthContext from the go context.
func GetAuthContext(ctx context.Context) (*AuthContext, error) {
	authContext, ok := ctx.Value(AuthContextKey).(*AuthContext)
	if !ok {
		if glog.V(4) {
			reqID := GetRequestID(ctx)
			// This is not supposed to happen
			glog.V(4).Infof("Request %s: authContext is not set", reqID)
		}
		return nil, errcode.NewBadRequestError("authContext")
	}
	return authContext, nil
}

// GetHTTPRequestContext gets the HTTP request context.
func GetHTTPRequestContext(context context.Context) HTTPRequestContext {
	httpContext, ok := context.Value(HTTPRequestContextKey).(HTTPRequestContext)
	if !ok {
		return HTTPRequestContext{}
	}
	return httpContext
}

func GetAdminContext(reqID string, tenantID string) context.Context {
	authContext := &AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), AuthContextKey, authContext)
	ctx = context.WithValue(ctx, RequestIDKey, reqID)
	return ctx
}

func GetAdminContextWithTenantID(ctx context.Context, tenantID string) context.Context {
	authContext := &AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	return context.WithValue(ctx, AuthContextKey, authContext)
}

// GetOperatorContext returns the go context with the operator role set in the authcontext
func GetOperatorContext(ctx context.Context) context.Context {
	authContext := &AuthContext{
		TenantID: GetUUID(),
		Claims: jwt.MapClaims{
			"specialRole": "operator",
		},
	}
	return context.WithValue(ctx, AuthContextKey, authContext)
}

// IsEdgeRequest returns true along with edge ID if the authContext is from an edge.
// Otherwise, it returns the false with the email
func IsEdgeRequest(authContext *AuthContext) (bool, string) {
	// Only an edge must be able to call this API
	if role, ok := authContext.Claims["specialRole"]; !ok || role != "edge" {
		email, _ := authContext.Claims["email"].(string)
		return false, email
	}
	edgeID, _ := authContext.Claims["edgeId"].(string)
	return true, edgeID
}

// GetHTTPQueryParams extracts the query parameters matching the json fields in the receiver struct pointer.
// It supports only in, int64 and string types
func GetHTTPQueryParams(req *http.Request, receiver interface{}) error {
	if req == nil {
		return errcode.NewBadRequestError("request")
	}
	if receiver == nil || reflect.TypeOf(receiver).Kind() != reflect.Ptr {
		return errcode.NewInternalError("Invalid receiver - pointer expected")
	}
	return extractQueryParams(req.Context(), req.URL.Query(), reflect.ValueOf(receiver))

}

func extractQueryParams(ctx context.Context, urlQueryValues url.Values, value reflect.Value) error {
	value = reflect.Indirect(value)
	valType := value.Type()
	for i := 0; i < valType.NumField(); i++ {
		typeField := valType.Field(i)
		kind := typeField.Type.Kind()
		valueField := value.FieldByName(typeField.Name)
		if kind == reflect.Struct {
			// Handle only embedded structs which do not have json tags
			err := extractQueryParams(ctx, urlQueryValues, valueField)
			if err != nil {
				return err
			}
			continue
		}
		jsonName, ok := typeField.Tag.Lookup("json")
		if !ok {
			continue
		}
		queryParams := urlQueryValues[jsonName]
		if len(queryParams) == 0 {
			continue
		}
		if kind == reflect.Int || kind == reflect.Int64 {
			numVal, err := strconv.Atoi(queryParams[0])
			if err != nil {
				glog.Errorf(PrefixRequestID(ctx, "Error in data conversion. Error: %s"), err.Error())
				return errcode.NewInternalError("Invalid receiver - mismatched type")
			}
			valueField.SetInt(int64(numVal))
		} else if kind == reflect.String {
			valueField.SetString(queryParams[0])
		} else if kind == reflect.Slice {
			sliceElemType := valueField.Type().Elem()
			sliceElemTypeKind := sliceElemType.Kind()
			sliceValue := reflect.MakeSlice(reflect.SliceOf(sliceElemType), 0, len(queryParams))
			if sliceElemTypeKind == reflect.Int || sliceElemTypeKind == reflect.Int64 {
				for _, queryParam := range queryParams {
					numVal, err := strconv.Atoi(queryParam)
					if err != nil {
						glog.Errorf(PrefixRequestID(ctx, "Error in data conversion. Error: %s"), err.Error())
						return errcode.NewInternalError("Invalid receiver - mismatched type")
					}
					sliceValue.Set(reflect.Append(sliceValue, reflect.ValueOf(numVal)))
				}
			} else if sliceElemTypeKind == reflect.String {
				for _, queryParam := range queryParams {
					sliceValue = reflect.Append(sliceValue, reflect.ValueOf(queryParam))
				}
			} else {
				glog.Errorf(PrefixRequestID(ctx, "Unsupported slice type %s"), sliceElemTypeKind)
				return errcode.NewInternalError("Invalid receiver - unsupported type")
			}
			valueField.Set(sliceValue)
		} else if kind == reflect.Bool {
			valueField.SetBool(strings.ToLower(queryParams[0]) == "true")
		} else {
			glog.Errorf(PrefixRequestID(ctx, "Unsupported type %s"), kind)
			return errcode.NewInternalError("Invalid receiver - unsupported type")
		}
	}
	return nil
}
