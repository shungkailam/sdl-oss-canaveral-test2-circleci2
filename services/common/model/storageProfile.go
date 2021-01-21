package model

const (
	NutanixVolumesType = "NutanixVolumes"
	EBSType            = "EBS"
	VSphereType        = "vSphere"
)

// NutanixVolumesStorageProfileConfig - struct for Nutanix volume Storage Profile config.
type NutanixVolumesStorageProfileConfig struct {
	// required: true
	PrismElementClusterVIP string `json:"prismElementClusterVIP" db:"pe_cluster_vip"`
	// required: true
	PrismElementUserName string `json:"prismElementUserName" db:"pe_username"`
	// required: true
	PrismElementPassword string `json:"prismElementPassword" db:"pe_password"`
	// required: true
	PrismElementClusterPort int64 `json:"prismElementClusterPort" db:"pe_cluster_port"`
	// required: true
	DataServicesIP string `json:"dataServicesIP" db:"dataservices_ip"`
	// required: true
	DataServicesPort int64 `json:"dataServicesPort" db:"dataservices_port"`
	// required: true
	StorageContainerName string `json:"storageContainerName" db:"storage_container_name"`
	// required: false
	FlashMode bool `json:"flashMode" db:"flash_mode"`
}

// EBSStorageProfileConfig - struct for AWS EBS Storage Profile config.
type EBSStorageProfileConfig struct {
	// required: true
	Type string `json:"type" db:"type"`
	// required: true
	IOPSPerGB string `json:"iops_per_gb" db:"iops_per_gb"`
	// required: true
	Encrypted string `json:"encrypted" db:"encrypted"`
}

// VSphereStorageProfileConfig - struct for VMware Vsphere Storage Profile config.
type VSphereStorageProfileConfig struct {
}

// StorageProfile is the object model for storage profile.
// swagger:model StorageProfile
type StorageProfile struct {
	// required: true
	BaseModel
	//
	// Name for the storage profile.
	//
	// required: true
	Name string `json:"name" validate:"range=1:200"`
	//
	// Storage type for this Storage profile.
	//
	// enum: NutanixVolumes,EBS,vSphere
	// required: true
	Type string `json:"type" validate:"options=NutanixVolumes:EBS:vSphere"`
	// the following representation for credential is not ideal,
	// but we are constrained by what tsoa supports
	//
	// Storage config for the storage profile.
	// Required when type == NutanixVolumes.
	//
	NutanixVolumesConfig *NutanixVolumesStorageProfileConfig `json:"nutanixVolumesConfig,omitempty"`
	//
	// Storage config for the storage profile.
	// Required when type == EBS.
	//
	EBSStorageConfig *EBSStorageProfileConfig `json:"ebsStorageConfig,omitempty"`
	//
	// Storage config for the storage profile.
	// Required when type == vSphere.
	//
	VSphereStorageConfig *VSphereStorageProfileConfig `json:"vSphereStorageConfig,omitempty"`
	//
	// ntnx:ignore
	//
	// Internal Flag - encrypted - for internal migration use
	//
	// required: false
	IFlagEncrypted bool `json:"iflagEncrypted,omitempty" db:"iflag_encrypted"`
	//
	// Flag to specify if it is default storage profile
	//
	// required: false
	IsDefault bool `json:"isDefault,omitempty" db:"isdefault"`
}

// StorageProfileCreateParam is StorageProfile used as API parameter
// swagger:parameters StorageProfileCreate
type StorageProfileCreateParam struct {
	// Description for the storage profile.
	// in: body
	// required: true
	Body *StorageProfile `json:"body"`
}

// StorageProfileUpdateParam is StorageProfile used as API parameter
// swagger:parameters StorageProfileUpdate
type StorageProfileUpdateParam struct {
	// in: body
	// required: true
	Body *StorageProfile `json:"body"`
}

// Ok
// swagger:response StorageProfileListResponse
type StorageProfileListResponse struct {
	// in: body
	// required: true
	Payload *StorageProfileListResponsePayload
}

// payload for StorageProfileListResponse
type StorageProfileListResponsePayload struct {
	// required: true
	EntityListResponsePayload
	// list of storage profiles
	// required: true
	StorageProfileList []StorageProfile `json:"result"`
}

// swagger:parameters SvcDomainGetStorageProfiles StorageProfileCreate StorageProfileUpdate
// in: header
type storageProfileAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}
