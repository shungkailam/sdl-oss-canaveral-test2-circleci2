package router_test

import (
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/tenantpool/testhelper"
)

var (
	TenantPoolEdgeProvisioner *testhelper.TestEdgeProvisioner
)

func init() {
	TenantPoolEdgeProvisioner = testhelper.NewTestEdgeProvisioner()
	apitesthelper.StartServices(&apitesthelper.StartServicesConfig{StartPort: 9060, EdgeProvisioner: TenantPoolEdgeProvisioner})
}
