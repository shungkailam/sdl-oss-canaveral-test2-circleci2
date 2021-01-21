package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"github.com/golang/glog"
)

type AuditLog struct {
	TenantID        string    `json:"tenantId" db:"tenant_id" validate:"range=1:36"`
	UserEmail       string    `json:"userEmail" db:"user_email" validate:"range=0:200"`
	EdgeIDs         *string   `json:"edgeIds,omitempty" db:"edge_ids"`
	Hostname        string    `json:"hostname" db:"hostname" validate:"range=1:64"`
	RequestID       string    `json:"requestId" db:"request_id" validate:"range=1:36"`
	RequestMethod   string    `json:"requestMethod" db:"request_method" validate:"range=1:16"`
	RequestURL      string    `json:"requestUrl" db:"request_url" validate:"range=1:200"`
	RequestPayload  *string   `json:"requestPayload,omitempty" db:"request_payload" validate:"range=1:1024"`
	RequestHeader   *string   `json:"requestHeader,omitempty" db:"request_header" validate:"range=1:1024"`
	ResponseCode    int       `json:"responseCode" db:"response_code"`
	ResponseMessage *string   `json:"responseMessage,omitempty" db:"response_message" validate:"range=0:1024"`
	ResponseLength  int       `json:"responseLength" db:"response_length"`
	TimeMS          float32   `json:"timeMs" db:"time_ms"`
	StartedAt       time.Time `json:"startedAt" db:"started_at"`
	CreatedAt       time.Time `json:"createdAt" db:"created_at"`
}

const (
	LogResponseMaxLength = 1024
	LogPayloadMaxLength  = 1024
	LogHeaderMaxLength   = 1024
)

// Ok
// swagger:response AuditLogGetResponse
type AuditLogGetResponse struct {
	// in: body
	// required: true
	Payload *[]AuditLog
}

// swagger:parameters AuditLogGet AuditLogGetV2 AuditLogList AuditLogListV2
// in: header
type auditlogAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// Ok
// swagger:response AuditLogListResponse
type AuditLogListResponse struct {
	// in: body
	// required: true
	Payload *AuditLogListResponsePayload
}

// payload for AuditLogListResponse
type AuditLogListResponsePayload struct {
	// required: true
	PagedListResponsePayload
	// list of audit logs
	// required: true
	AuditLogList []AuditLog `json:"result"`
}

// FillInTime fill in CreatedAt and TimeMS audit log properties
func (auditLog *AuditLog) FillInTime() {
	stop := time.Since(auditLog.StartedAt)
	auditLog.CreatedAt = time.Now()
	auditLog.TimeMS = float32(float64(stop) / float64(time.Millisecond))
}

// don't include these HTTP headers in audit log
var reqWriteExcludeHeaderDump = map[string]bool{
	"Host":              true, // not in Header map anyway
	"Transfer-Encoding": true,
	"Trailer":           true,
	"Authorization":     true,
}

var reBearer = regexp.MustCompile("Bearer\\s+(.*)")

// NewAuditLogFromRequest create audit log entry from http request
// If request contains auth token, will extract (tenant id, email, edge id)
// from token into audit log entry. Will also append the decoded token
// payload to RequestHeader as AuthToken
func NewAuditLogFromRequest(r *http.Request) *AuditLog {
	auditLog := &AuditLog{StartedAt: time.Now()}

	auditLog.RequestMethod = r.Method
	auditLog.Hostname = os.Getenv("HOSTNAME")
	auditLog.RequestURL = r.URL.String()
	var w bytes.Buffer
	r.Header.WriteSubset(&w, reqWriteExcludeHeaderDump)
	s := w.String()
	auth := r.Header.Get("Authorization")
	if auth != "" {
		// append decoded auth token if present
		sm := reBearer.FindStringSubmatch(auth)
		if len(sm) > 0 {
			// match
			token := sm[1]
			decodedToken := UpdateAuditLogFromToken(auditLog, token)
			if decodedToken != "" {
				s = fmt.Sprintf("%sAuthToken: %s\r\n", s, decodedToken)
			}
		}
	}
	auditLog.RequestHeader = &s
	return auditLog
}
func UpdateAuditLogFromToken(auditLog *AuditLog, token string) string {
	result := ""
	parts := strings.Split(token, ".")
	if len(parts) == 3 {
		// expected 3 parts jwt token
		p2 := parts[1]
		decodedToken, err := jwt.DecodeSegment(p2)
		if err == nil {
			result = string(decodedToken)
			m := map[string]interface{}{}
			err = json.Unmarshal(decodedToken, &m)
			if err == nil {
				// extract tenant id, email, edge id
				tenantID, ok := m["tenantId"].(string)
				if ok {
					auditLog.TenantID = tenantID
				}
				email, ok := m["email"].(string)
				if ok {
					auditLog.UserEmail = email
				}
				edgeID, ok := m["edgeId"].(string)
				if ok {
					auditLog.EdgeIDs = &edgeID
				}
			} else {
				glog.Warningf("UpdateAuditLogFromToken: json unmarshal failed with error %s\n", err.Error())
			}
		} else {
			// base 64 decode failed
			glog.Warningf("UpdateAuditLogFromToken: base64 decode of %s failed with error %s\n", p2, err.Error())
		}
	}
	return result
}

func spStr(sp *string) string {
	if sp != nil {
		return *sp
	}
	return "<nil>"
}
func (a AuditLog) String() string {
	return fmt.Sprintf("{TenantID:%s, UserEmail:%s, EdgeIDs:%s, Hostname:%s, RequestID:%s, RequestMethod:%s, RequestURL:%s, RequestPayload:%s, RequestHeader:%s, ResponseCode:%d, ResponseMessage:%s, ResponseLength: %d, TimeMS: %f, StartedAt: %s, CreatedAt: %s}",
		a.TenantID, a.UserEmail, spStr(a.EdgeIDs), a.Hostname, a.RequestID, a.RequestMethod,
		a.RequestURL, spStr(a.RequestPayload), spStr(a.RequestHeader), a.ResponseCode,
		spStr(a.ResponseMessage), a.ResponseLength, a.TimeMS,
		a.StartedAt.Format(time.RFC3339), a.CreatedAt.Format(time.RFC3339))
}
