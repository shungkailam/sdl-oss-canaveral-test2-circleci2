package router_test

import (
	"net/url"
	"testing"

	"cloudservices/cloudmgmt/router"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEdgeIDAndURL(t *testing.T) {
	t.Parallel()
	t.Log("running TestParseEdgeIDAndURL test")

	t.Run("Test parse edge id and url", func(t *testing.T) {
		positiveTests := []struct {
			path    string
			edgeID  string
			URL     string
			fullURL string
		}{
			{"/http/<edge id>/path-to-svc", "<edge id>", "http://path-to-svc", "https://test.ntnxsherlock.com/v1.0/proxy/http/<edge id>/path-to-svc"},
			{"/https/<edge id>/path-to-svc", "<edge id>", "https://path-to-svc", "https://test.ntnxsherlock.com/v1.0/proxy/https/<edge id>/path-to-svc"},

			{"/http/1aacfc85-9c03-4026-9460-01db1add5920/kiali.istio-system.svc:20001/kiali/api/namespaces/abc/metrics", "1aacfc85-9c03-4026-9460-01db1add5920", "http://kiali.istio-system.svc:20001/kiali/api/namespaces/abc/metrics", "https://test.ntnxsherlock.com/v1.0/proxy/http/1aacfc85-9c03-4026-9460-01db1add5920/kiali.istio-system.svc:20001/kiali/api/namespaces/abc/metrics"},

			{"/http/<edge id>/path-to-svc", "<edge id>", "http://path-to-svc?k1=v1&k2=v2", "https://test.ntnxsherlock.com/v1.0/proxy/http/<edge id>/path-to-svc?k1=v1&k2=v2"},
		}
		for _, pt := range positiveTests {
			edgeID, u, err := router.ParseEdgeIDAndURL(pt.path, pt.fullURL)
			require.NoError(t, err)
			assert.Equal(t, edgeID, pt.edgeID, "edgeID should be equal")
			assert.Equal(t, u, pt.URL, "URL should be equal")

			_, err = url.Parse(u)
			require.NoError(t, err)

		}
	})
}
