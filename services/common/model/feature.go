package model

// Features available for an edge.
// swagger:model Features
type Features struct {
	URLupgrade            bool `json:"urlUpgrade"`
	HighMemAlert          bool `json:"highMemAlert"`
	RealTimeLogs          bool `json:"realTimeLogs"`
	MultiNodeAware        bool `json:"multiNodeAware"`
	DownloadAndUpgrade    bool `json:"downloadAndUpgrade"`
	RemoteSSH             bool `json:"remoteSSH"`
	ProjectUserKubeConfig bool `json:"projectUserKubeConfig"`
}
