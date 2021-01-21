package model_test

import (
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestCloudCreds will test CloudCreds struct
func TestCloudCreds(t *testing.T) {

	var tenantID = "tenant-id-waldot"
	var awsCredential = model.AWSCredential{
		AccessKey: "foo",
		Secret:    "bar",
	}
	var gcpCredential = model.GCPCredential{
		Type:                    "type-val",
		ProjectID:               "project-id-val",
		PrivateKeyID:            "private-key-id-val",
		PrivateKey:              "private-key-val",
		ClientEmail:             "client-email-val",
		ClientID:                "client-id-val",
		AuthURI:                 "auth-uri-val",
		TokenURI:                "token-uri-val",
		AuthProviderX509CertURL: "auth-provider-x509-cert-url-val",
		ClientX509CertURL:       "client-x509-cert-url",
	}
	now := timeNow(t)
	cloudCredsList := []model.CloudCreds{
		{
			BaseModel: model.BaseModel{
				ID:        "aws-cloud-creds-id",
				TenantID:  tenantID,
				Version:   5,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:          "aws-cloud-creds-name",
			Type:          "AWS",
			Description:   "aws-cloud-creds-desc",
			AWSCredential: &awsCredential,
			GCPCredential: nil,
		},
		{
			BaseModel: model.BaseModel{
				ID:        "gcp-cloud-creds-id",
				TenantID:  tenantID,
				Version:   0,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:          "gcp-cloud-creds-name",
			Type:          "GCP",
			Description:   "aws-cloud-creds-desc",
			AWSCredential: nil,
			GCPCredential: &gcpCredential,
		},
	}
	cloudCredsStrings := []string{
		`{"id":"aws-cloud-creds-id","version":5,"tenantId":"tenant-id-waldot","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"aws-cloud-creds-name","type":"AWS","description":"aws-cloud-creds-desc","awsCredential":{"accessKey":"foo","secret":"bar"}}`,
		`{"id":"gcp-cloud-creds-id","tenantId":"tenant-id-waldot","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"gcp-cloud-creds-name","type":"GCP","description":"aws-cloud-creds-desc","gcpCredential":{"type":"type-val","project_id":"project-id-val","private_key_id":"private-key-id-val","private_key":"private-key-val","client_email":"client-email-val","client_id":"client-id-val","auth_uri":"auth-uri-val","token_uri":"token-uri-val","auth_provider_x509_cert_url":"auth-provider-x509-cert-url-val","client_x509_cert_url":"client-x509-cert-url"}}`,
	}

	var version float64 = 5
	cloudCredsMaps := []map[string]interface{}{
		{
			"id":          "aws-cloud-creds-id",
			"version":     version,
			"tenantId":    tenantID,
			"name":        "aws-cloud-creds-name",
			"description": "aws-cloud-creds-desc",
			"type":        "AWS",
			"awsCredential": map[string]interface{}{
				"accessKey": "foo",
				"secret":    "bar",
			},
			"createdAt": NOW,
			"updatedAt": NOW,
		},
		{
			"id":          "gcp-cloud-creds-id",
			"tenantId":    tenantID,
			"name":        "gcp-cloud-creds-name",
			"description": "aws-cloud-creds-desc",
			"type":        "GCP",
			"gcpCredential": map[string]interface{}{
				"type":                        "type-val",
				"project_id":                  "project-id-val",
				"client_id":                   "client-id-val",
				"token_uri":                   "token-uri-val",
				"client_x509_cert_url":        "client-x509-cert-url",
				"private_key_id":              "private-key-id-val",
				"private_key":                 "private-key-val",
				"client_email":                "client-email-val",
				"auth_uri":                    "auth-uri-val",
				"auth_provider_x509_cert_url": "auth-provider-x509-cert-url-val",
			},
			"createdAt": NOW,
			"updatedAt": NOW,
		},
	}

	for i, cloudCreds := range cloudCredsList {
		cloudCredsData, err := json.Marshal(cloudCreds)
		require.NoError(t, err, "failed to marshal cloudCreds")

		if cloudCredsStrings[i] != string(cloudCredsData) {
			t.Fatalf("cloudCreds json string mismatch: %s", string(cloudCredsData))
		}

		var sdoc interface{}
		sdoc = model.ScopedEntity{
			Doc:     cloudCreds,
			EdgeIDs: []string{},
		}
		_, ok := sdoc.(model.ScopedEntity)
		require.True(t, ok, "sdoc should be a scoped entity")

		// alternative form: m := make(map[string]interface{})
		m := map[string]interface{}{}
		err = json.Unmarshal(cloudCredsData, &m)
		require.NoError(t, err, "failed to unmarshal cloudCreds to map")

		// reflect.DeepEqual fails on equivalent slices here,
		// so use weaker marshal equal
		if !model.MarshalEqual(&m, &cloudCredsMaps[i]) {
			t.Fatalf("cloudCreds map marshal mismatch: %+v", m)
		}
	}

}
