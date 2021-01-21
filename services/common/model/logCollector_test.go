package model_test

import (
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"reflect"
	"strings"
	"testing"
)

// TestProject will test Project struct
func TestLogCollector(t *testing.T) {
	now := timeNow(t)
	sourceList := model.LogCollectorSources{
		Edges: []string{"e-1", "e-2"},
		Tags: map[string]string{
			"k": "v",
		},
	}
	code := "This is a test code"
	creditID := ""
	projectId := "project-id"
	logCollectors := []model.LogCollector{
		{
			BaseModel: model.BaseModel{
				ID:        "log-collector-id-1",
				TenantID:  "tenant-id",
				Version:   5,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:         "log-collector-name",
			Type:         model.ProjectCollector,
			State:        model.LogCollectorActive,
			ProjectID:    &projectId,
			Code:         &code,
			Sources:      sourceList,
			CloudCredsID: creditID,
			Destination:  model.AWSCloudWatch,
			CloudwatchDetails: &model.LogCollectorCloudwatch{
				Destination: "1",
				GroupName:   "2",
				StreamName:  "3",
			},
		}, {
			BaseModel: model.BaseModel{
				ID:        "log-collector-id-2",
				TenantID:  "tenant-id",
				Version:   5,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:               "log-collector-name",
			Type:               model.InfraCollector,
			State:              model.LogCollectorStopped,
			Code:               &code,
			Sources:            sourceList,
			CloudCredsID:       creditID,
			Destination:        model.GCPStackDriver,
			StackdriverDetails: &model.LogCollectorStackdriver{},
		},
	}
	logCollectorsString := []string{
		`{"id":"log-collector-id-1","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"log-collector-name","type":"Project","projectId":"project-id","code":"This is a test code","sources":{"edges":["e-1","e-2"],"categories":{"k":"v"}},"state":"ACTIVE","cloudCredsID":"","dest":"CLOUDWATCH","cloudwatchDetails":{"dest":"1","group":"2","stream":"3"}}`,
		`{"id":"log-collector-id-2","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"log-collector-name","type":"Infrastructure","code":"This is a test code","sources":{"edges":["e-1","e-2"],"categories":{"k":"v"}},"state":"STOPPED","cloudCredsID":"","dest":"STACKDRIVER","stackdriverDetails":{}}`,
	}

	for i, lc := range logCollectors {
		collectorData, err := json.Marshal(lc)
		require.NoError(t, err, "failed to marshal log collector")
		if logCollectorsString[i] != string(collectorData) {
			t.Fatal("log collector json string mismatch", string(collectorData), logCollectorsString[i])
		}
		// alternative form: m := make(map[string]interface{})
		m := model.LogCollector{}
		err = json.Unmarshal(collectorData, &m)
		require.NoError(t, err, "failed to unmarshal log collector to map")
		// reflect.DeepEqual fails on equivalent slices here,
		// so use weaker marshal equal
		if !reflect.DeepEqual(&m, &lc) {
			t.Fatal("log collectors map marshal mismatch")
		}
	}
}

func TestLogCollectorValidation(t *testing.T) {
	now := timeNow(t)
	baseModel := model.BaseModel{
		ID:        "log-collector-id",
		TenantID:  "tenant-id",
		Version:   5,
		CreatedAt: now,
		UpdatedAt: now,
	}

	sourceList := model.LogCollectorSources{}
	creditID := ""
	projectId := "project-id"
	cloudwatchDetails := model.LogCollectorCloudwatch{
		Destination: "1",
		GroupName:   "2",
		StreamName:  "3",
	}

	ccAWS := model.CloudCreds{
		Type: "AWS",
	}

	shouldPass := []*model.LogCollector{
		{
			BaseModel:         baseModel,
			Name:              "log-collector-name",
			ProjectID:         &projectId,
			Type:              model.ProjectCollector,
			State:             model.LogCollectorStopped,
			Code:              nil,
			Sources:           sourceList,
			CloudCredsID:      creditID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &cloudwatchDetails,
		},
		{
			BaseModel:         baseModel,
			Name:              "log-collector-name",
			Type:              model.InfraCollector,
			State:             model.LogCollectorStopped,
			Code:              nil,
			Sources:           sourceList,
			CloudCredsID:      creditID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &cloudwatchDetails,
		}, {
			BaseModel:         baseModel,
			Name:              "log-collector-name",
			ProjectID:         &projectId,
			Type:              "",
			State:             "Wrong state",
			Code:              nil,
			Sources:           sourceList,
			CloudCredsID:      creditID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &cloudwatchDetails,
		},
	}
	shouldFail := []*model.LogCollector{
		nil,
		{
			BaseModel:         baseModel,
			Name:              "log-collector-name",
			Type:              model.ProjectCollector,
			State:             model.LogCollectorStopped,
			Code:              nil,
			Sources:           sourceList,
			CloudCredsID:      creditID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &cloudwatchDetails,
		}, {
			BaseModel:         baseModel,
			Name:              "log-collector-name",
			Type:              "should fail",
			ProjectID:         &projectId,
			State:             model.LogCollectorActive,
			Code:              nil,
			Sources:           sourceList,
			CloudCredsID:      creditID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &cloudwatchDetails,
		},
	}

	for _, log := range shouldPass {
		err := model.ValidateLogCollector(log, &ccAWS)
		require.NoError(t, err)
	}

	for i, log := range shouldFail {
		err := model.ValidateLogCollector(log, &ccAWS)
		require.Error(t, err, "Should fall", i)
	}
}

func TestLogCollectorValidationCleanup(t *testing.T) {
	now := timeNow(t)
	baseModel := model.BaseModel{
		ID:        "log-collector-id",
		TenantID:  "tenant-id",
		Version:   5,
		CreatedAt: now,
		UpdatedAt: now,
	}

	sourceList := model.LogCollectorSources{}
	creditID := ""
	projectId := "project-id"
	cloudwatchDetails := model.LogCollectorCloudwatch{
		Destination: "1",
		GroupName:   "2",
		StreamName:  "3",
	}

	kinesisDetails := model.LogCollectorKinesis{}
	stackdriverDetails := model.LogCollectorStackdriver{}

	ccAWS := model.CloudCreds{
		Type: "AWS",
	}

	lc := model.LogCollector{
		BaseModel:    baseModel,
		Name:         "   log-collector-name   ",
		State:        model.LogCollectorStopped,
		Code:         nil,
		Sources:      sourceList,
		CloudCredsID: creditID,

		Type:      model.InfraCollector,
		ProjectID: &projectId,

		Destination:        model.AWSCloudWatch,
		CloudwatchDetails:  &cloudwatchDetails,
		KinesisDetails:     &kinesisDetails,
		StackdriverDetails: &stackdriverDetails,
	}

	err := model.ValidateLogCollector(&lc, &ccAWS)
	require.NoError(t, err)
	if lc.Name != "log-collector-name" {
		t.Fatal("Name")
	}
	if lc.ProjectID != nil {
		t.Fatal("ProjectID")
	}
	if lc.KinesisDetails != nil {
		t.Fatal("KinesisDetails")
	}
	if lc.StackdriverDetails != nil {
		t.Fatal("StackdriverDetails")
	}
}

func TestValidateLogCollectorDestinations(t *testing.T) {
	tooLong := strings.Repeat("a", 513)
	type args struct {
		destType    model.LogCollectorDestination
		cloudwatch  *model.LogCollectorCloudwatch
		kinesis     *model.LogCollectorKinesis
		stackdriver *model.LogCollectorStackdriver
		cc          model.CloudCreds
	}
	tests := []struct {
		name string
		msg  string
		ok   bool
		args args
	}{
		{
			name: "cloudwatch OK",
			ok:   true,
			args: args{
				destType:    model.AWSCloudWatch,
				cloudwatch:  &model.LogCollectorCloudwatch{Destination: "1", GroupName: "2", StreamName: "3"},
				kinesis:     nil,
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "cloudwatch FAIL on empty destination",
			msg:  "Destination is too short",
			ok:   false,
			args: args{
				destType:    model.AWSCloudWatch,
				cloudwatch:  &model.LogCollectorCloudwatch{Destination: "", GroupName: "2", StreamName: "3"},
				kinesis:     nil,
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "cloudwatch FAIL on destination too long",
			msg:  "Destination is too long",
			ok:   false,
			args: args{
				destType:    model.AWSCloudWatch,
				cloudwatch:  &model.LogCollectorCloudwatch{Destination: tooLong, GroupName: "2", StreamName: "3"},
				kinesis:     nil,
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "cloudwatch FAIL on destination regexp failure",
			ok:   false,
			args: args{
				destType:    model.AWSCloudWatch,
				cloudwatch:  &model.LogCollectorCloudwatch{Destination: "1#", GroupName: "2", StreamName: "3"},
				kinesis:     nil,
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "cloudwatch FAIL on empty group",
			ok:   false,
			args: args{
				destType:    model.AWSCloudWatch,
				cloudwatch:  &model.LogCollectorCloudwatch{Destination: "1", GroupName: "", StreamName: "3"},
				kinesis:     nil,
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "cloudwatch FAIL on group too long",
			ok:   false,
			args: args{
				destType:    model.AWSCloudWatch,
				cloudwatch:  &model.LogCollectorCloudwatch{Destination: "1", GroupName: tooLong, StreamName: "3"},
				kinesis:     nil,
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "cloudwatch FAIL on group regexp failure",
			ok:   false,
			args: args{
				destType:    model.AWSCloudWatch,
				cloudwatch:  &model.LogCollectorCloudwatch{Destination: "1", GroupName: "2#", StreamName: "3"},
				kinesis:     nil,
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "cloudwatch FAIL on empty stream",
			ok:   false,
			args: args{
				destType:    model.AWSCloudWatch,
				cloudwatch:  &model.LogCollectorCloudwatch{Destination: "1", GroupName: "2", StreamName: ""},
				kinesis:     nil,
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "cloudwatch FAIL on empty too long",
			ok:   false,
			args: args{
				destType:    model.AWSCloudWatch,
				cloudwatch:  &model.LogCollectorCloudwatch{Destination: "1", GroupName: "2", StreamName: tooLong},
				kinesis:     nil,
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "cloudwatch FAIL on stream regexp failure",
			ok:   false,
			args: args{
				destType:    model.AWSCloudWatch,
				cloudwatch:  &model.LogCollectorCloudwatch{Destination: "1", GroupName: "2", StreamName: "3#"},
				kinesis:     nil,
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "cloudwatch FAIL on CC type mismatch",
			ok:   false,
			args: args{
				destType:    model.AWSCloudWatch,
				cloudwatch:  &model.LogCollectorCloudwatch{Destination: "1", GroupName: "2", StreamName: "3"},
				kinesis:     nil,
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "GCP"},
			},
		}, {
			name: "cloudwatch FAIL on absent details",
			ok:   false,
			args: args{
				destType:    model.AWSCloudWatch,
				cloudwatch:  nil,
				kinesis:     nil,
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "cloudwatch FAIL on details mismatch 1",
			ok:   false,
			args: args{
				destType:    model.AWSCloudWatch,
				cloudwatch:  nil,
				kinesis:     nil,
				stackdriver: &model.LogCollectorStackdriver{},
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "cloudwatch FAIL on details mismatch 2",
			ok:   false,
			args: args{
				destType:    model.AWSCloudWatch,
				cloudwatch:  nil,
				kinesis:     &model.LogCollectorKinesis{Destination: "1", StreamName: "2"},
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "stackdriver OK",
			ok:   true,
			args: args{
				destType:    model.GCPStackDriver,
				cloudwatch:  nil,
				kinesis:     nil,
				stackdriver: &model.LogCollectorStackdriver{},
				cc:          model.CloudCreds{Type: "GCP"},
			},
		}, {
			name: "stackdriver FAIL on CC type mismatch",
			ok:   false,
			args: args{
				destType:    model.GCPStackDriver,
				cloudwatch:  nil,
				kinesis:     nil,
				stackdriver: &model.LogCollectorStackdriver{},
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "stackdriver FAIL on absent details",
			ok:   false,
			args: args{
				destType:    model.GCPStackDriver,
				cloudwatch:  nil,
				kinesis:     nil,
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "GCP"},
			},
		}, {
			name: "kinesis OK",
			ok:   true,
			args: args{
				destType:    model.AWSKinesis,
				cloudwatch:  nil,
				kinesis:     &model.LogCollectorKinesis{Destination: "1", StreamName: "2"},
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "kinesis FAIL on absent destination",
			ok:   false,
			args: args{
				destType:    model.AWSKinesis,
				cloudwatch:  nil,
				kinesis:     &model.LogCollectorKinesis{Destination: "", StreamName: "2"},
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "kinesis FAIL on destination too long",
			ok:   false,
			args: args{
				destType:    model.AWSKinesis,
				cloudwatch:  nil,
				kinesis:     &model.LogCollectorKinesis{Destination: tooLong, StreamName: "2"},
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "kinesis FAIL on destination regexp failure",
			ok:   false,
			args: args{
				destType:    model.AWSKinesis,
				cloudwatch:  nil,
				kinesis:     &model.LogCollectorKinesis{Destination: "1#", StreamName: "2"},
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "kinesis FAIL on absent stream",
			ok:   false,
			args: args{
				destType:    model.AWSKinesis,
				cloudwatch:  nil,
				kinesis:     &model.LogCollectorKinesis{Destination: "1", StreamName: ""},
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "kinesis FAIL on stream too long",
			ok:   false,
			args: args{
				destType:    model.AWSKinesis,
				cloudwatch:  nil,
				kinesis:     &model.LogCollectorKinesis{Destination: "1", StreamName: tooLong},
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "kinesis FAIL on stream regexp failure",
			ok:   false,
			args: args{
				destType:    model.AWSKinesis,
				cloudwatch:  nil,
				kinesis:     &model.LogCollectorKinesis{Destination: "1", StreamName: "2#"},
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "kinesis FAIL on CC type mismatch",
			ok:   false,
			args: args{
				destType:    model.AWSKinesis,
				cloudwatch:  nil,
				kinesis:     &model.LogCollectorKinesis{Destination: "1", StreamName: ""},
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "GCP"},
			},
		}, {
			name: "kinesis OK on full config",
			ok:   true,
			args: args{
				destType:    model.AWSKinesis,
				cloudwatch:  &model.LogCollectorCloudwatch{Destination: "1", GroupName: "2", StreamName: "3"},
				kinesis:     &model.LogCollectorKinesis{Destination: "1", StreamName: "2"},
				stackdriver: &model.LogCollectorStackdriver{},
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "firehose OK",
			ok:   true,
			args: args{
				destType:    model.AWSFirehose,
				cloudwatch:  nil,
				kinesis:     &model.LogCollectorKinesis{Destination: "1", StreamName: "2"},
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "firehose FAIL on absent destination",
			ok:   false,
			args: args{
				destType:    model.AWSFirehose,
				cloudwatch:  nil,
				kinesis:     &model.LogCollectorKinesis{Destination: "", StreamName: "2"},
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "firehose FAIL on destination too long",
			ok:   false,
			args: args{
				destType:    model.AWSFirehose,
				cloudwatch:  nil,
				kinesis:     &model.LogCollectorKinesis{Destination: tooLong, StreamName: "2"},
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "firehose FAIL on destination regexp failure",
			ok:   false,
			args: args{
				destType:    model.AWSFirehose,
				cloudwatch:  nil,
				kinesis:     &model.LogCollectorKinesis{Destination: "1#", StreamName: "2"},
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "firehose FAIL on absent stream",
			ok:   false,
			args: args{
				destType:    model.AWSFirehose,
				cloudwatch:  nil,
				kinesis:     &model.LogCollectorKinesis{Destination: "1", StreamName: ""},
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "firehose FAIL on stream too long",
			ok:   false,
			args: args{
				destType:    model.AWSFirehose,
				cloudwatch:  nil,
				kinesis:     &model.LogCollectorKinesis{Destination: "1", StreamName: tooLong},
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "firehose FAIL on stream regexp failure",
			ok:   false,
			args: args{
				destType:    model.AWSFirehose,
				cloudwatch:  nil,
				kinesis:     &model.LogCollectorKinesis{Destination: "1", StreamName: "2#"},
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}, {
			name: "firehose FAIL on CC type mismatch",
			ok:   false,
			args: args{
				destType:    model.AWSFirehose,
				cloudwatch:  nil,
				kinesis:     &model.LogCollectorKinesis{Destination: "1", StreamName: ""},
				stackdriver: nil,
				cc:          model.CloudCreds{Type: "GCP"},
			},
		}, {
			name: "firehose OK on full config",
			ok:   true,
			args: args{
				destType:    model.AWSFirehose,
				cloudwatch:  &model.LogCollectorCloudwatch{Destination: "1", GroupName: "2", StreamName: "3"},
				kinesis:     &model.LogCollectorKinesis{Destination: "1", StreamName: "2"},
				stackdriver: &model.LogCollectorStackdriver{},
				cc:          model.CloudCreds{Type: "AWS"},
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := timeNow(t)
			lc := model.LogCollector{
				BaseModel: model.BaseModel{
					ID:        "log-collector-id",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:               "log-collector-name",
				Type:               model.InfraCollector,
				State:              model.LogCollectorActive,
				Code:               nil,
				CloudCredsID:       "",
				Destination:        tt.args.destType,
				CloudwatchDetails:  tt.args.cloudwatch,
				KinesisDetails:     tt.args.kinesis,
				StackdriverDetails: tt.args.stackdriver,
			}

			err := model.ValidateLogCollector(&lc, &tt.args.cc)
			if (err == nil) != tt.ok {
				t.Errorf("ValidateLogCollector() error = %v, want error = %v", err, !tt.ok)
			}
			if err != nil && tt.msg != "" {
				msg := err.(*errcode.BadRequestExError).Msg
				if msg != tt.msg {
					t.Errorf("ValidateLogCollector() error = %v, want error = %v", msg, tt.msg)
				}
			}
		})
	}
}

func TestValidateLogCollectorDestinationRegexps(t *testing.T) {
	tests := []string{
		"",
		"    ",
		strings.Repeat("1", 513),
		"#",
		"\\",
		"#",
		"", "{}",
		"][",
		"$1",
		"^",
	}

	for _, tt := range tests {
		now := timeNow(t)
		lc := model.LogCollector{
			BaseModel: model.BaseModel{
				ID:        "log-collector-id",
				TenantID:  "tenant-id",
				Version:   5,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:         "log-collector-name",
			Type:         model.InfraCollector,
			State:        model.LogCollectorActive,
			Code:         nil,
			CloudCredsID: "",
			Destination:  model.AWSCloudWatch,
			CloudwatchDetails: &model.LogCollectorCloudwatch{
				Destination: tt,
				GroupName:   "2",
				StreamName:  "3",
			},
			KinesisDetails:     nil,
			StackdriverDetails: nil,
		}

		if err := model.ValidateLogCollector(&lc, &model.CloudCreds{Type: "AWS"}); err == nil {
			t.Errorf("ValidateLogCollector() should fail on detination that contains= %v", tt)
		}
	}
}
