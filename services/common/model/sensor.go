package model

// Sensor is object model for sensor
//
// For .NEXT Nice we do not have a way to identify a sensor (e.g., via certificate).
// The sensor discovery service will make wildcard (#) subscription to
// mqtt server and report each distinct mqtt topic as a sensor.
//
// swagger:model Sensor
type Sensor struct {
	// required: true
	EdgeBaseModel
	//
	// mqtt topic name that identifies the sensor.
	//
	// required: true
	TopicName string `json:"topicName" db:"topicName" validate:"range=0:4096"`
}

// SensorCreateParam is Sensor used as API parameter
// swagger:parameters SensorCreate SensorCreateV2
type SensorCreateParam struct {
	// This is a sensor creation request description
	// in: body
	// required: true
	Body *Sensor `json:"body"`
}

// SensorUpdateParam is Sensor used as API parameter
// swagger:parameters SensorUpdate SensorUpdateV2 SensorUpdateV3
type SensorUpdateParam struct {
	// in: body
	// required: true
	Body *Sensor `json:"body"`
}

// Ok
// swagger:response SensorGetResponse
type SensorGetResponse struct {
	// in: body
	// required: true
	Payload *Sensor
}

// Ok
// swagger:response SensorListResponse
type SensorListResponse struct {
	// in: body
	// required: true
	Payload *[]Sensor
}

// Ok
// swagger:response SensorListResponseV2
type SensorListResponseV2 struct {
	// in: body
	// required: true
	Payload *SensorListPayload
}

// payload for SensorListResponseV2
type SensorListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of sensors
	// required: true
	SensorList []Sensor `json:"result"`
}

// swagger:parameters SensorList SensorListV2 SensorGet SensorGetV2 SensorCreate SensorCreateV2 SensorUpdate SensorUpdateV2 SensorUpdateV3 SensorDelete SensorDeleteV2 EdgeGetSensors EdgeGetSensorsV2
// in: header
type sensorAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ObjectRequestBaseSensor is used as websocket Sensor message
// swagger:model ObjectRequestBaseSensor
type ObjectRequestBaseSensor struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc Sensor `json:"doc"`
}

// ReportSensorsRequest is used as websocket reportSensors message
// swagger:model ReportSensorsRequest
type ReportSensorsRequest struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Sensors []Sensor `json:"sensors"`
}

type SensorsByID []Sensor

func (a SensorsByID) Len() int           { return len(a) }
func (a SensorsByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SensorsByID) Less(i, j int) bool { return a[i].ID < a[j].ID }
