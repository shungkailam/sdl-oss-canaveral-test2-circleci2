package model_test

import (
	"cloudservices/common/base"
	"cloudservices/common/model"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestServiceDomain will test ServiceDomain struct
func TestServiceDomain(t *testing.T) {
	var version float64 = 5
	now := timeNow(t)
	svcDomains := []model.ServiceDomain{
		{
			BaseModel: model.BaseModel{
				ID:        "service-domain-id",
				TenantID:  "tenant-id",
				Version:   version,
				CreatedAt: now,
				UpdatedAt: now,
			},
			ServiceDomainCore: model.ServiceDomainCore{
				Name: "service-domain-name",
			},
			Labels:    nil,
			Connected: true,
			Profile:   nil,
			Env:       nil,
		},
	}

	svcDomainStrings := []string{
		`{"id":"service-domain-id","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"service-domain-name","shortId":null,"virtualIp":null,"description":"","labels":null,"connected":true,"profile":null,"env":null}`,
	}

	svcDomainMaps := []map[string]interface{}{
		{
			"id":          "service-domain-id",
			"tenantId":    "tenant-id",
			"version":     version,
			"name":        "service-domain-name",
			"description": "",
			"connected":   true,
			"shortId":     nil,
			"labels":      nil,
			"virtualIp":   nil,
			"profile":     nil,
			"env":         nil,
			"createdAt":   NOW,
			"updatedAt":   NOW,
		},
	}
	for i, svcDomain := range svcDomains {
		svcDomainData, err := json.Marshal(svcDomain)
		require.NoError(t, err, "failed to marshal service domain")

		t.Logf("service domain json: %s", string(svcDomainData))
		if !reflect.DeepEqual(svcDomainData, []byte(svcDomainStrings[i])) {
			t.Fatalf("service domain json string mismatch: %s\n%s", string(svcDomainData), svcDomainStrings[i])
		}
		m := make(map[string]interface{})
		err = json.Unmarshal(svcDomainData, &m)
		require.NoError(t, err, "failed to unmarshal service domain to map")

		if !reflect.DeepEqual(m, svcDomainMaps[i]) {
			t.Fatalf("expected %+v, but got %+v", svcDomainMaps[i], m)
		}
	}
}

type svcDomainResp struct {
	StatusCode int                  `json:"statusCode"`
	Doc        *model.ServiceDomain `json:"doc"`
}

func TestServiceDomainPtr(t *testing.T) {
	var svcDomain = model.ServiceDomain{
		BaseModel: model.BaseModel{
			ID:       "service-domain-id",
			TenantID: "tenant-id",
			Version:  5,
		},
		ServiceDomainCore: model.ServiceDomainCore{
			Name: "service-domain-name",
		},
	}
	er1 := svcDomainResp{
		StatusCode: 200,
		Doc:        &svcDomain,
	}
	er2 := svcDomainResp{
		StatusCode: 500,
		Doc:        nil,
	}
	ers1, err := json.Marshal(er1)
	require.NoError(t, err, "failed to marshal er1")

	ers2, err := json.Marshal(er2)
	require.NoError(t, err, "failed to marshal er2")

	t.Logf("er1 marshal to %s", string(ers1))
	t.Logf("er2 marshal to %s", string(ers2))
}

func TestServiceDomainValidation(t *testing.T) {
	var svcDomain = model.ServiceDomain{
		BaseModel: model.BaseModel{
			ID:       "service-domain-id",
			TenantID: "tenant-id",
			Version:  5,
		},
		ServiceDomainCore: model.ServiceDomainCore{
			Name: "service-domain-name",
		},
	}
	err := model.ValidateServiceDomain(&svcDomain)
	require.NoError(t, err)
	goodNames := []string{
		"sherlock-test-master-shyan-ming-perng-2018-08-31-10-23-37-82",
		"0123",
		"a.b.c",
		"foo.com",
		"foo-bar.baz",
		"a.b.c.d.e.f",
	}
	for _, name := range goodNames {
		svcDomain.Name = name
		err = model.ValidateServiceDomain(&svcDomain)
		require.NoError(t, err)
		t.Logf("validating svcDomain name %s length: %d", svcDomain.Name, len(svcDomain.Name))

		err = base.ValidateStruct("Name", &svcDomain, "create")
		require.NoError(t, err, "bad update svcDomain name: %s", svcDomain.Name)

		err = base.ValidateStruct("Name", &svcDomain, "update")
		require.NoError(t, err, "bad update svcDomain name: %s", svcDomain.Name)
	}
	badNames := []string{
		"-abcd",
		"abc-",
		"ab c",
		"ab=c",
		"ab,c",
		"abc.",
		"ab.-cd",
		"TestEdge",
		"My Edge 2",
	}
	for _, name := range badNames {
		svcDomain.Name = name
		err = model.ValidateServiceDomain(&svcDomain)
		require.Errorf(t, err, "expect bad name to fail validation: %s", name)
	}
	longNames := []string{
		// too long - max = 60
		"sherlock-test-master-shyan-ming-perng-2018-08-31-10-23-37-823",
	}
	for _, name := range longNames {
		svcDomain.Name = name
		err = base.ValidateStruct("Name", &svcDomain, "create")
		require.Error(t, err, "expect long name to fail create validation: %s", name)

		err = base.ValidateStruct("Name", &svcDomain, "update")
		require.Error(t, err, "expect long name to fail update validation: %s", name)
	}
}
