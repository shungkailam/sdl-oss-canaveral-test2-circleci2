package model

type TenantRootCA struct {
	BaseModel
	//
	// Root CA certificate for the tenant.
	//
	// required: true
	Certificate string `json:"certificate" db:"certificate" validate:"range=0:4096"`
	//
	// Encrypted Root CA private key.
	//
	// required: true
	PrivateKey string `json:"privateKey" db:"private_key" validate:"range=0:4096"`
	//
	// Encrypted AWS Data Key.
	//
	// required: true
	AWSDataKey string `json:"awsDataKey" db:"aws_data_key" validate:"range=0:4096"`
}
