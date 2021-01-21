package api

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"cloudservices/common/service"
	gapi "cloudservices/event/generated/grpc"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	"google.golang.org/grpc"
)

const (
	// constants used for filtering stale alerts
	AppVersion     = "applicationVersion"
	StreamVersion  = "streamVersion"
	SourceVersion  = "sourceVersion"
	ProjectVersion = "projectVersion"
	Status         = "status"
	Project        = "project"
	DataPipeline   = "dataPipeline"
	Application    = "application"
	DataSource     = "dataSource"
	LogCollector   = "logCollector"

	pushTimeStampLabel = "pushTimeStamp"

	StringTimeFormat = "2006-01-02 15:04:05.999999999 -0700 MST"
)

type eventComponents struct {
	model.EventPathComponents
	ProjectVersion      time.Time
	AppVersion          time.Time
	PushTimeStamp       string
	DataPipelineVersion time.Time
	DataSourceVersion   time.Time
	logCollectorVersion time.Time
}

var (
	svcDomainPushTimeStampRequest *gapi.QueryEventsRequest
)

func init() {
	queryMap["Event_GetAppMetadata"] =
		"select id, updated_at from application_model where id IN ('%s')"
	queryMap["Event_GetStreamMetadata"] =
		"select id, updated_at from data_stream_model where id IN ('%s')"
	queryMap["Event_GetSourceMetadata"] =
		"select id, updated_at from data_source_model where id IN ('%s')"
	queryMap["Event_GetProjectMetadata"] =
		"select id, updated_at from project_model where id IN ('%s')"

	// Create a request to get pushTimeStamp events for all serviceDomains
	svcDomainPushTimeStampRequest = &gapi.QueryEventsRequest{
		Path: "/serviceDomain:.*/clusterPushTimeStamp",
	}
}

// ParseStringTime parses the time.String() timestamp which does not fit into any standard format.
// It looks like 2020-05-29 13:42:57.02607 -0700 PDT m=+0.000850568
// TODO change SD time to standard format
func ParseStringTime(ctx context.Context, strTime string) (time.Time, error) {
	// Try with standard time format
	// "2006-01-02T15:04:05Z07:00"
	tm, err := time.Parse(time.RFC3339, strTime)
	if err == nil {
		return tm, nil
	}
	tokens := strings.SplitN(strTime, " ", 5)
	if len(tokens) >= 5 {
		// Remove m=+0.000850568
		tokens = tokens[0:4]
	}
	modifiedStrTime := strings.Join(tokens, " ")
	tm, err = time.Parse(StringTimeFormat, modifiedStrTime)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in parsing string time %s. Error: %s"), strTime, err.Error())
		err = errcode.NewInternalError(err.Error())
	}
	return tm, err
}

// GetServiceDomainPushTimeStamps returns Map of <svcDomains, latestPushTimeStamp> to filter stale events.
func (dbAPI *dbObjectModelAPI) GetServiceDomainPushTimeStamps(ctx context.Context) (map[string]time.Time, error) {
	svcDomainPushTimeStamps := map[string]time.Time{}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewEventServiceClient(conn)
		var gEvent *gapi.Event

		authContext, err := base.GetAuthContext(ctx)
		if err != nil {
			glog.V(4).Infof(base.PrefixRequestID(ctx, "Error in getting auth context: %v"), err)
			return err
		}
		infraCtx := base.GetAdminContextWithTenantID(ctx, authContext.TenantID)
		stream, err := client.QueryEvents(infraCtx, svcDomainPushTimeStampRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(infraCtx, "Error in QueryEvents. Error: %s"), err.Error())
			return err
		}
		for {
			gEvent, err = stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				err = errcode.NewInternalError(err.Error())
				glog.Errorf(base.PrefixRequestID(ctx, "Error in QueryEvents. Error: %s"), err.Error())
				return err
			}
			event := model.Event{}
			err = base.Convert(gEvent, &event)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in QueryEvents. Error: %s"), err.Error())
				return err
			}
			comps := model.ExtractEventPathComponents(event.Path)
			if comps == nil {
				continue
			}
			if event.Properties[pushTimeStampLabel] == "" {
				continue
			}
			tm, err := ParseStringTime(ctx, event.Properties[pushTimeStampLabel])
			if err != nil {
				continue
			}
			svcDomainPushTimeStamps[comps.SvcDomainID] = tm
		}
		return nil
	}
	err := service.CallClient(ctx, service.EventService, handler)
	return svcDomainPushTimeStamps, err
}

func (dbAPI *dbObjectModelAPI) QueryEvents(ctx context.Context, filter model.EventFilter) ([]model.Event, error) {
	events := []model.Event{}
	svcDomainPushTimeStampEvents := []model.Event{}
	request := &gapi.QueryEventsRequest{}
	err := base.Convert(&filter, request)
	if err != nil {
		return nil, err
	}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewEventServiceClient(conn)
		var gEvent *gapi.Event

		authContext, err := base.GetAuthContext(ctx)
		if err != nil {
			glog.V(4).Infof("Error in getting auth context: %v", err)
			return err
		}
		// Fix for ENG-280001
		infraCtx := base.GetAdminContextWithTenantID(ctx, authContext.TenantID)
		stream, err := client.QueryEvents(infraCtx, svcDomainPushTimeStampRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(infraCtx, "Error in QueryEvents. Error: %s"), err.Error())
			return err
		}
		for {
			gEvent, err = stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				err = errcode.NewInternalError(err.Error())
				glog.Errorf(base.PrefixRequestID(ctx, "Error in QueryEvents. Error: %s"), err.Error())
				return err
			}
			event := model.Event{}
			err = base.Convert(gEvent, &event)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in QueryEvents. Error: %s"), err.Error())
				return err
			}
			svcDomainPushTimeStampEvents = append(svcDomainPushTimeStampEvents, event)
		}

		stream, err = client.QueryEvents(ctx, request)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in QueryEvents. Error: %s"), err.Error())
			return err
		}

		for {
			gEvent, err = stream.Recv()
			if err == io.EOF {
				events = FilterEventsForLatestVersions(ctx, events, svcDomainPushTimeStampEvents, dbAPI.addObjectMetadataToMap, dbAPI.SelectAllApplications)
				return nil
			}
			if err != nil {
				err = errcode.NewInternalError(err.Error())
				glog.Errorf(base.PrefixRequestID(ctx, "Error in QueryEvents. Error: %s"), err.Error())
				return err
			}
			event := model.Event{}
			err = base.Convert(gEvent, &event)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in QueryEvents. Error: %s"), err.Error())
				return err
			}
			events = append(events, event)
		}
	}
	err = service.CallClient(ctx, service.EventService, handler)
	return events, err
}

// FilterEventsForLatestVersions filters stale events based on timestamp, push timestamp for apps
func FilterEventsForLatestVersions(
	ctx context.Context,
	events []model.Event,
	svcDomainPushTimeStampEvents []model.Event,
	addDBObjectMetadata func(ctx context.Context, objIDs []string, queryX string, mapUUIDVer map[string]time.Time),
	selectApps func(context context.Context) ([]model.Application, error)) []model.Event {
	var res []model.Event
	mapUUIDVer := make(map[string]time.Time) // Map of objects [uuid:updatedTSfromDB]
	mapAppIDs := make(map[string]bool)
	mapStreamIDs := make(map[string]bool)
	mapSourceIDs := make(map[string]bool) // map for datasources
	mapProjIDs := make(map[string]bool)

	mapAppIDSvcDomainIDs := make(map[string](map[string]bool)) // map of appID:[](EdgeID, true)
	// Map of <svcDomains, latestPushTimeStamp> to filter stale events.
	mapSvcDomainPushTimstamp := GetSvcDomainPushTimestampsMap(svcDomainPushTimeStampEvents)

	// Fix for ENG-280299
	apps, err := selectApps(ctx)
	if err == nil { // in case db is not reachable, we won't show any app alerts. RFC is it acceptable?
		for _, app := range apps {
			temp := make(map[string]bool)
			for _, edgeID := range app.EdgeIDs {
				temp[edgeID] = true
			}
			mapAppIDSvcDomainIDs[app.ID] = temp
		}
	}

	// contains events that are not added to return list after first iteration over events
	var filterEvents []model.Event
	for _, event := range events {
		if event.Properties == nil ||
			(event.Properties[AppVersion] == "" &&
				event.Properties[SourceVersion] == "" &&
				event.Properties[StreamVersion] == "" &&
				event.Properties[ProjectVersion] == "" &&
				event.Properties[pushTimeStampLabel] == "") {
			// Old schema alert
			res = append(res, event)
			continue
		}

		eventComponents := extractEventComponents(ctx, event)
		if eventComponents == nil {
			// If new alert is added which doesn't confine to existing path,
			// then we hit this error
			glog.V(4).Infof(base.PrefixRequestID(ctx, "Unhandled event: %v"), event)
			res = append(res, event)
			continue
		}

		if eventComponents.ApplicationID != "" {
			mapAppIDs[eventComponents.ApplicationID] = true
			if mapAppIDSvcDomainIDs[eventComponents.ApplicationID] == nil || // DB should have the app
				mapAppIDSvcDomainIDs[eventComponents.ApplicationID][eventComponents.SvcDomainID] == false { // And app should be running on the svc domain
				continue
			}
		} else if eventComponents.DataPipelineID != "" {
			mapStreamIDs[eventComponents.DataPipelineID] = true
		} else if eventComponents.DataSourceID != "" {
			mapSourceIDs[eventComponents.DataSourceID] = true
		} else if eventComponents.ProjectID != "" {
			mapProjIDs[eventComponents.ProjectID] = true
		} else {
			glog.V(4).Infof(base.PrefixRequestID(ctx, "Unhandled event path: %s"), event.Path)
			res = append(res, event)
			continue
		}
		// Filter based on push time stamp if present.
		if eventComponents.PushTimeStamp != "" {
			pushTimeStamp := mapSvcDomainPushTimstamp[eventComponents.SvcDomainID]
			if pushTimeStamp != "" && strings.Compare(eventComponents.PushTimeStamp, pushTimeStamp) >= 0 {
				res = append(res, event) // Event is current.
			} else if pushTimeStamp == "" {
				// shouldn't be the case
				glog.V(4).Infof(base.PrefixRequestID(ctx, "Event has push timestamp but not service domain: Event Components: %v, MapSvcIDTimestamp: %v"),
					eventComponents, mapSvcDomainPushTimstamp)
				res = append(res, event)
			} else {
				continue // Event is stale
			}
		} else {
			// We have to filter event
			filterEvents = append(filterEvents, event)
		}
	}

	// Get latest versions of apps/streams/sources/projects
	mapToSlice := func(m map[string]bool) []string {
		ids := make([]string, 0, len(m))
		for id := range m {
			ids = append(ids, id)
		}
		return ids
	}

	appIDs := mapToSlice(mapAppIDs)
	addDBObjectMetadata(ctx, appIDs, "Event_GetAppMetadata", mapUUIDVer)

	streamIDs := mapToSlice(mapStreamIDs)
	addDBObjectMetadata(ctx, streamIDs, "Event_GetStreamMetadata", mapUUIDVer)

	projIDs := mapToSlice(mapProjIDs)
	addDBObjectMetadata(ctx, projIDs, "Event_GetProjectMetadata", mapUUIDVer)

	sourceIDs := mapToSlice(mapSourceIDs)
	addDBObjectMetadata(ctx, sourceIDs, "Event_GetSourceMetadata", mapUUIDVer)

	timesEqual := func(t1, t2 time.Time) bool {
		// For backward compatibility
		// if t1, t2 are apart by less than 2 micro seconds, we consider them equal

		// Earlier postgres converted nanoseconds to micro seconds and in the process sometimes values were rounded up.
		// Once all existing apps are updated at least once, we can remove this
		return t1.Sub(t2) < 2*time.Microsecond && t2.Sub(t1) < 2*time.Microsecond
		// return t1.Equal(t2)
	}
	// filter stale events
	for _, event := range filterEvents {
		eventComponents := extractEventComponents(ctx, event)
		// eventComponents will never be nil
		if eventComponents.ApplicationID != "" {
			if timesEqual(mapUUIDVer[eventComponents.ApplicationID], eventComponents.AppVersion) {
				res = append(res, event)
				continue
			}
			//else stale alert
		} else if eventComponents.DataPipelineID != "" {
			if timesEqual(mapUUIDVer[eventComponents.DataPipelineID], eventComponents.DataPipelineVersion) {
				res = append(res, event)
				continue
			}
			//else stale alert
		} else if eventComponents.DataSourceID != "" {
			if timesEqual(mapUUIDVer[eventComponents.DataSourceID], eventComponents.DataSourceVersion) {
				res = append(res, event)
				continue
			}
			//else stale alert
		} else if eventComponents.ProjectID != "" {
			/* This has to be last else if condition as other alert schemas
			will also have non empty ProjectID */
			if timesEqual(mapUUIDVer[eventComponents.ProjectID], eventComponents.ProjectVersion) {
				res = append(res, event)
				continue
			}
			//else stale alert
		} // else shouldn't come here
	}
	return res
}

func GetSvcDomainPushTimestampsMap(pushTimeStampEvents []model.Event) map[string]string {
	if pushTimeStampEvents == nil || len(pushTimeStampEvents) == 0 {
		return nil
	}
	mapSvcDomainPushTimeStamp := make(map[string]string)

	for _, ev := range pushTimeStampEvents {
		comps := model.ExtractEventPathComponents(ev.Path)
		if comps == nil {
			continue
		}
		mapSvcDomainPushTimeStamp[comps.SvcDomainID] = ev.Properties[pushTimeStampLabel]
	}

	return mapSvcDomainPushTimeStamp
}

// Modifies incoming map.
func (dbAPI *dbObjectModelAPI) addObjectMetadataToMap(
	ctx context.Context, ids []string, queryStr string, mapUUID map[string]time.Time,
) {
	if len(ids) == 0 {
		return
	}

	query := fmt.Sprintf(queryMap[queryStr], strings.Join(ids, "', '"))
	metadata, err := dbAPI.getEntityVersionMetadataList(ctx, query)
	if err != nil {
		glog.V(4).Infof(base.PrefixRequestID(ctx, "Error in fetching metadata: %v for query string: %s, IDs: %v"), err, queryStr, ids)
	} else {
		for _, resourceIDVer := range metadata {
			mapUUID[resourceIDVer.ID] = resourceIDVer.UpdatedAt
		}
	}
	return
}

func extractEventComponents(ctx context.Context, ev model.Event) *eventComponents {
	if ev.Properties == nil { // Old schema
		return nil
	}
	comps := model.ExtractEventPathComponents(ev.Path)
	if comps == nil {
		return nil
	}
	event := &eventComponents{
		EventPathComponents: *comps,
	}

	// SHLK-433 Set push time stamp for events which are periodically refreshed by edge-mgmt.
	// This included application and service events.
	if pushTimeStamp, ok := ev.Properties[pushTimeStampLabel]; ok {
		event.PushTimeStamp = pushTimeStamp
	}

	// an event must /serviceDomain:svcDomainID/...
	if event.SvcDomainID == "" {
		return nil
	}

	// Project/DataSource/LogCollector
	if event.ProjectID != "" {
		if ev.Properties[ProjectVersion] != "" {
			if dt, err := ParseStringTime(ctx, ev.Properties[ProjectVersion]); err == nil {
				event.ProjectVersion = dt
			}
		}
	} else if event.DataSourceID != "" {
		if dt, err := ParseStringTime(ctx, ev.Properties[SourceVersion]); err == nil {
			event.DataSourceVersion = dt
		}
		return event
	} else {
		return nil
	}

	if event.ApplicationID != "" {
		if dt, err := ParseStringTime(ctx, ev.Properties[AppVersion]); err == nil {
			event.AppVersion = dt
		}
		return event
	} else if event.DataPipelineID != "" {
		if dt, err := ParseStringTime(ctx, ev.Properties[StreamVersion]); err == nil {
			event.DataPipelineVersion = dt
		}
		return event
	} else if event.ProjectID != "" { // Project event
		return event
	} else { // Not any known schema
		return nil
	}
}

func (dbAPI *dbObjectModelAPI) QueryEventsW(context context.Context, w io.Writer, r *http.Request) error {
	reader := io.Reader(r.Body)
	// EventFilter is still compatible when EventFilterV2 is passed.
	doc := model.EventFilter{}
	err := base.Decode(&reader, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error decoding into event filter. Error: %s"), err.Error())
		return err
	}
	pageQueryParam := model.GetEntitiesQueryParam(r).PageQueryParam
	// Support backward compatibilty of the APIs across versions.
	// EventFilterV2 is passed, the query params can be optionally set.
	// The previous version using EventFilter is not made public.
	if pageQueryParam.PageIndex != 0 {
		doc.Start = pageQueryParam.PageIndex
	}
	if pageQueryParam.PageSize != base.MaxRowsLimit {
		doc.Size = pageQueryParam.PageSize
	}
	events, err := dbAPI.QueryEvents(context, doc)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, events)
}

func (dbAPI *dbObjectModelAPI) UpsertEvents(ctx context.Context, docs model.EventUpsertRequest, callback func(context.Context, interface{}) error) ([]model.Event, error) {
	resp := []model.Event{}
	gEvents := []*gapi.Event{}
	for _, event := range docs.Events {
		gEvent := gapi.Event{}
		err := base.Convert(event, &gEvent)
		if err != nil {
			return resp, err
		}
		// Alerts are critical by default.
		if gEvent.Type == "ALERT" && gEvent.Severity == "" {
			gEvent.Severity = "CRITICAL"
		}
		gEvents = append(gEvents, &gEvent)
	}
	request := &gapi.UpsertEventsRequest{Events: gEvents}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewEventServiceClient(conn)
		var gEvent *gapi.Event
		stream, err := client.UpsertEvents(ctx, request)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in UpsertEvents. Error: %s"), err.Error())
			return err
		}
		for {
			gEvent, err = stream.Recv()
			if err == io.EOF {
				glog.Info(base.PrefixRequestID(ctx, "QueryEvents completed"))
				return nil
			}
			if err != nil {
				return err
			}
			event := model.Event{}
			err = base.Convert(gEvent, &event)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Data conversion error in UpsertEvents. Error: %s"), err.Error())
				return err
			}
			resp = append(resp, event)
		}
	}
	err := service.CallClient(ctx, service.EventService, handler)
	if callback != nil {
		go callback(ctx, resp)
	}
	return resp, err
}

func (dbAPI *dbObjectModelAPI) UpsertEventsW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	doc := model.EventUpsertRequest{}
	err := base.Decode(&r, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error decoding into event request in UpsertEventsW. Error: %s"), err.Error())
		return err
	}
	resp, err := dbAPI.UpsertEvents(context, doc, callback)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, resp)
}
