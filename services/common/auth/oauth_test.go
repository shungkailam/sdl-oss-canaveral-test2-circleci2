package auth_test

import (
	"cloudservices/common/auth"
	"testing"
)

func TestOAuth(t *testing.T) {
	clientID := "r5OVZbVdam1haHpfKUQoLmiUEtga"
	clientSecret := "BBfnauBPs44A_T0mvmUN1a9UyPMa"
	idp := "https://idp-dev.nutanix.com"
	redirectURL := "https://my.ntnxsherlock.com/auth/oauth"
	handler := auth.NewOAuthHandler("Iot", clientID, clientSecret, idp, redirectURL)

	/*
		response, err := handler.CreateMyNutanixIOTUser(context.Background(), "nsing@nutanix.com", "Test", "User")
		require.NoError(t, err)
		t.Logf("Response %+v", response)
		if response.TenantID != "r0yfst3bhi1m11hsjg50sr80ojewptx113qq" {
			t.Fatalf("Wrong tenant ID %s returned", response.TenantID)
		}
		if response.StatusCode != 200 {
			t.Fatalf("Wrong status code %d returned", response.StatusCode)
		}
	*/
	auth.RegisterOAuthHandler("Iot", clientID, clientSecret, idp, redirectURL)
	handler = auth.LookupOAuthHandler("Iot")
	if handler == nil {
		t.Fatal("Lookup failed")
	}
	assignURL := handler.GetAssignXIIOTRoleURL()
	if assignURL != "https://demo-my.nutanix.com/api/v1/auth/iot" {
		t.Fatalf("expected assign url https://demo-my.nutanix.com/api/v1/auth/iot, found %s", assignURL)
	}
	handler = auth.LookupOAuthHandler("key2")
	if handler != nil {
		t.Fatal("Lookup smust fail")
	}
	t.Log("OAuth tests passed")
}
