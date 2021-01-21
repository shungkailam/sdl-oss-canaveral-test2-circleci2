package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"

	"net/http"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func TestService(t *testing.T) {
	t.Parallel()
	t.Log("running TestService test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	ctx, _, _ := makeContext("", []string{})

	// Teardown
	defer func() {
		dbAPI.Close()
	}()

	t.Run("GetServicesW", func(t *testing.T) {
		t.Log("running GetServicesW test")
		inputs := []struct {
			host        string
			serviceType string
			expected    string
		}{
			{"", "", "IoT"},
			{"foo", "", "IoT"},
			{"foo", "bar", "IoT"},
			{"PAAS", "", "PaaS"},
			{"usepaashere", "", "PaaS"},
			{"", "PaAs", "PaaS"},
			{"paas", "iot", "IoT"},
			{"PaaS", "IOT", "IoT"},
			{"iot", "paas", "PaaS"},
			{"graymatter", "graymatter", "GrayMatter"},
			{"gray-matter", "graymatter", "GrayMatter"},
		}

		for _, input := range inputs {
			var w bytes.Buffer
			req := &http.Request{Header: http.Header{
				"X-Forwarded-Host": []string{input.host},
			}}
			*config.Cfg.ServiceType = input.serviceType
			err := dbAPI.GetServicesW(ctx, &w, req)
			require.NoError(t, err)
			resp := model.Service{}
			err = json.NewDecoder(&w).Decode(&resp)
			require.NoError(t, err)
			if resp.ServiceType != input.expected {
				t.Fatalf("for host=%s, config=%s, expect service type to be %s", input.host, input.serviceType, input.expected)
			}
		}
	})
}
