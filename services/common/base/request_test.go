package base_test

import (
	"cloudservices/common/base"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type InnerReceiver struct {
	InnerValue int `json:"innerValue"`
}

type Receiver struct {
	InnerReceiver
	Scope string   `json:"scope"`
	Type  string   `json:"type"`
	Index int      `json:"index"`
	Tags  []string `json:"tags"`
}

func TestGetHTTPQueryParams(t *testing.T) {
	target := "https://iot.nutanix.com/v1.0/serviceinstances?scope=my-scope&type=kafka&index=5&innerValue=10&tags=category%3Dcat1&tags=category%3Dcat2&essential%3Dtrue"
	req := httptest.NewRequest(http.MethodGet, target, nil)

	receiver := Receiver{}
	err := base.GetHTTPQueryParams(req, &receiver)
	require.NoError(t, err)
	expectedReceiver := Receiver{
		InnerReceiver: InnerReceiver{InnerValue: 10},
		Scope:         "my-scope",
		Type:          "kafka",
		Index:         5,
		Tags: []string{
			"category=cat1", "category=cat2",
		},
	}
	if !reflect.DeepEqual(&receiver, &expectedReceiver) {
		t.Fatalf("mismatch values - expected %+v, found %+v", expectedReceiver, receiver)
	}
	t.Logf("receiver: %+v", receiver)
}
