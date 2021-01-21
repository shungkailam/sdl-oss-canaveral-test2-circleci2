package base_test

import (
	"cloudservices/common/base"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func TestOrderByHelper(t *testing.T) {
	obh := base.NewOrderByHelper()
	entityKey := "entitiKey"
	obh.Setup(entityKey, []string{"k1:rk1", "k2"})
	orderByKeys := obh.GetOrderByKeys(entityKey)
	expectedOrderByKeys := []string{"k1", "k2"}
	if !reflect.DeepEqual(expectedOrderByKeys, orderByKeys) {
		t.Fatal("expect keys to match")
	}
	keyMap := obh.GetLogicalKeyMap(entityKey)
	expectedKeyMap := map[string]string{"k1": "rk1", "k2": "k2"}
	if !reflect.DeepEqual(expectedKeyMap, keyMap) {
		t.Fatal("expect keys to match")
	}
	var keys []string
	keys = obh.GetOrderByKeys("foo")
	if keys != nil {
		t.Fatal("expect get keys for foo to return nil")
	}

	var c string
	var err error
	defaultClause := "default"
	c, err = obh.BuildOrderByClause("foo", nil, defaultClause, "alias", nil)
	require.Error(t, err, "expect foo entity type to be unknown")
	aliasMap := map[string]string{"bar1": "alias1"}
	c, err = obh.BuildOrderByClause("foo", []string{"bar"}, defaultClause, "alias", aliasMap)
	require.Error(t, err, "expect foo entity type to be unknown")

	keys = []string{"bar:bar1", "baz"}
	obh.Setup("foo", keys)

	expectedKeys := []string{"bar", "baz"}
	keys2 := obh.GetOrderByKeys("foo")
	if !reflect.DeepEqual(expectedKeys, keys2) {
		t.Fatal("expect keys to match")
	}

	c, err = obh.BuildOrderByClause("foo", []string{"bar"}, defaultClause, "alias", nil)
	require.NoError(t, err)
	if c != "ORDER BY alias.bar1" {
		t.Fatal("expect foo clause to match")
	}
	c, err = obh.BuildOrderByClause("foo", []string{"bar desc"}, defaultClause, "", aliasMap)
	require.NoError(t, err)
	if c != "ORDER BY alias1.bar1 DESC" {
		t.Fatal("expect foo clause to match")
	}
	c, err = obh.BuildOrderByClause("foo", []string{"bar", "baz"}, defaultClause, "", aliasMap)
	require.NoError(t, err)
	if c != "ORDER BY alias1.bar1, baz" {
		t.Fatal("expect foo clause to match")
	}
	c, err = obh.BuildOrderByClause("foo", []string{"bar desc", "baz"}, defaultClause, "alias", aliasMap)
	require.NoError(t, err)
	if c != "ORDER BY alias1.bar1 DESC, alias.baz" {
		t.Fatal("expect foo clause to match")
	}
	c, err = obh.BuildOrderByClause("foo", []string{"bar desc", "baz desc"}, defaultClause, "", nil)
	require.NoError(t, err)
	if c != "ORDER BY bar1 DESC, baz DESC" {
		t.Fatal("expect foo clause to match")
	}

	c, err = obh.BuildOrderByClause("foo", []string{"bar", "duh", "baz"}, defaultClause, "", nil)
	require.Errorf(t, err, "expect order by key %s to be unknown", "duh")
}
