package model

import (
	"bytes"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"time"
)

var (
	EventPathVariableRegex = regexp.MustCompile(`\$\{[a-zA-Z0-9-_]+\}`)
)

// swagger:model Metrics
type Metrics struct {
	Counters map[string]float64 `json:"counters,omitempty"`
	Gauges   map[string]float64 `json:"gauges,omitempty"`
}

// Event is the object model for events
// swagger:model Event
type Event struct {
	Timestamp     time.Time         `json:"timestamp"`
	ID            string            `json:"id"`
	Type          string            `json:"type" validate:"range=1"`
	SourceType    string            `json:"sourceType"`
	Path          string            `json:"path" validate:"range=1"`
	State         string            `json:"state"`
	Version       string            `json:"version"`
	Message       string            `json:"message,omitempty"`
	Severity      string            `json:"severity,omitempty"`
	Properties    map[string]string `json:"properties,omitempty"`
	Metrics       Metrics           `json:"metrics,omitempty"`
	IsEncrypted   bool              `json:"isEncrypted,omitempty"`
	IsInfraEntity bool              `json:"isInfraEntity,omitempty"`
	Audience      string            `json:"audience"`
}

// EventUpsertRequest is the request payload for UpsertEvents
// swagger:model EventUpsertRequest
type EventUpsertRequest struct {
	Events []Event `json:"events"`
}

// EventUpsertParam is used as API parameter for UpsertEvents
// swagger:parameters UpsertEvents UpsertEventsV2
type EventUpsertParam struct {
	// This is events upsert request description
	// in: body
	// required: true
	Body *EventUpsertRequest
}

// EventFilter is the event filter in QueryEvents.
// StartTime is the later time (inclusive) going back to the earlier EndTime (exclusive)
// swagger:model EventFilter
type EventFilter struct {
	// required: true
	Path      string            `json:"path"`
	Keys      map[string]string `json:"keys"`
	Start     int               `json:"start"`
	Size      int               `json:"size"`
	StartTime *time.Time        `json:"startTime"`
	EndTime   *time.Time        `json:"endTime"`
}

// EventFilterV2 is the event filter in v1.0 QueryEvents.
// swagger:model EventFilterV2
type EventFilterV2 struct {
	//
	// Unique path to identify the resource, as in:/serviceDomain:serviceDomainID/project:ProjectID/...
	//
	// required: true
	Path string `json:"path"`
	//
	// Optional search parameters like "keys" : {"type": "ALERT"}
	//
	Keys map[string]string `json:"keys"`
	//
	// Search for events by this later timestamp (inclusive)
	//
	StartTime *time.Time `json:"startTime"`
	//
	// Search for events by this earlier timestamp (exclusive).
	//
	EndTime *time.Time `json:"endTime"`
}

// EventFilterParam is the event filter used as API parameter
// swagger:parameters QueryEvents
type EventFilterParam struct {
	// in: body
	// required: true
	Payload *EventFilter
}

// EventFilterParamV2 is the event filter used as v1.0 API parameter
// swagger:parameters QueryEventsV2
type EventFilterParamV2 struct {
	// in: body
	// required: true
	Payload *EventFilterV2
}

// Ok
// swagger:response EventListResponse
type EventListResponse struct {
	// in: body
	// required: true
	Payload *[]Event
}

// Enable authorization on the endpoints
// swagger:parameters QueryEvents QueryEventsV2 UpsertEvents UpsertEventsV2
// in: header
type eventAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// EventPathComponents is a placeholder for components in a path.
// JSON field names must match the component names in the path
type EventPathComponents struct {
	SvcDomainID    string `json:"serviceDomain"`
	ProjectID      string `json:"project"`
	ApplicationID  string `json:"application"`
	DataPipelineID string `json:"dataPipeline"`
	DataSourceID   string `json:"dataSource"`
	NodeID         string `json:"node"`
	UpgradeVersion string `json:"upgrade"`
	Topic          string `json:"topic"`
	Container      string `json:"container"`
	Function       string `json:"function"`
	Output         string `json:"output"`
	SvcType        string `json:"service"`
	SvcInstanceID  string `json:"instance"`
	SvcBindingID   string `json:"binding"`
	EventID        string `json:"event"`
}

// ExtractEventPathComponents extracts the components from the event path
func ExtractEventPathComponents(path string) *EventPathComponents {
	comps := &EventPathComponents{}
	jsonStr := ConvertEventPathToJSON(path, 0)
	if err := base.ConvertFromJSON([]byte(jsonStr), comps); err != nil {
		return nil
	}
	return comps
}

// ExtractEventPathComponentsN extracts the components from the event path upto
// the level
func ExtractEventPathComponentsN(path string, level int) *EventPathComponents {
	comps := &EventPathComponents{}
	jsonStr := ConvertEventPathToJSON(path, level)
	if err := base.ConvertFromJSON([]byte(jsonStr), comps); err != nil {
		return nil
	}
	return comps
}

// ConvertEventPathToJSON converts the name value pairs of the path e.g project:<projectID> into JSON string
func ConvertEventPathToJSON(path string, level int) string {
	comps := strings.Split(path, "/")
	if len(comps) == 0 {
		return "{}"
	}
	keys := make(map[string]bool, len(comps))
	pairs := make([]string, 0, len(comps))
	levelCount := 0
	for _, comp := range comps {
		parts := strings.SplitN(comp, ":", 2)
		if len(parts) < 2 || keys[parts[0]] {
			continue
		}
		levelCount++
		keys[parts[0]] = true
		pairs = append(pairs, fmt.Sprintf("\"%s\": \"%s\"", parts[0], parts[1]))
		if level > 0 && levelCount >= level {
			break
		}
	}
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}

// GenerateEventQueryPath subsitutes the path variables with the non-zero (non-empty and non nil)
// values in the interface. For the zero values, the value is .* matching any character.
// A true return value means wildcard appears in the path.
func GenerateEventQueryPath(pathTemplate string, values interface{}) (string, bool, error) {
	wildCard := ".*"
	w := bytes.Buffer{}
	valuesMap := map[string]interface{}{}
	err := base.Convert(values, &valuesMap)
	if err != nil {
		return w.String(), false, err
	}
	vars := EventPathVariableRegex.FindAllString(pathTemplate, -1)
	replacements := []string{}
	for _, va := range vars {
		runes := []rune(va)
		vName := string(runes[2 : len(runes)-1])
		replacements = append(replacements, va)
		// If not empty, use the value else use .*
		replacements = append(replacements, fmt.Sprintf("{{with .%s}}{{.}}{{else}}%s{{end}}", vName, wildCard))
	}
	replacer := strings.NewReplacer(replacements...)
	gTemplatePath := replacer.Replace(pathTemplate)
	tm, err := template.New("new").Parse(gTemplatePath)
	if err != nil {
		return w.String(), false, errcode.NewInternalError(err.Error())
	}
	err = tm.Execute(&w, valuesMap)
	if err != nil {
		return w.String(), false, errcode.NewInternalError(err.Error())
	}
	path := w.String()
	if strings.Contains(path, wildCard) {
		return path, true, nil
	}
	return path, false, nil
}

func GenerateEventUpsertPath(pathTemplate string, values interface{}) (string, error) {
	ePath, isWildCardPresent, err := GenerateEventQueryPath(pathTemplate, values)
	if err != nil {
		return ePath, err
	}
	if isWildCardPresent {
		return ePath, errcode.NewBadRequestExError("values", "Some missing path values")
	}
	return ePath, nil
}
