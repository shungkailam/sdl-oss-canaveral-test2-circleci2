package model_test

import (
	"cloudservices/common/model"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAuditLog will test AuditLog struct
func TestAuditLog(t *testing.T) {
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InRlc3Qtcm9vdC0yMDE5LTAxLTI5LTEyLTAzLTI2QG50bnhzaGVybG9jay5jb20iLCJleHAiOjE1NDkwNjYxMTIsImlkIjoiNTRhMmRkY2QtMDQ3MS00OWYwLTlhOTctNGIxNDg4NWIyNzMyIiwibmFtZSI6InRlc3Qtcm9vdC0yMDE5LTAxLTI5LTEyLTAzLTI2IiwibmJmIjoxNDQ0NDc4NDAwLCJyb2xlcyI6W10sInNjb3BlcyI6W10sInNwZWNpYWxSb2xlIjoiYWRtaW4iLCJ0ZW5hbnRJZCI6ImVhYTIyZmI2LTI0MDAtMTFlOS04MTlhLTUwNmI4ZGVlMjYxMSJ9.CzAJ2RUvgWt6bCVRw1LwCuexnmGSuxbmLQE47TadSGs"
	auditLog := &model.AuditLog{}
	decodedToken := model.UpdateAuditLogFromToken(auditLog, token)
	if len(decodedToken) == 0 {
		t.Fatal("expect decoded token to be non-empty")
	}

	url := "http://example.com/foo"
	req := httptest.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	auditLog2 := model.NewAuditLogFromRequest(req)
	if auditLog2.RequestURL != url {
		t.Fatal("expect url to match")
	}
	if false == strings.Contains(*auditLog2.RequestHeader, "AuthToken") {
		t.Fatal("expect header to contain auth token")
	}
	t.Logf("Got request header: %s", *auditLog2.RequestHeader)

	t.Logf("audit log: %s", auditLog)
	t.Logf("audit log 2: %s", auditLog2)
}
