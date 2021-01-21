package model

// swagger:parameters K8sDashboardGetAdminToken K8sDashboardGetViewonlyToken K8sDashboardGetUserToken K8sDashboardGetAdminKubeConfig K8sDashboardGetUserKubeConfig K8sDashboardGetViewonlyKubeConfig K8sDashboardGetViewonlyUsers K8sDashboardAddViewonlyUsers K8sDashboardRemoveViewonlyUsers
// in: header
type k8sDashboardAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// K8sDashboardTokenResponsePayload response for k8s dashboard get token call
// swagger:model K8sDashboardTokenResponsePayload
type K8sDashboardTokenResponsePayload struct {
	// required: true
	Token string `json:"token"`
}

// Ok
// swagger:response K8sDashboardTokenResponse
type K8sDashboardTokenResponse struct {
	// in: body
	Payload *K8sDashboardTokenResponsePayload
}

type cfgClusterData struct {
	Server string `json:"server"`
	CAData string `json:"certificate-authority-data"`
}
type cfgCluster struct {
	Name    string         `json:"name"`
	Cluster cfgClusterData `json:"cluster"`
}
type cfgUserData struct {
	Token string `json:"token"`
}
type cfgUser struct {
	Name string      `json:"name"`
	User cfgUserData `json:"user"`
}

type cfgContextData struct {
	Cluster string `json:"cluster"`
	User    string `json:"user"`
}
type cfgContext struct {
	Name    string         `json:"name"`
	Context cfgContextData `json:"context"`
}

type KubeConfig struct {
	// required: true
	Kind string `json:"kind"`
	// required: true
	APIVersion string `json:"apiVersion"`
	// required: true
	Clusters []cfgCluster `json:"clusters"`
	// required: true
	Users []cfgUser `json:"users"`
	// required: true
	Contexts []cfgContext `json:"contexts"`
	// required: true
	CurrentContext string `json:"current-context"`
}

func MakeKubeConfig(name, token, server, ca string) *KubeConfig {
	currentCtx := name + "-context"
	username := "default-user-" + name
	return &KubeConfig{
		Kind:       "Config",
		APIVersion: "v1",
		Clusters: []cfgCluster{
			{
				Name: name,
				Cluster: cfgClusterData{
					Server: server,
					CAData: ca,
				},
			},
		},
		Users: []cfgUser{
			{
				Name: username,
				User: cfgUserData{
					Token: token,
				},
			},
		},
		Contexts: []cfgContext{
			{
				Name: currentCtx,
				Context: cfgContextData{
					Cluster: name,
					User:    username,
				},
			},
		},
		CurrentContext: currentCtx,
	}
}

// KubeConfigPayload response for k8s dashboard get kube config call
// swagger:model KubeConfigPayload
type KubeConfigPayload struct {
	// string representation of kubeconfig yaml
	// required: true
	KubeConfig string `json:"kubeconfig"`
}

// Ok
// swagger:response K8sDashboardKubeConfigResponse
type K8sDashboardKubeConfigResponse struct {
	// in: body
	Payload *KubeConfigPayload
}

// Ok
// swagger:response K8sDashboardViewonlyUserListResponse
type K8sDashboardViewonlyUserListResponse struct {
	// in: body
	// required: true
	Payload *K8sDashboardViewonlyUserListPayload
}

// K8sDashboardViewonlyUserListPayload is the payload for K8sDashboardViewonlyUserListResponse
type K8sDashboardViewonlyUserListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of HTTP Service Proxies
	// required: true
	ViewonlyUserList []User `json:"result"`
}

type K8sDashboardViewonlyUserParams struct {
	UserIDs []string `json:"userIds"`
}

// K8sDashboardViewonlyUserRequestParams userIds to Add / Remove
// swagger:parameters K8sDashboardAddViewonlyUsers K8sDashboardRemoveViewonlyUsers
type K8sDashboardViewonlyUserRequestParams struct {
	// in: body
	// required: true
	Payload *K8sDashboardViewonlyUserParams
}

// Ok
// swagger:response K8sDashboardViewonlyUserUpdateResponse
type K8sDashboardViewonlyUserUpdateResponse struct {
	// in: body
	// required: true
	Payload *K8sDashboardViewonlyUserUpdatePayload
}

// K8sDashboardViewonlyUserUpdatePayload - empty for now
type K8sDashboardViewonlyUserUpdatePayload struct {
}
