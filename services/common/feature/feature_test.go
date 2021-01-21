package feature_test

import (
	"cloudservices/common/feature"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

type Features struct {
	URLupgrade     bool `json:"urlUpgrade"`
	HighMemAlert   bool `json:"highMemAlert"`
	RealTimeLogs   bool `json:"realTimeLogs"`
	MultiNodeAware bool `json:"multiNodeAware"`
}

func TestFeature(t *testing.T) {
	features := &feature.Features{}
	features.Add("urlUpgrade", "v1.5.0", "v1.8.0")
	features.Add("highMemAlert", "v1.7.0", "")
	features.Add("realTimeLogs", "v1.11.0", "")
	features.Add("multiNodeAware", "v1.15.0", "")

	versions := map[string]*Features{
		"v1.5.0":  {URLupgrade: true},
		"v1.7.0":  {HighMemAlert: true, URLupgrade: true},
		"v1.12.0": {RealTimeLogs: true, HighMemAlert: true},
		"v1.15.0": {RealTimeLogs: true, HighMemAlert: true, MultiNodeAware: true},
	}
	for ver, fts := range versions {
		outFts := &Features{}
		err := features.Get(ver, outFts)
		require.NoError(t, err)
		if !reflect.DeepEqual(outFts, fts) {
			t.Fatalf("expected %+v, found %+v for version %s", fts, outFts, ver)
		}
	}

}
