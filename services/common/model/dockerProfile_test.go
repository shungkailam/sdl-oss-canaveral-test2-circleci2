package model_test

import (
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestdockerCreds will test DockerProfile struct
func TestDockerProfile(t *testing.T) {

	var tenantID = "tenant-id-waldot"

	now := timeNow(t)
	dockerProfileList := []model.DockerProfile{
		{
			BaseModel: model.BaseModel{
				ID:        "aws-cloud-creds-id",
				TenantID:  tenantID,
				Version:   5,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:        "foo",
			Type:        "AWS",
			Description: "bar",
			Credentials: "{\"AccessKeyId\":\"AWS-Access\",\"SecretAccessKey\":\"AWS-SecretAccessKey\",\"Account\":\"AWS-test\",\"Region\":\"us-west-2\",\"Server\":\"aws-server\",\"User\":\"aws-user\",\"Pwd\":\"aws-pwd\",\"Email\":\"aws-email\"}",
		},
	}
	dockerProfileStrings := []string{
		`{"id":"aws-cloud-creds-id","version":5,"tenantId":"tenant-id-waldot","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"foo","description":"bar","type":"AWS","server":"","userName":"","email":"","pwd":"","credentials":"{\"AccessKeyId\":\"AWS-Access\",\"SecretAccessKey\":\"AWS-SecretAccessKey\",\"Account\":\"AWS-test\",\"Region\":\"us-west-2\",\"Server\":\"aws-server\",\"User\":\"aws-user\",\"Pwd\":\"aws-pwd\",\"Email\":\"aws-email\"}"}`,
	}

	var version float64 = 5
	dockerProfileMaps := []map[string]interface{}{
		{
			"id":          "aws-cloud-creds-id",
			"version":     version,
			"tenantId":    tenantID,
			"name":        "foo",
			"type":        "AWS",
			"description": "bar",
			"server":      "",
			"userName":    "",
			"email":       "",
			"pwd":         "",
			"credentials": "{\"AccessKeyId\":\"AWS-Access\",\"SecretAccessKey\":\"AWS-SecretAccessKey\",\"Account\":\"AWS-test\",\"Region\":\"us-west-2\",\"Server\":\"aws-server\",\"User\":\"aws-user\",\"Pwd\":\"aws-pwd\",\"Email\":\"aws-email\"}",
			"createdAt":   NOW,
			"updatedAt":   NOW,
		},
	}

	for i, dockerProfile := range dockerProfileList {
		dockerProfileData, err := json.Marshal(dockerProfile)
		require.NoError(t, err, "failed to marshal DockerProfile")

		if dockerProfileStrings[i] != string(dockerProfileData) {
			t.Fatalf("dockerProfile json string mismatch: %s", string(dockerProfileData))
		}
		// alternative form: m := make(map[string]interface{})
		m := map[string]interface{}{}
		err = json.Unmarshal(dockerProfileData, &m)
		require.NoError(t, err, "failed to unmarshal DockerProfile to map")

		// reflect.DeepEqual fails on equivalent slices here,
		// so use weaker marshal equal
		if !model.MarshalEqual(&m, &dockerProfileMaps[i]) {
			t.Fatalf("DockerProfile map marshal mismatch: %+v", m)
		}
	}

}
