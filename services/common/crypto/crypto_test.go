package crypto_test

import (
	"cloudservices/common/crypto"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSha256(t *testing.T) {
	p0 := "apex"
	x := crypto.GetRawSha256(p0)
	xh := "846be5fc541c5039f7972c6f0a054a3460d3f4b3ee3541991baee4027465abdb"
	if x != xh {
		t.Fatalf("raw sha256 failure\n")
	}
	x2 := crypto.GetSha256(p0)
	x2h := "86454ea5bb9fb8dfcc7c035b96f19dc9032944de02490fde30d54527c51f72a6"
	if x2 != x2h {
		t.Fatalf("sha256 failure\n")
	}

	p := "MyDarkSecret"
	ph := "$2a$10$b6yCVNpaaS9GDhbWAACFxezwDSZAMJSijtWhVCnN.kW2347Ks0Wly"
	if crypto.MatchHashAndPassword(ph, p) == false {
		t.Fatal("expect password and hash to match")
	}

	if crypto.MatchHashAndPassword(x2h, p0) == false {
		t.Fatal("expect password and hash to match")
	}

	ep, err := crypto.EncryptPassword(p)
	require.NoError(t, err)
	if ep == p {
		t.Fatal("failed to encrypt password")
	}

	if crypto.MatchHashAndPassword(p0, p0) == true {
		t.Fatal("expect password to not match self")
	}
}

func TestCryptoEncryptDecrypt(t *testing.T) {
	msg := "Hello darkness my old friend"
	key := []byte("3zTvzr3p67VC61jmV54rIYu1545x4TlY")
	cipher, err := crypto.Encrypt(msg, key)
	require.NoError(t, err, "encrypt failed")

	plain, err := crypto.Decrypt(cipher, key)
	require.NoError(t, err, "decrypt failed")
	if plain != msg {
		t.Fatalf("msg mismatch, original: %s, decrypted: %s", msg, plain)
	}
}
func TestVerifySignature(t *testing.T) {
	const certificate = `
-----BEGIN CERTIFICATE-----
MIID7TCCAtWgAwIBAgIUCFF+/sXeHhz4MZ+KsxyG9v8OybwwDQYJKoZIhvcNAQEL
BQAwazELMAkGA1UEBhMCVVMxDzANBgNVBAgTBk9yZWdvbjERMA8GA1UEBxMIUG9y
dGxhbmQxEzARBgNVBAoTCkt1YmVybmV0ZXMxCzAJBgNVBAsTAkNBMRYwFAYDVQQD
Ew1LdWJlcm5ldGVzIENBMB4XDTE4MDUxNTE3MDYwMFoXDTE5MDUxNTE3MDYwMFow
cDELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExETAPBgNVBAcTCFNh
biBKb3NlMRAwDgYDVQQKEwdOdXRhbml4MREwDwYDVQQLEwhTaGVybG9jazEUMBIG
A1UEAxMLbXF0dC1jbGllbnQwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIB
AQDRmyG+rgR9coaOU9ua0phlegzP5ffTpzWxfRzdjf5f5Z1l76cuO+jwKKN6al0w
fRaC+atuawzalrPefAnYbkT6C8PDPkIMTopo6wEZJs9/fN/zVwjRnih/yEfLTCjU
sR7xJ4tg+K1R4BZphhsCwRuh7nBMYSB6F0LWdekvuXEunTcYMVvoXWg0qn6tRm7h
WHU3fZH099Vfzet+/Wl/6e3FP8kO4dNiO46/cIQaV7AO4AV+AlbRzrojnamykpg5
5CxzfX3+jRC+AC1ip3l7mSc8/hsby/rAbV48GIfOmIPtMMZbQRP3b7c/ycslkiFB
0EHOCGKktTbLpE/FfyH6/VsHAgMBAAGjgYMwgYAwDgYDVR0PAQH/BAQDAgWgMBMG
A1UdJQQMMAoGCCsGAQUFBwMCMAwGA1UdEwEB/wQCMAAwHQYDVR0OBBYEFBruFhaF
XlEH4cyHBXpkqawO4nRPMB8GA1UdIwQYMBaAFMTOBZBACJiUSum+WpevJYh8cvMX
MAsGA1UdEQQEMAKCADANBgkqhkiG9w0BAQsFAAOCAQEAMtnmk/FS8RIJk6jaZ49f
faeRq4NhIc+o0pJ4rALmHI3brU2zR/iSDxuwwEseleExTbxqJ0Fey/hzaIWcw2Pj
XeSoU/C5n8xTd/zMm3jxch+bo7P5DFVmH4Z9Nr5jA9G6jQqiYLbrPPmga2vU0CJ1
3Z26Y+zRJ/e5rYlPa4CzNYmbbmlU5Sd6FM5ngMLjjcDI4v+paPWhbOHR+6lN6ZAi
ejRte9WlueiwjNwL1DzT+3NH9jzH/xRJAiHcp1W/zAJ1byMTzdW0VBriMUNh2ivD
xko69KN/ZlEtMI7z389B3msEuvmng5VxSWhT/z1Xm3SpimIaOplvm4u+cL775Gid
Fg==
-----END CERTIFICATE-----`
	msg := "tenant-id-waldot|6961ad59-2c83-4e88-9df9-0fb8a48bd055"
	signature := "5312b49918901257439b162360334a23d6fdc78de8b88deacb4c04cc863c54ca9cdec76c967454f29ccbc3736c82594279edf343ea958208cbc04b64bbdf335ac464ce0e2640746dd40eaee72e69e2da73fc72024aba2cd513c51201249c4fac6dd3f8f016a66741cace30e055494c66248aacb24e1ac03eeb1e23010fa1670e9e26446aa4590aa886057c10191c712fa36cec966e0a7ee1700d5877805697fcfb205e4ea04571a9e108190b76e72af7baa8c7b25240d7e43e98b5691977ba1f94f9765a645d2f635645e9b4a3808b2c99c59348c0e133261c3a76b9d02a09ccc19d3ead2469aa015bdfce8476208590e2e35d5a5c392a05a45e28ec6a68ec59"
	err := crypto.VerifySignature(certificate, msg, signature)
	require.NoErrorf(t, err, "Failed to verify signature")
}
