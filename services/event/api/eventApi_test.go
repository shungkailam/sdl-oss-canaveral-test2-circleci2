package api_test

import (
	"cloudservices/common/base"
	"cloudservices/event/api"
	"reflect"
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
)

func TestTransformPath(t *testing.T) {
	t.Parallel()

	// Old to new expected paths
	paths := map[string]string{
		"/edge:12345/project:678/stream:901/status/liveness":                  "/serviceDomain:12345/project:678/dataPipeline:901/status/liveness",
		"/edge:12345/project:678/stream:901/output:connector/status":          "/serviceDomain:12345/project:678/dataPipeline:901/output:connector/status",
		"/edge:12345/project:678/stream:901/$PROTOCOL_TYPE:connector/status":  "/serviceDomain:12345/project:678/dataPipeline:901/$PROTOCOL_TYPE:connector/status",
		"/edge:12345/project:678/application:11/container:$IMAGE_NAME/status": "/serviceDomain:12345/project:678/application:11/container:$IMAGE_NAME/status",
		"/edge:12345/project:678/application:11/status":                       "/serviceDomain:12345/project:678/application:11/status",
		"/edge:12345/project:678/status":                                      "/serviceDomain:12345/project:678/status",
		"/edge:12345/source:10/topic:test/status":                             "/serviceDomain:12345/dataSource:10/topic:test/status",
		"/edge:12345/upgrade:v1.15.0:12345/event":                             "/serviceDomain:12345/upgrade:v1.15.0:12345/event",
		"/edge:12345/upgrade:v1.15.0:12345/progress":                          "/serviceDomain:12345/upgrade:v1.15.0:12345/progress",
	}
	for old, new := range paths {
		output := api.TransformPath(old)
		t.Logf("Output path: %s", output)
		if new != output {
			t.Fatalf("expected %s, found %s", new, output)
		}
	}
}

func TestSetEndTimeMaybe(t *testing.T) {
	t.Parallel()
	var endTime *timestamp.Timestamp
	currentEpochSecs := base.RoundedNow().Unix()
	outEndTime := api.SetSearchEndTimeMaybe(endTime)
	secsDiff := currentEpochSecs - outEndTime.GetSeconds()
	if secsDiff < api.DefaultEndTimeWindowSecs {
		t.Fatalf("End time %d is smaller than minimum window", secsDiff)
	}

	if secsDiff > api.DefaultEndTimeWindowSecs+60 {
		t.Fatalf("End time %d is much larger than minimum window", secsDiff)
	}
	endTime = &timestamp.Timestamp{Seconds: currentEpochSecs}
	outEndTime = api.SetSearchEndTimeMaybe(endTime)
	if !reflect.DeepEqual(outEndTime, endTime) {
		t.Fatalf("End time %d must not change", endTime)
	}
}
