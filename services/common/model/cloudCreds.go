package model

import (
	"cloudservices/common/base"
)

const (
	AWSType = "AWS"
	GCPType = "GCP"
	AZType  = "Azure"
)

//
// AWSCredential - AWS access key and secret credentials.
//
// swagger:model AWSCredential
type AWSCredential struct {
	//
	// Provide the AWS Access Key Id for programmatic access to the AWS services
	//
	// required: true
	AccessKey string `json:"accessKey" db:"accessKey"`
	//
	// Provide the AWS Secret Key for programmatic access to the AWS services
	//
	// required: true
	Secret string `json:"secret" db:"secret"`
}

// GCPCredential - Google Cloud Platform credentials.
type GCPCredential struct {
	// required: true
	Type string `json:"type" db:"type"`
	//
	// The project resource is the base-level organizing entity in Google Cloud Platform
	// Specify the unique Id for the project in GCP 
	//
	// required: true
	ProjectID string `json:"project_id" db:"project_id"`
	// required: true
	PrivateKeyID string `json:"private_key_id" db:"private_key_id"`
	// required: true
	PrivateKey string `json:"private_key" db:"private_key"`
	// required: true
	ClientEmail string `json:"client_email" db:"client_email"`
	// required: true
	ClientID string `json:"client_id" db:"client_id"`
	// required: true
	AuthURI string `json:"auth_uri" db:"auth_uri"`
	// required: true
	TokenURI string `json:"token_uri" db:"token_uri"`
	//
	// Google service account key generated using the gcloud command
	// GCP service account key formats depend on when you use the gcloud command or the REST API/client library
	// to generate the key. The gcloud format is supported in this case. 
	// Use the key generated using gcloud command as is, for all field values as follows.
	//
	// Type is set to 'service_account' when you generate the key using gcloud command
	//
	// required: true
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url" db:"auth_provider_x509_cert_url"`
	// required: true
	ClientX509CertURL string `json:"client_x509_cert_url" db:"client_x509_cert_url"`
}

// AZCredential - Azure credentials.
// swagger:model AZCredential
type AZCredential struct {
	//
	// Azure storage account name and access key. 
	// When you create a storage account, Azure generates 2 access keys. Provide the primary access key here. 
	//
	// required: true
	StorageAccountName string `json:"storageAccountName" db:"storageAccountName"`
	// required: true
	StorageKey string `json:"storageKey" db:"storageKey"`
}

// CloudCreds is the object model for cloud credentials.
// swagger:model CloudCreds
type CloudCreds struct {
	// required: true
	BaseModel
	//
	// Name for the cloud profile.
	//
	// required: true
	Name string `json:"name" db:"name" validate:"range=1:200"`
	//
	// Cloud type for this cloud profile. 
	// Set value to one of the following: AWS, GCP, Azure
	//
	// enum: AWS,GCP,Azure
	// required: true
	Type string `json:"type" db:"type" validate:"options=AWS:GCP:Azure"`
	//
	// Describes the cloud service profile.
	//
	// required: true
	Description string `json:"description" db:"description" validate:"range=0:200"`
	// the following representation for credential is not ideal,
	// but we are constrained by what tsoa supports
	//
	// Credential for the cloud profile.
	// Required when type == AWS.
	//
	// required:false
	AWSCredential *AWSCredential `json:"awsCredential,omitempty" db:"aws_credential"`
	//
	// Credential for the cloud profile.
	// Required when type == GCP.
	//
	// required:false
	GCPCredential *GCPCredential `json:"gcpCredential,omitempty" db:"gcp_credential"`
	//
	// Credential for the cloud profile.
	// Required when type == AZ.
	//
	// required:false
	AZCredential *AZCredential `json:"azCredential,omitempty" db:"az_credential"`
	//
	// ntnx:ignore
	//
	// Internal Flag - encrypted - for internal migration use
	//
	// required: false
	IFlagEncrypted bool `json:"iflagEncrypted,omitempty" db:"iflag_encrypted"`
}

// CloudCredsCreateParam is CloudCreds used as API parameter
// swagger:parameters CloudCredsCreate
type CloudCredsCreateParam struct {
	// Description for the cloud profile.
	// in: body
	// required: true
	Body *CloudCreds `json:"body"`
}

// CloudCredsUpdateParam is CloudCreds used as API parameter
// swagger:parameters CloudCredsUpdate CloudCredsUpdateV2
type CloudCredsUpdateParam struct {
	// in: body
	// required: true
	Body *CloudCreds `json:"body"`
}

// Ok
// swagger:response CloudCredsGetResponse
type CloudCredsGetResponse struct {
	// in: body
	// required: true
	Payload *CloudCreds
}

// Ok
// swagger:response CloudCredsListResponse
type CloudCredsListResponse struct {
	// in: body
	// required: true
	Payload *[]CloudCreds
}

// Ok
// swagger:response CloudCredsListResponseV2
type CloudCredsListResponseV2 struct {
	// in: body
	// required: true
	Payload *CloudCredsListResponsePayload
}

// payload for CloudCredsListResponseV2
type CloudCredsListResponsePayload struct {
	// required: true
	EntityListResponsePayload
	// list of cloud profiles
	// required: true
	CloudCredsList []CloudCreds `json:"result"`
}

// swagger:parameters CloudCredsList CloudProfileList CloudCredsGet CloudProfileGet CloudCredsCreate CloudProfileCreate CloudCredsUpdate CloudCredsUpdateV2 CloudProfileUpdate CloudCredsDelete CloudProfileDelete ProjectGetCloudCreds ProjectGetCloudProfiles
// in: header
type cloudCredsAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ObjectRequestBaseCloudCreds is used as websocket CloudCreds message
// swagger:model ObjectRequestBaseCloudCreds
type ObjectRequestBaseCloudCreds struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc CloudCreds `json:"doc"`
}

type CloudCredssByID []CloudCreds

func (a CloudCredssByID) Len() int           { return len(a) }
func (a CloudCredssByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a CloudCredssByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

// must only call this on unencrypted object
func (cc *CloudCreds) MaskObject() {
	if cc.Type == AWSType {
		cc.AWSCredential.maskAWSCreds()
	} else if cc.Type == GCPType {
		cc.GCPCredential.maskGCPCreds()
	}
}
func (awsCreds *AWSCredential) maskAWSCreds() {
	awsCreds.Secret = base.MaskString(awsCreds.Secret, "*", 0, 4)
}
func (gcpCreds *GCPCredential) maskGCPCreds() {
	gcpCreds.PrivateKey = base.MaskString(gcpCreds.PrivateKey, "*", 0, 4)
}

func MaskCloudCreds(cloudCredss []CloudCreds) {
	for i := 0; i < len(cloudCredss); i++ {
		cloudCredss[i].MaskObject()
	}
}
