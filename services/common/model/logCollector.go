package model

import (
	"cloudservices/common/errcode"
	"fmt"
	"github.com/golang/glog"
	"regexp"
	"strings"
)

const (
	// States of the log stream
	LogCollectorActive  LogCollectorStatus = "ACTIVE"
	LogCollectorStopped LogCollectorStatus = "STOPPED"
	LogCollectorFailed  LogCollectorStatus = "FAILED"

	// Log collector types
	InfraCollector   LogCollectorType = "Infrastructure"
	ProjectCollector LogCollectorType = "Project"

	// Destinations
	AWSCloudWatch  LogCollectorDestination = "CLOUDWATCH"
	AWSKinesis     LogCollectorDestination = "KINESIS"
	AWSFirehose    LogCollectorDestination = "FIREHOSE"
	GCPStackDriver LogCollectorDestination = "STACKDRIVER"
)

// Status of the log entry - one of ACTIVE, STOPPED, FAILED
// swagger:model LogCollectorStatus
// enum: ACTIVE,STOPPED,FAILED
type LogCollectorStatus string

// Type of the log collector - one of INFRASTRUCTURE, PROJECT
// swagger:model LogCollectorType
// enum: INFRASTRUCTURE,PROJECT
type LogCollectorType string

// Type of the log collector - one of CLOUDWATCH, KINESIS, STACKDRIVER
// swagger:model LogCollectorDestination
// enum: CLOUDWATCH,KINESIS,STACKDRIVER
type LogCollectorDestination string

// Type of the kinesis integration - one of FIREHOSE, STREAM
// swagger:model LogCollectorKinesisType
// enum:  FIREHOSE,STREAM
type LogCollectorKinesisType string

// LogCollectorSources LogCollectorSources - Log collector sources structure
// swagger:model LogCollectorSources
type LogCollectorSources struct {
	//
	// List of edges to enable on
	//
	// required: false
	Edges []string `json:"edges"`

	//
	// List of categories to run on
	//
	// required: false
	Tags map[string]string `json:"categories"`
}

// LogCollectorCloudwatch LogCollectorCloudwatch - Log collector destination config for AWS CloudWatch
// swagger:model LogCollectorCloudwatch
type LogCollectorCloudwatch struct {
	//
	// Destination for log collection (url or region)
	//
	// required: true
	Destination string `json:"dest" db:"dest" validate:"range=1:200"`
	//
	// Stream name
	//
	// required: true
	GroupName string `json:"group" db:"group" validate:"range=1:200"`
	//
	// Stream name
	//
	// required: true
	StreamName string `json:"stream" db:"stream" validate:"range=1:200"`
}

// LogCollectorKinesis LogCollectorKinesis - Log collector destination config for AWS Kinesis
// swagger:model LogCollectorKinesis
type LogCollectorKinesis struct {
	//
	// Destination for log collection (url or region)
	//
	// required: true
	Destination string `json:"dest" db:"dest" validate:"range=1:200"`
	//
	// Stream name
	//
	// required: true
	StreamName string `json:"stream" db:"stream" validate:"range=1:200"`
	//
	// AWS Kinesis type: Firehose or Data Stream
	//
	// required: true
	Type LogCollectorKinesisType `json:"type" db:"type" validate:"range=1:200"`
}

// LogCollectorStackdriver LogCollectorStackdriver - Log collector destination config for GCP StackDriver
// swagger:model LogCollectorStackdriver
type LogCollectorStackdriver struct {
	//
	// A dummy placeholder
	//
	// required: false
	Dummy string `json:"dummy,omitempty" db:"dummy"`
}

// LogCollector is object model for log collection flow
//
// LogCollectors allow to collect logs from multiple components and stream them to the cloud
//
// swagger:model LogCollector
type LogCollector struct {
	// required: true
	BaseModel
	//
	// Name of the LogCollector.
	// Visible by UI only
	//
	// required: true
	Name string `json:"name" db:"name" validate:"range=1:200"`
	//
	// Type of the LogCollector.
	// Infrastructure for infrastructure logs
	// Project for user level logs
	//
	// required: true
	Type LogCollectorType `json:"type" db:"type" validate:"range=1:200"`
	//
	// ID of parent project.
	// This should be required for PROJECT log collectors.
	//
	// required: false
	ProjectID *string `json:"projectId,omitempty" db:"project_id" validate:"range=0:64"`
	//
	// A code to modify logs during collection
	// Log stream modifications (script source code)
	//
	// required: false
	Code *string `json:"code,omitempty" db:"code"`
	//
	// Sources for log collection.
	//
	// required: true
	Sources LogCollectorSources `json:"sources" db:"sources"`
	//
	// State of this entity
	//
	// required: true
	State LogCollectorStatus `json:"state" db:"state"`
	//
	// CloudCreds id.
	// Destination id for the cloud (should match with the CloudDestinationType)
	//
	// required: true
	CloudCredsID string `json:"cloudCredsID" db:"cloud_creds_id" validate:"range=0:36"`
	//
	// Destination of the log collector.
	//
	// required: true
	Destination LogCollectorDestination `json:"dest" db:"dest" validate:"range=1:200"`
	//
	// Credential for the cloud profile.
	// Required when CloudDestinationType == CLOUDWATCH.
	//
	CloudwatchDetails *LogCollectorCloudwatch `json:"cloudwatchDetails,omitempty" db:"aws_cloudwatch"`
	//
	// Credential for the kinesis profile.
	// Required when CloudDestinationType == KINESIS or FIREHOSE.
	//
	KinesisDetails *LogCollectorKinesis `json:"kinesisDetails,omitempty" db:"aws_kinesis"`
	//
	// Credential for the stackdriver profile.
	// Required when CloudDestinationType == STACKDRIVER.
	//
	StackdriverDetails *LogCollectorStackdriver `json:"stackdriverDetails,omitempty" db:"gcp_stackdriver"`
}

// ObjectRequestBaseLogCollector is used as a websocket LogCollector message
// swagger:model ObjectRequestBaseLogCollector
type ObjectRequestBaseLogCollector struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc LogCollector `json:"doc"`
}

// LogCollectorCreateParam is LogCollectorCreate used as API parameter
// swagger:parameters LogCollectorCreate
type LogCollectorCreateParam struct {
	// Describes the log collector creation request
	// in: body
	// required: true
	Body *LogCollector `json:"body"`
}

// LogCollectorUpdateParam is LogCollectorUpdate used as API parameter
// swagger:parameters LogCollectorUpdate
type LogCollectorUpdateParam struct {
	// Describes the log collector update request
	// in: body
	// required: true
	Body *LogCollector `json:"body"`
}

// LogCollectorResponse is a LogCollectorGet response
// swagger:response LogCollectorResponse
type LogCollectorResponse struct {
	// in: body
	// required: true
	Payload *LogCollector
}

// LogCollectorListResponse is a LogCollectorsList response
// swagger:response LogCollectorListResponse
type LogCollectorListResponse struct {
	// in: body
	// required: true
	Payload *LogCollectorListPayload
}

// LogCollectorListPayload is payload for the LogCollectorsList response
type LogCollectorListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of log collectors
	// required: true
	LogCollectorList []LogCollector `json:"result"`
}

// swagger:parameters LogCollectorsList LogCollectorGet LogCollectorCreate LogCollectorUpdate LogCollectorDelete LogCollectorStart LogCollectorStop
// in: header
type logCollectorAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ValidateLogCollector is used to validate newly created or updated LocCollector record
func ValidateLogCollector(model *LogCollector, cc *CloudCreds) (err error) {
	if model == nil {
		err = errcode.NewBadRequestError("LogCollector")
		return
	}

	if cc == nil {
		err = errcode.NewBadRequestError("CloudCreds")
		return
	}

	switch model.Destination {
	case AWSCloudWatch:
		if model.CloudwatchDetails == nil {
			err = errcode.NewBadRequestError("CloudwatchDetails")
			return
		}

		if cc.Type != AWSType {
			err = errcode.NewBadRequestError("CloudCredsID")
			return
		}

		model.CloudwatchDetails.Destination, err = validateAllowedNames(model.CloudwatchDetails.Destination, "Destination")
		if err != nil {
			return
		}

		model.CloudwatchDetails.GroupName, err = validateAllowedNames(model.CloudwatchDetails.GroupName, "GroupName")
		if err != nil {
			return
		}

		model.CloudwatchDetails.StreamName, err = validateAllowedNames(model.CloudwatchDetails.StreamName, "StreamName")
		if err != nil {
			return
		}

		model.KinesisDetails = nil
		model.StackdriverDetails = nil
	case AWSKinesis, AWSFirehose:
		if model.KinesisDetails == nil {
			err = errcode.NewBadRequestError("KinesisDetails")
			return
		}

		if cc.Type != AWSType {
			err = errcode.NewBadRequestError("CloudCredsID")
			return
		}

		model.KinesisDetails.Destination, err = validateAllowedNames(model.KinesisDetails.Destination, "Destination")
		if err != nil {
			return
		}

		model.KinesisDetails.StreamName, err = validateAllowedNames(model.KinesisDetails.StreamName, "StreamName")
		if err != nil {
			return
		}

		model.CloudwatchDetails = nil
		model.StackdriverDetails = nil
	case GCPStackDriver:
		if model.StackdriverDetails == nil {
			return errcode.NewBadRequestError("StackdriverDetails")
		}

		if cc.Type != GCPType {
			return errcode.NewBadRequestError("CloudCredsID")
		}

		model.CloudwatchDetails = nil
		model.KinesisDetails = nil
	default:
		return errcode.NewBadRequestError("Destination")
	}

	if len(model.Type) != 0 {
		switch model.Type {
		case ProjectCollector, InfraCollector:
			break
		default:
			return errcode.NewBadRequestError("Type")
		}
	} else {
		model.Type = ProjectCollector
	}

	if model.ProjectID != nil && model.Type == InfraCollector {
		glog.Warningf("Project is set on infra level log collector name=%s", model.Name)
		model.ProjectID = nil
	} else if model.ProjectID == nil && model.Type == ProjectCollector {
		return errcode.NewBadRequestError("ProjectID")
	}

	if len(model.State) == 0 {
		model.State = LogCollectorStopped
	} else if model.State != LogCollectorActive && model.State != LogCollectorStopped {
		model.State = LogCollectorFailed
	}

	model.Name = strings.TrimSpace(model.Name)

	return nil
}

// Name can be between 1 and 512 characters long. Allowed characters include a-z, A-Z, 0-9, '_' (underscore), '-' (hyphen), '/' (forward slash), and '.'
func validateAllowedNames(val string, valName string) (res string, err error) {
	res = strings.TrimSpace(val)

	if len(res) < 1 {
		err = errcode.NewBadRequestExError(valName, fmt.Sprintf("%s is too short", valName))
		return
	}
	if len(res) > 512 {
		err = errcode.NewBadRequestExError(valName, fmt.Sprintf("%s is too long", valName))
		return
	}

	var match bool
	match, err = regexp.MatchString("[^a-zA-Z0-9_\\-/.]", res)
	if err != nil {
		return
	}
	if match {
		err = errcode.NewBadRequestExError(valName, "illegal characters")
	}
	return
}
