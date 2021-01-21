package model_test

import (
	"cloudservices/common/model"
	"testing"
)

// TestExecuteEdgeUpgrade will test Script struct
func TestExecuteEdgeUpgrade(t *testing.T) {
	var x interface{}
	x = model.ExecuteEdgeUpgradeData{
		EdgeID: "foo",
	}

	edgeID := model.GetEdgeID(x)

	if edgeID != nil {
		t.Fatal("expect edgeID = nil")
	}

	y, ok := x.(model.ExecuteEdgeUpgradeData)

	if !ok {
		t.Fatal("expect x of type model.ExecuteEdgeUpgradeData")
	}
	edgeID = &y.EdgeID
	if edgeID == nil {
		t.Fatal("expect edgeID != nil")
	}

}
