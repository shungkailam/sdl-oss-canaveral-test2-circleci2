package cfssl_test

import (
	"cloudservices/cloudmgmt/cfssl"
	"cloudservices/cloudmgmt/generated/cfssl/models"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetCert(t *testing.T) {
	t.Log("running TestGetCert test")

	cfssl.InitCfssl("https", "cfssl-test.ntnxsherlock.com", 443)

	certResp, err := cfssl.GetCert("tenant-id-waldot", models.CertificatePostParamsTypeClient)
	require.NoError(t, err, "Get cert failed")
	t.Logf("Got cert response: %+v", certResp)
}
