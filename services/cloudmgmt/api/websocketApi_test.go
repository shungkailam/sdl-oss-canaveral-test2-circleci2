package api_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func wsEncode(t *testing.T, doc interface{}) string {
	jsonText, err := json.Marshal(&doc)
	require.NoError(t, err)
	return string(jsonText)
}

func TestObjectRequest(t *testing.T) {
	t.Parallel()
	t.Log("running TestObjectRequest test")
	// Setup
	dbAPI := newObjectModelAPI(t)

	// // Teardown x
	defer func() {

		dbAPI.Close()
	}()

	t.Run("TestObjectRequest", func(t *testing.T) {
		t.Log("running TestObjectRequest test")

		var doc interface{}

		user := model.User{
			Name:     "user-name",
			Email:    "user-email",
			Password: "user-password",
		}
		objRequest := api.ObjectRequest{
			RequestID: "request-id",
			TenantID:  "tenant-id",
			Doc:       user,
		}

		b, err := json.Marshal(objRequest)
		require.NoError(t, err)
		s := string(b)

		t.Logf("marshal of objRequest gives: %s", s)

		objReq2 := api.ObjectRequest{}
		err = json.Unmarshal([]byte(s), &objReq2)
		require.NoError(t, err)
		t.Logf("unmarshal of objRequest string gives: %+v", objReq2)

		doc = s
		t.Logf("type and value of s: %+v, %+v", reflect.TypeOf(doc), reflect.ValueOf(doc))

		doc = objReq2
		t.Logf("type and value of obj req: %+v, %+v", reflect.TypeOf(doc), reflect.ValueOf(doc))

		// given s

		t.Logf("ws encode of struct: %s", wsEncode(t, objRequest))

		doc = s
		t.Logf("ws encode of json string of struct: %s", wsEncode(t, s))

		doc = struct{}{}
		err = json.Unmarshal([]byte(s), &doc)
		require.NoError(t, err)
		t.Logf("ws encode of json unmarshal of struct: %s", wsEncode(t, doc))

		objReq3 := api.ObjectRequest{}
		err = json.Unmarshal([]byte(wsEncode(t, doc)), &objReq3)
		require.NoError(t, err)
		if !reflect.DeepEqual(objReq2, objReq3) {
			t.Fatal("expect struct to be equal")
		}

		// Doc = pointer case
		objRequest4 := api.ObjectRequest{
			RequestID: "request-id",
			TenantID:  "tenant-id",
			Doc:       &user,
		}
		b, err = json.Marshal(objRequest4)
		require.NoError(t, err)
		s = string(b)
		doc = struct{}{}
		err = json.Unmarshal([]byte(s), &doc)
		require.NoError(t, err)
		t.Logf("ws encode of json unmarshal of struct: %s", wsEncode(t, doc))

		objReq4 := api.ObjectRequest{}
		err = json.Unmarshal([]byte(wsEncode(t, doc)), &objReq4)
		require.NoError(t, err)

		if !reflect.DeepEqual(objRequest4, objReq4) {
			t.Logf("objRequest4=%+v", objRequest4)
			t.Logf("objReq4=%+v", objReq4)
			// t.Fatal("expect struct to be equal") // objReq4.Doc is a map, not pointer
		}

		b, err = json.Marshal(objReq4)
		require.NoError(t, err)
		s4 := string(b)
		if s4 != s {
			t.Logf("s=%s", s)
			t.Logf("s4=%s", s4)
			// t.Fatal("expect strings to be equal") // sort differ
		}

		u := model.User{}
		err = base.Convert(objReq4.Doc, &u)
		require.NoError(t, err)
		objReq5 := objReq4
		objReq5.Doc = &u
		if !reflect.DeepEqual(objRequest4, objReq5) {
			t.Logf("objRequest4=%+v", objRequest4)
			t.Logf("objReq5=%+v", objReq5)
			t.Fatal("expect struct to be equal")
		}
		b, err = json.Marshal(objReq5)
		require.NoError(t, err)
		s5 := string(b)
		if s5 != s {
			t.Logf("s=%s", s)
			t.Logf("s5=%s", s5)
			t.Fatal("expect strings to be equal")
		}

	})
}

func assertEdgeConnected(t *testing.T, tenantID string, edgeID string, connected bool) {
	if connected != api.IsEdgeConnected(tenantID, edgeID) {
		t.Fatalf("expect edge connected to be %t, tenantID=%s, edgeID=%s", connected, tenantID, edgeID)
	}
}

func TestIsEdgeConnected(t *testing.T) {
	t.Parallel()
	assertEdgeConnected(t, "tenant-id-waldot", "foo", true)
	assertEdgeConnected(t, "tenant-id-numart-stores", "foo", true)
	assertEdgeConnected(t, "tenant-id-smart-retail", "foo", true)
	assertEdgeConnected(t, "tid-demo-foo", "eid-demo-foo", true)
	assertEdgeConnected(t, "tid-demo-foo", "x-eid-demo-foo", false)
}
