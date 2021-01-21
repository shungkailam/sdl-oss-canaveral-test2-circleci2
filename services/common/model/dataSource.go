package model

import (
	"cloudservices/common/errcode"
	"strings"
)

// DataSourceIfcPorts - the ports that will be used by this data source
//
type DataSourceIfcPorts struct {
	//
	// Name of the port for the service that runs inside the data source
	//
	// required: true
	Name string `json:"name" db:"name" validate:"range=1:200"`

	//
	// Port number in the container that the data source listens to
	//
	// required: true
	Port int `json:"port" db:"port"`
}

// DataSourceIfcInfo - metadata related to the datasource interface
//
type DataSourceIfcInfo struct {
	//
	// Class of the data source
	// DataInterface or Legacy
	//
	// enum: DATAINTERFACE,LEGACY
	// required: true
	Class string `json:"ifcClass" db:"ifc_class"`

	//
	// Kind of data source
	// IN, OUT, PIPE (bidirectional)
	//
	// enum: IN,OUT,PIPE
	// required: true
	Kind string `json:"ifcKind" db:"ifc_kind"`

	//
	// Primary protocol that this data source implements
	//
	// required: true
	Protocol string `json:"ifcProtocol" db:"ifc_protocol"`

	//
	// The docker img that includes the data source
	//
	// required: true
	Img string `json:"ifcImg" db:"ifc_img"`

	//
	// The project that contains this data source
	//
	ProjectID string `json:"ifcProjectId" db:"ifc_project_id"`

	//
	// Driver from which this data source is derived.
	//
	// required: true
	DriverID string `json:"ifcDriverId" db:"ifc_driver_id"`

	//
	// Any ports that will be opened and used by this datasource
	//
	Ports []DataSourceIfcPorts `json:"ifcPorts" db:"ifc_ports"`
}

type DataSourceFieldInfoCore struct {
	//
	// A unique name within the the data source.
	//
	//
	// required: true
	Name string `json:"name" db:"name" validate:"range=1:200"`
	//
	// Data type for the field.
	// For example, Temperature, Pressure, Custom, and so on.
	// Specify Custom for the entire topic payload. No special extraction is performed.
	// When you specify Custom, Karbon Platform Services might not perform intelligent operations automatically
	// when you specify other fields like Temperature.
	// In the future custom extraction functions
	// for each field might be allowed.
	// DataSource dataType is derived from fieldType of all fields in the data source.
	//
	// required: true
	FieldType string `json:"fieldType" db:"field_type" validate="range=0:20"`
}

// DataSourceFieldInfo - data source field info struct
//
// A field represents specific information within a data source topic payload.
// A topic payload may contain multiple fields.
// The user defines fields extractable from a topic
// by specifying DataSourceFieldInfo for each field of a data source in the user interface.
// The fieldType for a field is used to extract the field value from the topic payload.
//
type DataSourceFieldInfo struct {
	// required: true
	DataSourceFieldInfoCore
	//
	// Topic for the field.
	// The topic specified depends on the protocol in the data source. Specify the mqqtTopic for the MQTT protocol.
	// For the RTSP protocol, the topic is the server endpoint or named protocol stream in the RSTP URL.
	//
	// required: true
	MQTTTopic string `json:"mqttTopic" db:"mqtt_topic" validate="range=0:4096"`
}

// DataSourceFieldInfoV2 - data source field info struct
//
// A field represents specific information within a data source topic payload.
// A topic payload may contain multiple fields.
// The user defines fields extractable from a topic
// by specifying DataSourceFieldInfo for each field of a data source in the user interface.
// The fieldType for a field is used to extract the field value from the topic payload.
//
type DataSourceFieldInfoV2 struct {
	//
	// Name of the field.
	// A unique name within the the data source.
	//
	// required: true
	Name string `json:"name" db:"name" validate:"range=1:200"`
	//
	// topic for the field
	// The topic specified depends on the protocol in the data source. Specify the mqqtTopic for the MQTT protocol.
	// For the RTSP protocol, the topic is the server endpoint or named protocol stream in the RSTP URL.
	//
	// required: true
	Topic string `json:"topic" db:"mqtt_topic" validate="range=0:4096"`
}

func (ds DataSourceFieldInfo) ToV2() DataSourceFieldInfoV2 {
	return DataSourceFieldInfoV2{
		Name:  ds.DataSourceFieldInfoCore.Name,
		Topic: ds.MQTTTopic,
	}
}
func (ds DataSourceFieldInfoV2) FromV2() DataSourceFieldInfo {
	return DataSourceFieldInfo{
		DataSourceFieldInfoCore: DataSourceFieldInfoCore{
			Name:      ds.Name,
			FieldType: "",
		},
		MQTTTopic: ds.Topic,
	}
}

type DataSourceFieldInfoSlice []DataSourceFieldInfo

func (dss DataSourceFieldInfoSlice) ToV2() []DataSourceFieldInfoV2 {
	dssv2 := []DataSourceFieldInfoV2{}
	for _, ds := range dss {
		dssv2 = append(dssv2, ds.ToV2())
	}
	return dssv2
}

type DataSourceFieldInfoV2Slice []DataSourceFieldInfoV2

func (dssv2 DataSourceFieldInfoV2Slice) FromV2() []DataSourceFieldInfo {
	dss := []DataSourceFieldInfo{}
	for _, dsv2 := range dssv2 {
		dss = append(dss, dsv2.FromV2())
	}
	return dss
}

// DataSourceFieldSelector - data source field selector struct
//
// A DataSourceFieldSelector specifies a chosen category value
// and a specified scope (that is, which fields) where this value applies.
// The user annotates each data source field with one or more
// CategoryInfo objects.
// Categories enables the user to specify the data pipeline input.
// The list of categories specified is checked against each data source
// to determine if a field in the DataSource is
// included in the input of the data pipeline.
//
type DataSourceFieldSelector struct {
	// required: true
	CategoryInfo
	//
	// Field name(s) applicable to this CategoryInfo.
	// The special value '\_\_ALL\_\_' indicates that CategoryInfo is applicable to
	// all fields in this data source.
	//
	// required: true
	Scope []string `json:"scope" db:"scope"`
}

type DataSourceCore struct {
	// required: true
	Name string `json:"name" validate:"range=1:200"`
	//
	// Data source type:
	// Sensor or Gateway
	//
	// enum: Sensor,Gateway
	// required: true
	Type string `json:"type" db:"type" validate:"options=Sensor:Gateway"`

	//
	// Sensor connection type:
	// Secure or Unsecure
	//
	// enum: Secure,Unsecure
	// required: true
	Connection string `json:"connection" db:"connection" validate:"options=Secure:Unsecure"`
	//
	// A list of DataSourceFieldSelector users assigned to the data source.
	// Allows a user to use Category selectors to identify the
	// data pipeline source.
	// Selectors with different category IDs are combined with the AND operator,
	// while selectors with the same category ID are combined with the OR operator.
	//
	// required: true
	Selectors []DataSourceFieldSelector `json:"selectors" db:"selectors"`
	// Protocol used by the Sensor.
	// enum: MQTT,RTSP,GIGEVISION,DATAINTERFACE
	// required: true
	Protocol string `json:"protocol" db:"protocol" validate:"options=MQTT:RTSP:GIGEVISION:DATAINTERFACE"`
	// Authentication type used by the sensor.
	// enum: CERTIFICATE,PASSWORD,TOKEN
	// required: true
	AuthType string `json:"authType" db:"auth_type" validate:"options=CERTIFICATE:PASSWORD:TOKEN"`
	//
	// Metadata on interfaces based on datainterface drivers
	//
	IfcInfo *DataSourceIfcInfo `json:"ifcInfo" db:"ifc_info"`
	// Note: Sensor dataType is a property derived
	// from the collection of sensor fields.
	// For example, if only one field exists, then dataType = field.fieldType.
	// Generally, dataType is a union of field types.
}

// Removed "Connection" field as compared to DataSourceCore
type DataSourceCoreV2 struct {
	// required: true
	Name string `json:"name" validate:"range=1:200"`
	//
	// Type of data source.
	// Sensor or Gateway
	//
	// enum: Sensor,Gateway
	// required: true
	Type string `json:"type" db:"type" validate:"options=Sensor:Gateway"`
	//
	// A list of DataSourceFieldSelector users assigned to the data source.
	// Allows a user to use Category selectors to identify the
	// data pipeline source.
	// Selectors with different category IDs are combined with the AND operator,
	// while selectors with the same category ID are combined with the OR operator.
	//
	// required: true
	Selectors []DataSourceFieldSelector `json:"selectors" db:"selectors"`
	// Sensor protocol
	// enum: MQTT,RTSP,GIGEVISION,OTHER,DATAINTERFACE
	// required: true
	Protocol string `json:"protocol" db:"protocol" validate:"options=MQTT:RTSP:GIGEVISION:OTHER:DATAINTERFACE"`
	//
	// Metadata on interfaces based on datainterface drivers
	//
	IfcInfo *DataSourceIfcInfo `json:"ifcInfo" db:"ifc_info"`
	// Type of authentication used by sensor
	// enum: CERTIFICATE,PASSWORD,TOKEN
	// required: true
	AuthType string `json:"authType" db:"auth_type" validate:"options=CERTIFICATE:PASSWORD:TOKEN"`
	// Note: Sensor dataType is a property derived
	// from the collection of sensor fields.
	// For example, if only one field exists, then dataType = field.fieldType.
	// Generally, dataType is a union of field types.
}

// purpose varchar(200), edgeId varchar(36) references EdgeModel(id), name varchar(100), type varchar(20), connection varchar(20), protocol varchar(20), authtype varchar(20), createdAt timestamp, updatedAt timestamp, deletedAt timestamp);

// Data source is an object model for data source
//
// A data source represents a logical IoT Sensor or Gateway.
// Note: This grouping is a construct to store meta information
// for sensors. Defining a data source does not cause
// the topic message to flow into the Karbon Platform Services Service Domain (for example, NATS or Kafka).
// You must create a data pipeline to enable that flow.
// swagger:model DataSource
type DataSource struct {
	// required: true
	EdgeBaseModel
	// required: true
	DataSourceCore
	//
	// User defined fields to extract data from the topic payload.
	//
	// required: true
	Fields []DataSourceFieldInfo `json:"fields" db:"fields"`
	//
	// ntnx:ignore
	//
	// Sensor model
	// As we cannot currently detect sensor capability,
	// we need a list of supported sensorModel values
	// which maps to a predefined sensor payload format.
	SensorModel string `json:"sensorModel" db:"sensor_model" validate:"range=0:200"`
}

// DataSourceV2 is object model for data source
//
// A data source represents a logical IoT Sensor or Gateway.
// Note: This grouping is a construct to store meta information
// for sensors. Defining a data source does not cause
// the topic message to flow into the Karbon Platform Services Service Domain (for example, NATS or Kafka).
// You must create a data pipeline to enable that flow.
// swagger:model DataSourceV2
type DataSourceV2 struct {
	// required: true
	EdgeBaseModel
	// required: true
	DataSourceCoreV2
	//
	// User defined fields to extract data from the topic payload.
	//
	// required: true
	FieldsV2 []DataSourceFieldInfoV2 `json:"fields" db:"fields"`
}

type DataSourceArtifact struct {
	ArtifactBaseModel
	DataSourceID string `json:"dataSourceId"`
}

func (dsc DataSourceCore) ToV2() DataSourceCoreV2 {
	return DataSourceCoreV2{
		Name:      dsc.Name,
		Type:      dsc.Type,
		Selectors: dsc.Selectors,
		Protocol:  dsc.Protocol,
		AuthType:  dsc.AuthType,
		IfcInfo:   dsc.IfcInfo,
	}
}

func (dscV2 DataSourceCoreV2) FromV2() DataSourceCore {
	return DataSourceCore{
		Name:       dscV2.Name,
		Type:       dscV2.Type,
		Selectors:  dscV2.Selectors,
		Protocol:   dscV2.Protocol,
		AuthType:   dscV2.AuthType,
		Connection: "Secure",
		IfcInfo:    dscV2.IfcInfo,
	}
}

func (ds DataSource) ToV2() DataSourceV2 {
	return DataSourceV2{
		EdgeBaseModel:    ds.EdgeBaseModel,
		DataSourceCoreV2: ds.DataSourceCore.ToV2(),
		FieldsV2:         DataSourceFieldInfoSlice(ds.Fields).ToV2(),
	}
}
func (dsv2 DataSourceV2) FromV2() DataSource {
	return DataSource{
		EdgeBaseModel:  dsv2.EdgeBaseModel,
		DataSourceCore: dsv2.DataSourceCoreV2.FromV2(),
		Fields:         DataSourceFieldInfoV2Slice(dsv2.FieldsV2).FromV2(),
	}
}

// DataSourceCreateParam is DataSource used as API parameter
// swagger:parameters DataSourceCreate
type DataSourceCreateParam struct {
	// This is a data source creation request description
	// in: body
	// required: true
	Body *DataSource `json:"body"`
}

// DataSourceCreateParamV2 is DataSource used as API parameter
// swagger:parameters DataSourceCreateV2
type DataSourceCreateParamV2 struct {
	// This is a datasources creation request description
	// in: body
	// required: true
	Body *DataSourceV2 `json:"body"`
}

// DataSourceUpdateParam is DataSource used as API parameter
// swagger:parameters DataSourceUpdate DataSourceUpdateV2
type DataSourceUpdateParam struct {
	// in: body
	// required: true
	Body *DataSource `json:"body"`
}

// DataSourceUpdateParamV2 is DataSource used as API parameter
// swagger:parameters DataSourceUpdateV3
type DataSourceUpdateParamV2 struct {
	// in: body
	// required: true
	Body *DataSourceV2 `json:"body"`
}

// Ok
// swagger:response DataSourceGetResponse
type DataSourceGetResponse struct {
	// in: body
	// required: true
	Payload *DataSource
}

// Ok
// swagger:response DataSourceGetResponseV2
type DataSourceGetResponseV2 struct {
	// in: body
	// required: true
	Payload *DataSourceV2
}

// Ok
// swagger:response DataSourceListResponse
type DataSourceListResponse struct {
	// in: body
	// required: true
	Payload *[]DataSource
}

// Ok
// swagger:response DataSourceListResponseV2
type DataSourceListResponseV2 struct {
	// in: body
	// required: true
	Payload *DataSourceListPayload
}

// payload for DataSourceListResponseV2
type DataSourceListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of data sources
	// required: true
	DataSourceListV2 []DataSourceV2 `json:"result"`
}

// DataSourceCreateArtifactParamV2 is DataSourceArtifact used as API parameter
// swagger:parameters DataSourceCreateArtifactV2
type DataSourceCreateArtifactParamV2 struct {
	// This is a data source artifact creation request description
	// in: body
	// required: true
	Body *DataSourceArtifact `json:"body"`
}

// Ok
// swagger:response DataSourceGetArtifactResponseV2
type DataSourceGetArtifactResponseV2 struct {
	// in: body
	// required: true
	Payload *DataSourceArtifact
}

// swagger:parameters DataSourceList DataSourceListV2 DataSourceGet DataSourceGetV2 DataSourceGetArtifactV2 DataSourceCreate DataSourceCreateV2 DataSourceCreateArtifactV2 DataSourceUpdate DataSourceUpdateV2 DataSourceUpdateV3 DataSourceDelete DataSourceDeleteV2 EdgeGetDatasources EdgeGetDatasourcesV2 ProjectGetDatasources ProjectGetDatasourcesV2
// in: header
type dataSourceAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ObjectRequestBaseDataSource is used as websocket DataSource message
// swagger:model ObjectRequestBaseDataSource
type ObjectRequestBaseDataSource struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc DataSource `json:"doc"`
}

type DataSourcesByID []DataSource

func (a DataSourcesByID) Len() int           { return len(a) }
func (a DataSourcesByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a DataSourcesByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

func (dss DataSourcesByID) ToV2() []DataSourceV2 {
	dssv2 := []DataSourceV2{}
	for _, ds := range dss {
		dssv2 = append(dssv2, ds.ToV2())
	}
	return dssv2
}

type DataSourcesByIDV2 []DataSourceV2

func (a DataSourcesByIDV2) Len() int           { return len(a) }
func (a DataSourcesByIDV2) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a DataSourcesByIDV2) Less(i, j int) bool { return a[i].ID < a[j].ID }

func (dssv2 DataSourcesByIDV2) FromV2() []DataSource {
	dss := []DataSource{}
	for _, dsv2 := range dssv2 {
		dss = append(dss, dsv2.FromV2())
	}
	return dss
}

func ValidateDataSource(model *DataSource) error {
	if model == nil {
		return errcode.NewBadRequestError("DataSource")
	}
	model.Name = strings.TrimSpace(model.Name)
	model.Type = strings.TrimSpace(model.Type)
	model.SensorModel = strings.TrimSpace(model.SensorModel)
	model.Connection = strings.TrimSpace(model.Connection)
	model.Protocol = strings.TrimSpace(model.Protocol)
	model.AuthType = strings.TrimSpace(model.AuthType)
	switch model.Type {
	case "Sensor":
		break
	case "Gateway":
		break
	default:
		return errcode.NewBadRequestError("Type")
	}
	switch model.Connection {
	case "Secure":
		break
	case "Unsecure":
		break
	default:
		return errcode.NewBadRequestError("Connection")
	}
	// for RTSP Protocol, only PASSWORD AuthType is allowed
	// for MQTT Protocol, only CERTIFICATE AuthType is allowed
	// for GIGEVISION Protocol, only TOKEN AuthType  is allowed
	switch model.Protocol {
	case "MQTT":
		if model.AuthType != "CERTIFICATE" {
			return errcode.NewBadRequestError("Protocol/AuthType")
		}
		break
	case "RTSP":
		if model.AuthType != "PASSWORD" {
			return errcode.NewBadRequestError("Protocol/AuthType")
		}
		break
	case "GIGEVISION":
		if model.AuthType != "TOKEN" {
			return errcode.NewBadRequestError("Protocol/AuthType")
		}
		break
	case "DATAINTERFACE":
		if model.IfcInfo == nil {
			return errcode.NewBadRequestError("IfcInfo")
		}
		break
	default:
		return errcode.NewBadRequestError("Protocol")
	}

	if model.IfcInfo != nil {
		if model.Protocol != "DATAINTERFACE" {
			return errcode.NewBadRequestError("Protocol")
		}
		if model.IfcInfo.Class == "" {
			return errcode.NewBadRequestError("IfcInfo/Class")
		}
		if model.IfcInfo.Kind == "" {
			return errcode.NewBadRequestError("IfcInfo/Kind")
		}
		if model.IfcInfo.Protocol == "" {
			return errcode.NewBadRequestError("IfcInfo/Protocol")
		}
		if model.IfcInfo.Img == "" {
			return errcode.NewBadRequestError("IfcInfo/Img")
		}
		if model.IfcInfo.DriverID == "" {
			return errcode.NewBadRequestError("IfcInfo/DriverID")
		}
		for _, p := range model.IfcInfo.Ports {
			if p.Name == "" {
				return errcode.NewBadRequestError("IfcInfo/Ports/Name")
			}
			if p.Port == 0 {
				return errcode.NewBadRequestError("IfcInfo/Ports/Port")
			}
		}
	}

	// switch model.AuthType {
	// case "CERTIFICATE":
	// 	break
	// case "PASSWORD":
	// 	break
	// case "TOKEN":
	// 	break
	// default:
	// 	return errcode.NewBadRequestError("AuthType")
	// }

	return nil
}
