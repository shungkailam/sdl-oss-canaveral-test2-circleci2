package base_test

import (
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"testing"
)

func TestDBCache(t *testing.T) {
	ctx := context.Background()
	query := `DELETE FROM "category_model" WHERE id = '1223'1`
	keys := base.GetDBCacheObjectKeys(ctx, query, model.Category{BaseModel: model.BaseModel{TenantID: "test-tenant"}}, true)
	expectedKeys := map[string]bool{
		"test-tenant:category_model": true,
		":category_model":            true,
	}
	if len(keys) != len(expectedKeys) {
		t.Fatalf("Expected %d, found %d", len(expectedKeys), len(keys))
	}
	for _, key := range keys {
		delete(expectedKeys, key)
	}
	if len(expectedKeys) != 0 {
		t.Fatalf("Unmatched keys found %+v", expectedKeys)
	}

	keys = base.GetDBCacheObjectKeys(ctx, query, model.Category{BaseModel: model.BaseModel{TenantID: "test-tenant"}}, false)
	expectedKeys = map[string]bool{
		"test-tenant:category_model": true,
	}
	if len(keys) != len(expectedKeys) {
		t.Fatalf("Expected %d, found %d", len(expectedKeys), len(keys))
	}
	for _, key := range keys {
		delete(expectedKeys, key)
	}
	if len(expectedKeys) != 0 {
		t.Fatalf("Unmatched keys found %+v", expectedKeys)
	}

	query = `SELECT data_stream_origin_model.*, category_value_model.category_id "category_info.id", category_value_model.value "category_info.value"
		FROM data_stream_origin_model JOIN category_value_model ON data_stream_origin_model.category_value_id = category_value_model.id WHERE data_stream_origin_model.data_stream_id IN (:data_stream_ids)`

	keys = base.GetDBCacheObjectKeys(ctx, query, model.Category{BaseModel: model.BaseModel{TenantID: "test-tenant"}}, true)
	expectedKeys = map[string]bool{
		"test-tenant:data_stream_origin_model": true,
		":data_stream_origin_model":            true,
		"test-tenant:category_value_model":     true,
		":category_value_model":                true,
	}
	if len(keys) != len(expectedKeys) {
		t.Fatalf("Expected %d, found %d", len(expectedKeys), len(keys))
	}
	for _, key := range keys {
		delete(expectedKeys, key)
	}

	if len(expectedKeys) != 0 {
		t.Fatalf("Unmatched keys found %+v", expectedKeys)
	}

	keys = base.GetDBCacheObjectKeys(ctx, query, model.Category{BaseModel: model.BaseModel{TenantID: "test-tenant"}}, false)
	expectedKeys = map[string]bool{
		"test-tenant:data_stream_origin_model": true,
		"test-tenant:category_value_model":     true,
	}
	if len(keys) != len(expectedKeys) {
		t.Fatalf("Expected %d, found %d", len(expectedKeys), len(keys))
	}
	for _, key := range keys {
		delete(expectedKeys, key)
	}

	if len(expectedKeys) != 0 {
		t.Fatalf("Unmatched keys found %+v", expectedKeys)
	}

	isWrite := base.IsWriteQuery(query)
	if isWrite {
		t.Fatalf("Query %s is not write query", query)
	}

	query = `INSERT INTO data_source_field_model (name, data_source_id, mqtt_topic, field_type) VALUES (:name, :data_source_id, :mqtt_topic, :field_type) RETURNING id`
	isWrite = base.IsWriteQuery(query)
	if !isWrite {
		t.Fatalf("Query %s is write query", query)
	}
	query = `UPDATE edge_model SET short_id = :short_id WHERE tenant_id = :tenant_id AND id = :id`
	isWrite = base.IsWriteQuery(query)
	if !isWrite {
		t.Fatalf("Query %s is write query", query)
	}
	query = `  DELETE from edge_model WHERE tenant_id = :tenant_id AND id = :id`
	isWrite = base.IsWriteQuery(query)
	if !isWrite {
		t.Fatalf("Query %s is write query", query)
	}
}
