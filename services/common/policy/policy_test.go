package policy_test

import (
	"cloudservices/common/policy"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPolicy(t *testing.T) {
	policies := policy.Policies{
		{Name: "infra", Path: "/serviceDomain:.*"},
		{Name: "project", Path: "/serviceDomain:.*/project:.*"},
		{Name: "infra-project", Path: "/serviceDomain:.*/project:.*/service:kafka"},
		{Name: "infra-project", Path: "/serviceDomain:.*/project:.*/service:istio/instance:.*/binding:.*"}, // exact match for service
		{Name: "infra", Path: "/serviceDomain:.*/project:.*/service:.*/instance:.*/binding:.*"},
	}
	policyMgr := policy.NewManager()
	err := policyMgr.LoadPolicies(policies)
	require.NoError(t, err)
	policyMgr.DumpPolicies()

	testData := []struct {
		path           string
		expectedPolicy string
	}{
		{
			path:           "/serviceDomain:4da95e59-d6f9-4003-90e0-e85c5ecbf996/project:2faaf828-ef97-4c31-b7da-92902b25a108/application:aa5300a6-77a0-355c-a001-a325508d9790",
			expectedPolicy: "project",
		},
		{
			path:           "/serviceDomain:4da95e59-d6f9-4003-90e0-e85c5ecbf996/project:2faaf828-ef97-4c31-b7da-92902b25a108/service:istio",
			expectedPolicy: "project",
		},
		{
			path:           "/serviceDomain:4da95e59-d6f9-4003-90e0-e85c5ecbf996/testinfra/service:kafka",
			expectedPolicy: "infra",
		},
		{
			path:           "/serviceDomain:4da95e59-d6f9-4003-90e0-e85c5ecbf996/project:2faaf828-ef97-4c31-b7da-92902b25a108/service:kafka",
			expectedPolicy: "infra-project",
		},
		{
			path:           "/serviceDomain:e499de55-97b1-4542-89a9-dbaff46ed0a1/project:72b371c9-5cb0-481c-a9fe-d91e5e5175af/service:istio/instance:74f4f570-53a1-328b-a972-0d3d1559772a/binding:7f014ccd-604c-4237-83d3-4cad1bcd5896/status",
			expectedPolicy: "infra-project",
		},
		{
			path:           "/serviceDomain:e499de55-97b1-4542-89a9-dbaff46ed0a1/project:72b371c9-5cb0-481c-a9fe-d91e5e5175af/service:Prometheus/instance:74f4f570-53a1-328b-a972-0d3d1559772a/binding:7f014ccd-604c-4237-83d3-4cad1bcd5896/status",
			expectedPolicy: "infra",
		},
	}
	for _, td := range testData {
		policy, err := policyMgr.GetPolicy(td.path)
		require.NoError(t, err)
		t.Logf("Policy: %+v\n", policy)
		require.Equal(t, td.expectedPolicy, policy.Name, "Failed for %s", td.path)
	}
}
