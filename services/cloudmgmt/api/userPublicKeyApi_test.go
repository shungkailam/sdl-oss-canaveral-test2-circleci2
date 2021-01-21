package api_test

import (
	"bytes"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUserPublicKey(t *testing.T) {
	t.Parallel()

	t.Log("running TestUserPublicKey test")
	// Setup
	dbAPI := newObjectModelAPI(t)

	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	user := createUser(t, dbAPI, tenantID)
	userId := user.ID

	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{userId}, nil)
	projectID := project.ID
	ctx1, _, _ := makeContext(tenantID, []string{projectID})

	// Teardown
	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteUser(ctx1, userId, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/DeleteUserPublicKey", func(t *testing.T) {
		t.Log("running Create/Get/DeleteUserPublicKey test")

		// set context to correspond to user
		// since only user can get/update/delete own public keys
		authContext, err := base.GetAuthContext(ctx1)
		require.NoError(t, err)
		authContext.ID = userId
		claims := authContext.Claims
		claims["id"] = userId

		doc := model.UserPublicKey{ID: userId, TenantID: tenantID, PublicKey: "my-pub-key"}
		r, err := dbAPI.UpdateUserPublicKey(ctx1, &doc, nil)
		require.NoError(t, err)
		t.Log("create user public key got response: ", r)

		key, err := dbAPI.GetUserPublicKey(ctx1)
		require.NoError(t, err)
		t.Log("Got user public key:", key)
		if key.PublicKey != doc.PublicKey {
			t.Fatal("expect public key to match")
		}

		// now do update:
		key.PublicKey = "my-pub-key-updated"
		r, err = dbAPI.UpdateUserPublicKey(ctx1, &key, nil)
		require.NoError(t, err)
		t.Log("update user public key got response: ", r)

		key2, err := dbAPI.GetUserPublicKey(ctx1)
		require.NoError(t, err)
		t.Log("Got user public key:", key2)
		if key.PublicKey != key2.PublicKey {
			t.Fatal("expect public key to match")
		}

		keys, err := dbAPI.SelectAllUserPublicKeys(ctx1)
		require.NoError(t, err)
		if len(keys) != 1 {
			t.Fatal("expect to get one key")
		}
		if keys[0].PublicKey != key.PublicKey {
			t.Fatal("expect public key to match")
		}

		r2, err := dbAPI.DeleteUserPublicKey(ctx1, userId, nil)
		require.NoError(t, err)
		t.Log("Delete user public key response:", r2)

		// now get must fail with not found
		key, err = dbAPI.GetUserPublicKey(ctx1)
		require.Error(t, err, "expect get user public key to fail after delete")

	})
}

func TestUserPublicKeyW(t *testing.T) {
	t.Parallel()

	t.Log("running TestUserPublicKey test")
	// Setup
	dbAPI := newObjectModelAPI(t)

	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	user := createUser(t, dbAPI, tenantID)
	userId := user.ID

	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{userId}, nil)
	projectID := project.ID
	ctx1, _, _ := makeContext(tenantID, []string{projectID})

	// Teardown
	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteUser(ctx1, userId, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/DeleteUserPublicKey", func(t *testing.T) {
		t.Log("running Create/Get/DeleteUserPublicKey test")

		// set context to correspond to user
		// since only user can get/update/delete own public keys
		authContext, err := base.GetAuthContext(ctx1)
		require.NoError(t, err)
		authContext.ID = userId
		claims := authContext.Claims
		claims["id"] = userId

		doc := model.UserPublicKey{ID: userId, TenantID: tenantID, PublicKey: "my-pub-key"}
		reader, err := objToReader(doc)
		require.NoError(t, err)

		var w bytes.Buffer
		err = dbAPI.UpdateUserPublicKeyW(ctx1, &w, reader, nil)
		require.NoError(t, err)
		resp := model.UpdateDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&resp)
		require.NoError(t, err)

		t.Log("create user public key got response: ", resp)

		err = dbAPI.GetUserPublicKeyW(ctx1, "", &w, nil)
		require.NoError(t, err)
		key := model.UserPublicKey{}
		err = json.NewDecoder(&w).Decode(&key)
		require.NoError(t, err)
		t.Log("Got user public key:", key)

		// now do update:
		key.PublicKey = "my-pub-key-updated"
		reader, err = objToReader(key)
		err = dbAPI.UpdateUserPublicKeyW(ctx1, &w, reader, nil)
		require.NoError(t, err)
		resp = model.UpdateDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&resp)
		require.NoError(t, err)
		t.Log("update user public key got response: ", resp)
		err = dbAPI.GetUserPublicKeyW(ctx1, "", &w, nil)
		require.NoError(t, err)
		key = model.UserPublicKey{}
		err = json.NewDecoder(&w).Decode(&key)
		require.NoError(t, err)
		t.Log("Got user public key:", key)

		err = dbAPI.SelectAllUserPublicKeysW(ctx1, &w, nil)
		require.NoError(t, err)
		keys := []model.UserPublicKey{}
		err = json.NewDecoder(&w).Decode(&keys)
		require.NoError(t, err)
		if len(keys) != 1 {
			t.Fatal("expect to get one key")
		}
		if keys[0].PublicKey != key.PublicKey {
			t.Fatal("expect public key to match")
		}

		err = dbAPI.DeleteUserPublicKeyW(ctx1, key.ID, &w, nil)
		require.NoError(t, err)

		dresp := model.DeleteDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&dresp)
		require.NoError(t, err)

		t.Log("Delete user public key response:", dresp)

		err = dbAPI.GetUserPublicKeyW(ctx1, "", &w, nil)

		require.Error(t, err, "expect get user public key to fail after delete")
		// Note: no output is written to w when err happens
	})
}
