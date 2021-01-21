package cfssl

import (
	"cloudservices/cloudmgmt/generated/cfssl/client"
	"cloudservices/cloudmgmt/generated/cfssl/client/operations"
	"cloudservices/cloudmgmt/generated/cfssl/models"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/golang/glog"
)

const (
	CfsslTimeout     = time.Second * 60
	UnitTestTenantID = "tenant-id-cms-unit-test"
)

var clnt *client.Cfssl
var mxCfssl sync.Mutex
var unitTestSharedCertsMap = make(map[string]*CertResponse)

// CertResponse represents cfssl get cert response struct
type CertResponse struct {
	Cert   string `json:"Certificate"`
	Key    string `json:"PrivateKey"`
	CACert string `json:"CACertificate"`
}

// KeyPair public/private key pair used in ssh
type KeyPair struct {
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

// allow some code path to be skipped/optimized to improve testing performance
// in unit test mode we will only create one root CA and share them across all test tenants
// similarly we will only create one set of edge certs and share them across all test edges
// To simplify synchronization, we will use a fixed
// unit test tenant id in the DB, say tenant-id-cms-unit-test
// all instances of cms will try to create it first
// and ignore error.
// After that cms will just clone root ca from it.
func IsUnitTestMode() bool {
	return os.Getenv("UNIT_TEST_MODE") == "1"
}

// GenerateKeyPair generate KeyPair
func GenerateKeyPair() (keyPair KeyPair, err error) {
	dir, err := ioutil.TempDir("", "keypair")
	if err != nil {
		return
	}
	defer os.RemoveAll(dir)
	tmpFilePath := filepath.Join(dir, "key")
	cmd := exec.Command("/usr/bin/ssh-keygen", "-t", "ecdsa", "-f", tmpFilePath, "-P", "", "-b", "521")
	err = cmd.Run()
	if err != nil {
		return
	}
	bs, err := ioutil.ReadFile(tmpFilePath)
	if err != nil {
		return
	}
	keyPair.PrivateKey = string(bs)
	bs, err = ioutil.ReadFile(fmt.Sprintf("%s.pub", tmpFilePath))
	if err != nil {
		return
	}
	keyPair.PublicKey = string(bs)
	return
}

func getIPv4Address(host string) (net.IP, error) {
	addrs, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		ip := addr.To4()
		if ip != nil {
			return ip, nil
		}
	}
	return nil, errors.New("No IPv4 address found")
}

func getIPv4AddressWithDefault(host string, dflt string) string {
	ip, err := getIPv4Address(host)
	if err != nil {
		glog.Warningf("Host: %s is not found, defaulting to localhost", host)
		return dflt
	}
	return fmt.Sprintf("%s", ip)
}

// GetCert gets cert, private key, cacert from cfssl using /certificates endpoint
func GetCert(tenantID string, certType string) (*CertResponse, error) {
	if IsUnitTestMode() {
		return getCertUnitTest(tenantID, certType)
	} else {
		return getCertNormal(tenantID, certType)
	}
}

// In unit test we will share the same certs per certType.
// We will use UnitTestTenantID to create the shared certs.
// For this to work, test tenants all need to have the same
// root CA, kms data key etc. This is done in account tenantApi.go
func getCertUnitTest(tenantID string, certType string) (cr *CertResponse, err error) {
	mxCfssl.Lock()
	defer mxCfssl.Unlock()
	cr = unitTestSharedCertsMap[certType]
	if cr != nil {
		return
	}
	cr, err = getCertNormal(UnitTestTenantID, certType)
	if err == nil {
		unitTestSharedCertsMap[certType] = cr
	}
	return
}

func getCertNormal(tenantID string, certType string) (*CertResponse, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), CfsslTimeout)
	defer cancelFn()
	params := &operations.CertificatesPostParams{
		Body: &models.CertificatePostParams{
			TenantID: &tenantID,
			Type:     certType,
		},
		Context: ctx,
	}

	resp, err := clnt.Operations.CertificatesPost(params)
	if err != nil {
		glog.Errorf("Failed to get certificates for tenantID: %s with error: %s", tenantID, err)
		return nil, err
	}

	certResp := CertResponse{
		Cert:   resp.Payload.Certificate,
		Key:    resp.Payload.PrivateKey,
		CACert: resp.Payload.CACertificate,
	}
	return &certResp, nil
}

// CreateRootCA creates a root CA for the tenant which will be used to generate certs.
func CreateRootCA(tenantID string) error {
	glog.Infof("Creating new ROOT CA for tenant: %s", tenantID)
	ctx, cancelFn := context.WithTimeout(context.Background(), CfsslTimeout)
	defer cancelFn()
	params := &operations.RootCAPostParams{
		Body: &models.CertificatePostParams{
			TenantID: &tenantID,
		},
		Context: ctx,
	}
	if resp, err := clnt.Operations.RootCAPost(params); err != nil {
		glog.Errorf("Failed to create Root CA for tenant: %s with error: %s", tenantID, resp.Error())
		return err
	}
	return nil
}

func InitCfssl(protocol, host string, port int) {
	transCfg := client.TransportConfig{
		Host:     fmt.Sprintf("%s:%d", host, port),
		BasePath: client.DefaultBasePath,
		Schemes:  []string{protocol},
	}
	fmt.Printf(">>> cfssl.init, host:port=%s, protocol=%s\n", transCfg.Host, transCfg.Schemes[0])
	clnt = client.NewHTTPClientWithConfig(nil, &transCfg)
}
