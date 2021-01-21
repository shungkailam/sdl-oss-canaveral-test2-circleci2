package model_test

import (
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// TestCategory will test Category struct
func TestCategory(t *testing.T) {
	var tenantID = "tenant-id-waldot"
	var values = []string{"v1", "v2", "v3"}
	now := timeNow(t)
	categories := []model.Category{
		{
			BaseModel: model.BaseModel{
				ID:        "cat-id",
				TenantID:  tenantID,
				Version:   5,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:    "test-cat",
			Purpose: "test category",
			Values:  values,
		},
	}
	categoryStrings := []string{
		`{"id":"cat-id","version":5,"tenantId":"tenant-id-waldot","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"test-cat","purpose":"test category","values":["v1","v2","v3"]}`,
	}

	var version float64 = 5
	categoryMaps := []map[string]interface{}{
		{
			"id":        "cat-id",
			"version":   version,
			"tenantId":  tenantID,
			"name":      "test-cat",
			"purpose":   "test category",
			"values":    []string{"v1", "v2", "v3"},
			"createdAt": NOW,
			"updatedAt": NOW,
		},
	}

	for i, category := range categories {
		categoryData, err := json.Marshal(category)
		require.NoError(t, err, "failed to marshal category")

		if categoryStrings[i] != string(categoryData) {
			t.Fatalf("category json string mismatch: %s", string(categoryData))
		}
		// alternative form: m := make(map[string]interface{})
		m := map[string]interface{}{}
		err = json.Unmarshal(categoryData, &m)
		require.NoError(t, err, "failed to unmarshal category to map")

		// reflect.DeepEqual fails on equivalent slices here,
		// so use weaker marshal equal
		if !model.MarshalEqual(&m, &categoryMaps[i]) {
			t.Fatalf("category map marshal mismatch: %+v", m)
		}
	}

	// test category match
	ci1 := model.CategoryInfo{
		ID:    "c-id-1",
		Value: "c1-v1",
	}
	ci2 := model.CategoryInfo{
		ID:    "c-id-1",
		Value: "c1-v2",
	}
	ci3 := model.CategoryInfo{
		ID:    "c-id-2",
		Value: "c2-v1",
	}
	ci4 := model.CategoryInfo{
		ID:    "c-id-3",
		Value: "c3-v1",
	}
	ci5 := model.CategoryInfo{
		ID:    "c-id-1",
		Value: "c1-v3",
	}
	ci6 := model.CategoryInfo{
		ID:    "c-id-2",
		Value: "c3-v2",
	}
	labels := []model.CategoryInfo{ci1, ci2, ci3, ci4}
	selectors1 := []model.CategoryInfo{ci1, ci2, ci3, ci4}
	selectors2 := []model.CategoryInfo{ci1, ci3, ci4}
	selectors3 := []model.CategoryInfo{ci1, ci4}
	selectors4 := []model.CategoryInfo{ci1}
	selectors5 := []model.CategoryInfo{ci1, ci5}
	selectors6 := []model.CategoryInfo{ci1, ci5, ci3}
	selectors7 := []model.CategoryInfo{ci1, ci5, ci4}
	selectors8 := []model.CategoryInfo{ci1, ci5, ci3, ci4}
	selectors9 := []model.CategoryInfo{ci1, ci2, ci5, ci3, ci4}
	selectors10 := []model.CategoryInfo{ci1, ci2, ci5, ci6, ci3, ci4}

	nselectors1 := []model.CategoryInfo{ci5}
	nselectors2 := []model.CategoryInfo{ci1, ci5, ci6}
	nselectors3 := []model.CategoryInfo{ci5, ci3, ci6}
	nselectors4 := []model.CategoryInfo{ci1, ci2, ci4, ci6}
	nselectors5 := []model.CategoryInfo{ci5, ci3, ci4}

	matchingSelectors := [][]model.CategoryInfo{
		selectors1,
		selectors2,
		selectors3,
		selectors4,
		selectors5,
		selectors6,
		selectors7,
		selectors8,
		selectors9,
		selectors10,
	}
	notMatchingSelectors := [][]model.CategoryInfo{
		nselectors1,
		nselectors2,
		nselectors3,
		nselectors4,
		nselectors5,
	}
	for _, ms := range matchingSelectors {
		if !model.CategoryMatch(labels, ms) {
			t.Fatalf("expected selectors %+v to match labels %+v", ms, labels)
		}
	}
	for _, nms := range notMatchingSelectors {
		if model.CategoryMatch(labels, nms) {
			t.Fatalf("expected selectors %+v to NOT match labels %+v", nms, labels)
		}
	}
}

func TestValidateCategory(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		expected []string
	}{
		{"Empty", []string{}, nil},
		{"Empty as a value", []string{""}, nil},
		{"Space as a value", []string{" "}, nil},
		{"Spaces as a value", []string{"   ", " "}, nil},
		{"Normalize values", []string{"a", " b", "c ", " d ", "e  e"}, []string{"a", "b", "c", "d", "e  e"}},
		{"Collision during normalize 1", []string{" a", "a"}, nil},
		{"Collision during normalize 2", []string{"a ", "a"}, nil},
		{"Collision during normalize 3", []string{" a ", "a"}, nil},
		{"Collision during normalize 4", []string{"A", "a"}, nil},
		{"Capitalzie", []string{"A ", " b", "  C c "}, []string{"A", "b", "C c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat := model.Category{
				BaseModel: model.BaseModel{},
				Name:      "Name",
				Purpose:   "",
				Values:    tt.values,
			}
			err := model.ValidateCategory(&cat)

			if tt.expected == nil {
				require.Error(t, err, "ValidateCategory() expected to fail")
			} else {
				require.NoError(t, err, "ValidateCategory() failed %s, but expected to be OK")

				sort.Strings(tt.expected)
				sort.Strings(cat.Values)
				if !reflect.DeepEqual(tt.expected, cat.Values) {
					got := strings.Join(cat.Values, ",")
					exp := strings.Join(tt.values, ",")
					t.Errorf("ValidateCategory() values got %s, expected %s", got, exp)
				}
			}
		})
	}
}
