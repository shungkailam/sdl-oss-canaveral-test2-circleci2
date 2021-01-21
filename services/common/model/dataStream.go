package model

import (
	"cloudservices/common/errcode"
	"fmt"
	"regexp"
	"strings"
)

const (
	// DestinationDataInterface defines the data stream destination of type `DataInterface`
	DestinationDataInterface = "DataInterface"

	// DestinationCloud defines the data stream destination of type `Cloud`
	DestinationCloud = "Cloud"

	// DestinationEdge defines the data stream destination of type `Edge``
	DestinationEdge = "Edge"

	// IfcOutFieldType is the field type for data interface of kind `OUT`
	IfcOutFieldType = "DataInterfaceOut"

	// IfcKindOut is the constant for data Ifc kind `OUT`
	IfcKindOut = "OUT"
)

// RetentionInfo capture info for retention
type RetentionInfo struct {
	//
	// Retention type can be Time or Size.
	// For Time based retention, limit is in seconds
	// and specifies how long the data should be retained.
	// For Size based retention, limit is in GB
	// and specifies up to the maximum capacity/amount of data
	// to retain.
	//
	// enum: Time,Size
	// required: true
	Type string `json:"type" db:"type"`
	// required: true
	Limit int `json:"limit" db:"limit"`
}

// DataStream is object model for data stream
//
// DataStreams are fundamental building blocks for Karbon Platform Services data pipeline.
//
// swagger:model DataStream
type DataStream struct {
	// required: true
	BaseModel
	//
	// Name of the DataStream.
	// This is the published output (Kafka topic) name.
	//
	// required: true
	Name string `json:"name" db:"name" validate:"range=1:200"`
	//
	// The description of the DataStream
	//
	// required: false
	Description string `json:"description" db:"description" validate:"range=0:200"`
	//
	// Data type of the DataStream.
	// For example, Temperature, Pressure, Image, Multiple, etc.
	//
	// required: true
	DataType string `json:"dataType" db:"data_type" validate:"range=0:20"`
	//
	// The origin of the DataStream.
	// Either 'Data Source' or 'Data Stream'
	//
	// enum: Data Source,Data Stream
	// required: true
	Origin string `json:"origin" db:"origin" validate:"range=0:20"`
	//
	// A list of CategoryInfo used as criteria
	// to filter sources applicable to this DataStream.
	//
	// required: true
	OriginSelectors []CategoryInfo `json:"originSelectors" db:"origin_selectors"`
	//
	// If origin == 'Data Stream', then originId
	// can be used in place of originSelectors
	// to specify the origin data stream ID if the origin data stream is unique.
	//
	OriginID string `json:"originId,omitempty" db:"origin_id" validate:"range=0:36"`
	//
	// Destination of the DataStream.
	// Either Edge or Cloud or DataInterface.
	//
	// enum: Edge,Cloud,DataInterface
	// required: true
	Destination string `json:"destination" db:"destination" validate:"options=Edge:Cloud:DataInterface"`
	//
	// Cloud type, required if destination == Cloud
	//
	// enum: AWS,GCP,Azure
	CloudType string `json:"cloudType,omitempty" db:"cloud_type" validate:"options=AWS:GCP:Azure:"`
	//
	// CloudCreds id.
	// Required if destination == Cloud
	//
	CloudCredsID string `json:"cloudCredsId,omitempty" db:"cloud_creds_id" validate:"range=0:36"`
	//
	// AWS region. Required if cloudType == AWS
	//
	// enum: us-east-2,us-east-1,us-west-1,us-west-2,ap-northeast-1,ap-northeast-2,ap-northeast-3,ap-south-1,ap-southeast-1,ap-southeast-2,ca-central-1,cn-north-1,cn-northwest-1,eu-central-1,eu-west-1,eu-west-2,eu-west-3,sa-east-1
	AWSCloudRegion string `json:"awsCloudRegion,omitempty" db:"aws_cloud_region" validate:"range=0:100"`
	//
	// GCP region. Required if cloudType == GCP
	//
	// enum: northamerica-northeast1,us-central1,us-west1,us-east4,us-east1,southamerica-east1,europe-west1,europe-west2,europe-west3,europe-west4,asia-south1,asia-southeast1,asia-east1,asia-northeast1,australia-southeast1
	GCPCloudRegion string `json:"gcpCloudRegion,omitempty" db:"gcp_cloud_region" validate:"range=0:100"`
	//
	// Type of the DataStream at Edge.
	// Required if destination == Edge
	//
	// enum: Kafka,ElasticSearch,MQTT,DataDriver,None
	EdgeStreamType string `json:"edgeStreamType,omitempty" db:"edge_stream_type" validate:"range=0:20"`
	//
	// Type of the DataStream at AWS Cloud.
	// Required if cloudType == AWS
	//
	// enum: Kinesis,SQS,S3,DynamoDB
	AWSStreamType string `json:"awsStreamType,omitempty" db:"aws_stream_type" validate:"range=0:20"`
	//
	// Type of the DataStream at Azure Cloud.
	// Required if cloudType == Azure
	//
	// enum: Blob
	AZStreamType string `json:"azStreamType,omitempty" db:"az_stream_type" validate:"range=0:20"`
	//
	// Type of the DataStream at GCP Cloud.
	// Required if cloudType == GCP
	//
	// enum: PubSub,CloudDatastore,CloudSQL,CloudStorage
	GCPStreamType string `json:"gcpStreamType,omitempty" db:"gcp_stream_type" validate:"range=0:20"`
	//
	// Current size of the DataStream output in GB.
	//
	// required: true
	Size float64 `json:"size" db:"size"`
	//
	// Whether to turn sampling on.
	// If true, then samplingInterval should be set as well.
	//
	// required: true
	EnableSampling bool `json:"enableSampling" db:"enable_sampling"`
	//
	// Sampling interval in seconds.
	// The sampling interval applies to each mqtt/kafka topic separately.
	//
	SamplingInterval float64 `json:"samplingInterval,omitempty" db:"sampling_interval"`
	//
	// List of transformations (together with their args)
	// to apply to the origin data
	// to produce the destination data.
	// Could be empty if no transformation required.
	// Each entry is the id of the transformation Script to apply to input from origin
	// to produce output to destination.
	//
	// required: true
	TransformationArgsList []TransformationArgs `json:"transformationArgsList" db:"transformation_args_list"`
	//
	// Retention policy for this DataStream.
	// Multiple RetentionInfo are combined using AND semantics.
	// For example, retain data for 1 month AND up to 2 TB of data.
	//
	// required: true
	DataRetention []RetentionInfo `json:"dataRetention" db:"data_retention"`
	//
	// ID of parent project.
	// This should be required, but is not marked as such due to backward compatibility.
	//
	// required: false
	ProjectID string `json:"projectId,omitempty" db:"project_id" validate:"range=0:64"`
	//
	// End point of datastream.
	// User specifies the endpoint.
	// required: false
	EndPoint string `json:"endPoint,omitempty" db:"end_point" validate:"range=0:255"`
	//
	// Endpoint URI
	// Derived from existing fields
	// required false
	EndPointURI string `json:"endPointURI,omitempty"`
	//
	// State of this entity
	//
	// required: false
	State *string `json:"state,omitempty"`

	//
	// Data Ifc endpoints connected to this datastream
	//
	// required: false
	DataIfcEndpoints []DataIfcEndpoint

	// ntnx:ignore
	//
	// required: false
	OutDataIfc *DataSource `json:"outDataIfc,omitempty"`
}

func (ds DataStream) GetEntityState() EntityState {
	if ds.State == nil {
		return DeployEntityState
	}
	return EntityState(*ds.State)
}

//Sets endpoint of datastream. For S3 we append uuid and for other endpoints, just name is appended.
func (ds *DataStream) SetEndPoint() {
	if ds.EndPoint != "" {
		return
	}
	// As of now, endpoint is not mandatory.
	// Once we loosen the restrictions on ds name, this might not set to valid endpoint.
	if ds.Destination == DestinationCloud && ds.CloudType == "AWS" && ds.AWSStreamType == "S3" {
		ds.EndPoint = fmt.Sprintf("datastream-%s", ds.ID)
	} else {
		ds.EndPoint = fmt.Sprintf("datastream-%s", ds.Name)
	}
}

func getNamespaceFromProjectID(id string) string {
	return fmt.Sprintf("project-%s", id)
}

func (ds *DataStream) GenerateEndPointURI() {
	if ds.Destination == DestinationEdge {
		switch ds.EdgeStreamType {
		case "None":
			namespace := getNamespaceFromProjectID(ds.ProjectID)
			ds.EndPointURI = fmt.Sprintf("Server: nats://nats.%s.svc.cluster.local:4222 Topic:%s", namespace, ds.EndPoint) // Ex: nats.project-e90c8b1d-fd05-4888-93b7-e512fccd3c12.svc.cluster.local
		case "ElasticSearch":
			// TODO As of now, no official support
			ds.EndPointURI = "Unavailable"
			//ds.EndPointURI = fmt.Sprintf("http://elasticsearch.default.svc.cluster.local:9200/%s", ds.EndPoint) // Ex: http://elasticsearch.default.svc.cluster.local:9200/<endpoint>
		case "MQTT":
			// TODO default ns is hardcoded
			ds.EndPointURI = fmt.Sprintf("Server: http://mqttserver-svc.default.svc.cluster.local:1883 Topic:%s", ds.EndPoint) // TODO format??
		case "Kafka":
			ds.EndPointURI = "Unavailable" // TODO As of now, no support
		case "DataDriver":
			ds.EndPointURI = "Unavailable" // TODO As of now, no support
		default:
			ds.EndPointURI = "Unavailable"
			//return errcode.NewBadRequestError("DataStream Edgestream type")
		}
	} else if ds.Destination == DestinationCloud {
		switch ds.CloudType {
		case AWSType:
			switch ds.AWSStreamType {
			case "S3":
				// Source http://camel.apache.org/aws-s3.html
				ds.EndPointURI = fmt.Sprintf("%s", ds.EndPoint)
				// Ex: aws-s3://bucket-name?amazonS3Client=#client&...
			case "Kinesis":
				// Source http://camel.apache.org/aws-kinesis.html
				ds.EndPointURI = fmt.Sprintf("%s", ds.EndPoint)
				// Ex: aws-kinesis://mykinesisstream?amazonKinesisClient=#kinesisClient&...
			case "SQS":
				// Source http://camel.apache.org/aws-sqs.html
				ds.EndPointURI = fmt.Sprintf("%s", ds.EndPoint)
				// Ex: aws-sqs://MyQueue?amazonSQSClient=#client&defaultVisibilityTimeout=5000&deleteIfFiltered=false&...
			case "DynamoDB":
				ds.EndPointURI = "Unavailable" // TODO As of now, no support
				//return errcode.NewBadRequestError("DataStream AWSStreamType Error")
			default:
				ds.EndPointURI = "Unavailable"
				//return errcode.NewBadRequestError("DataStream AWSStreamType Error")
			}

		case GCPType: // We will have to access corresponding GCP secret to generate endpointuri.
			switch ds.GCPStreamType {
			case "CloudDatastore":
				ds.EndPointURI = "Unavailable"
			case "PubSub":
				ds.EndPointURI = "Unavailable"
			case "CloudSQL":
				ds.EndPointURI = "Unavailable"
			case "CloudStorage":
				ds.EndPointURI = "Unavailable"
			default:
				ds.EndPointURI = "Unavailable"
				//return errcode.NewBadRequestError("DataStream GCPType Error")
			}

		case AZType:
			ds.EndPointURI = "Unavailable"
			//return errcode.NewBadRequestError("DataStream CloudType Azure Not Supported")

		default:
			ds.EndPointURI = "Unavailable"
			//return errcode.NewBadRequestError("DataStream CloudType Error")
		}

	} else {
		ds.EndPointURI = "Unavailable"
		//return errcode.NewBadRequestError("DataStream Destination Error")
	}
}

// DataStreamCreateParam is DataStream used as API parameter
// swagger:parameters DataStreamCreate
type DataStreamCreateParam struct {
	// This is a datastream creation request description
	// in: body
	// required: true
	Body *DataStream `json:"body"`
}

// DataStreamUpdateParam is DataStream used as API parameter
// swagger:parameters DataStreamUpdate DataStreamUpdateV2
type DataStreamUpdateParam struct {
	// in: body
	// required: true
	Body *DataStream `json:"body"`
}

// Ok
// swagger:response DataStreamGetResponse
type DataStreamGetResponse struct {
	// in: body
	// required: true
	Payload *DataStream
}

// Ok
// swagger:response DataStreamListResponse
type DataStreamListResponse struct {
	// in: body
	// required: true
	Payload *[]DataStream
}

// Ok
// swagger:response DataStreamListResponseV2
type DataStreamListResponseV2 struct {
	// in: body
	// required: true
	Payload *DataStreamListPayload
}

// payload for DataStreamListResponseV2
type DataStreamListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of data streams
	// required: true
	DataStreamList []DataStream `json:"result"`
}

// swagger:parameters DataStreamList DataPipelineList DataStreamGet DataPipelineGet DataStreamCreate DataPipelineCreate DataStreamUpdate DataStreamUpdateV2 DataPipelineUpdate DataStreamDelete DataPipelineDelete ProjectGetDataStreams ProjectGetDataPipelines GetDataPipelineContainers
// in: header
type dataStreamAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ObjectRequestBaseDataStream is used as websocket DataStream message
// swagger:model ObjectRequestBaseDataStream
type ObjectRequestBaseDataStream struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc DataStream `json:"doc"`
}

func (doc DataStream) GetProjectID() string {
	return doc.ProjectID
}

type DataStreamsByID []DataStream

func (a DataStreamsByID) Len() int           { return len(a) }
func (a DataStreamsByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a DataStreamsByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

func ValidateDataStream(model *DataStream) error {
	if model == nil {
		return errcode.NewBadRequestError("DataStream")
	}
	if model.State != nil {
		if len(*model.State) == 0 || *model.State == string(DeployEntityState) {
			// Backward compatibility
			model.State = nil
		} else if *model.State != string(UndeployEntityState) {
			return errcode.NewBadRequestError("State")
		}
	}
	model.Destination = strings.TrimSpace(model.Destination)
	model.CloudType = strings.TrimSpace(model.CloudType)
	model.GCPStreamType = strings.TrimSpace(model.GCPStreamType)
	model.AWSStreamType = strings.TrimSpace(model.AWSStreamType)
	model.EdgeStreamType = strings.TrimSpace(model.EdgeStreamType)
	model.CloudCredsID = strings.TrimSpace(model.CloudCredsID)
	model.GCPCloudRegion = strings.TrimSpace(model.GCPCloudRegion)
	model.AWSCloudRegion = strings.TrimSpace(model.AWSCloudRegion)
	if model.Destination == DestinationCloud {
		if model.CloudType == "GCP" {
			if model.GCPStreamType == "" {
				return errcode.NewBadRequestError("GCPStreamType")
			}
			if model.GCPCloudRegion == "" {
				return errcode.NewBadRequestError("GCPCloudRegion")
			}
		} else if model.CloudType == "AWS" {
			if model.AWSStreamType == "" {
				return errcode.NewBadRequestError("AWSStreamType")
			}
			if model.AWSCloudRegion == "" {
				return errcode.NewBadRequestError("AWSCloudRegion")
			}
		} else if model.CloudType == "Azure" {
			// ok, noop
		} else {
			// unsupported cloud type
			return errcode.NewBadRequestError("CloudType")
		}
		if model.CloudCredsID == "" {
			return errcode.NewBadRequestError("CloudCredsID")
		}
	} else if model.Destination == DestinationEdge {
		if model.EdgeStreamType == "" {
			return errcode.NewBadRequestError("EdgeStreamType")
		}
	} else if model.Destination == DestinationDataInterface {
		// Either OutDataIfc or DataIfcEndpoints are required
		// TODO: Remove OutDataIfc
		if len(model.DataIfcEndpoints) == 0 {
			return errcode.NewBadRequestExError("DataIfcEndpoints", "No DataIfcEndpoints provided")
		}
		if len(model.DataIfcEndpoints) > 1 {
			return errcode.NewBadRequestExError("DataIfcEndpoints", "maximum, only one data Ifc endpoint is allowed")
		}
		for _, endpoint := range model.DataIfcEndpoints {
			if endpoint.Name == "" {
				return errcode.NewBadRequestError("DataIfcEndpoints[i]/Name")
			}
			if endpoint.Value == "" {
				return errcode.NewBadRequestError("DataIfcEndpoints[i]/Value")
			}
			if endpoint.ID == "" {
				return errcode.NewBadRequestError("DataIfcEndpoints[i]/ID")
			}
		}
	} else {
		return errcode.NewBadRequestError("Destination")
	}
	return nil
}

func GCPEndpointSanityCheck(endpoint string) bool {
	/*
		only lowercase letters, numbers,dashes (-), underscores (_)
		Bucket names must start and end with a number or letter.
		Bucket names must contain 3-63 characters
		Bucket names cannot begin with the “goog” prefix.
		Bucket names cannot contain “google”
	*/

	if len(endpoint) >= 3 && len(endpoint) <= 63 {
		if strings.HasPrefix(endpoint, "goog") {
			return false
		}
		if index := strings.Index(endpoint, "google"); index != -1 {
			return false
		}

		re := regexp.MustCompile("^[a-z0-9][a-z0-9_-]+[a-z0-9]$")
		/*
			^[a-z0-9] - should start with either lower case letter or a digit
			[a-z0-9_-]+ should have one or more characters from alphabets, digits, -,_
			[a-z0-9]$ should end with either lower case letter or a digit
		*/
		if valid := re.MatchString(endpoint); !valid {
			return false
		}

		return true
	}

	return false
}
