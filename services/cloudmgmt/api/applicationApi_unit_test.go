package api

import (
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
)

func testAuthCtx() *base.AuthContext {
	return &base.AuthContext{
		TenantID: "foo",
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"projects": []model.ProjectRole{
				{
					ProjectID: "12345",
					Role:      model.ProjectRoleAdmin,
				},
			},
		},
	}
}

func TestValidateDataIfcEndpointsCountLimits(t *testing.T) {
	ctx := context.WithValue(context.Background(), base.AuthContextKey, testAuthCtx())
	ifcIn := model.DataSourceIfcInfo{Kind: model.DataIfcEndpointKindIn}
	ifcOut := model.DataSourceIfcInfo{Kind: model.DataIfcEndpointKindOut}

	dataSrcIfcIn := model.DataSource{}
	dataSrcIfcIn.IfcInfo = &ifcIn

	dataSrcIfcOut := model.DataSource{}
	dataSrcIfcOut.IfcInfo = &ifcOut

	// IfcInfo=nil
	dataSrc := model.DataSource{}

	testCases := []struct {
		dataSources         []model.DataSource
		limitIns, limitOuts int
		errExpected         bool
	}{
		{dataSources: []model.DataSource{dataSrc}, limitIns: 1, limitOuts: 1, errExpected: false},
		{dataSources: []model.DataSource{dataSrcIfcIn}, limitIns: 1, limitOuts: 0, errExpected: false},
		{dataSources: []model.DataSource{dataSrcIfcOut}, limitIns: 1, limitOuts: 1, errExpected: false},
		{dataSources: []model.DataSource{dataSrcIfcOut}, limitIns: 0, limitOuts: 1, errExpected: false},
		{dataSources: []model.DataSource{dataSrcIfcIn, dataSrcIfcOut}, limitIns: 1, limitOuts: 1, errExpected: false},
		{dataSources: []model.DataSource{dataSrcIfcIn, dataSrcIfcOut}, limitIns: 1, limitOuts: 0, errExpected: true},
		{dataSources: []model.DataSource{dataSrcIfcIn, dataSrcIfcOut, dataSrc}, limitIns: 0, limitOuts: 1, errExpected: true},
		{dataSources: []model.DataSource{dataSrcIfcIn, dataSrcIfcOut, dataSrc}, limitIns: 0, limitOuts: 0, errExpected: true},
	}

	for _, testCase := range testCases {
		err := ValidateDataIfcEndpointsCountLimits(ctx, testCase.dataSources, testCase.limitIns, testCase.limitOuts)
		if testCase.errExpected != (err != nil) {
			t.Fatalf("expected %v but got %v", testCase.errExpected, err != nil)
		}
	}
}
