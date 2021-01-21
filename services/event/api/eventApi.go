package api

import (
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/events"
	"cloudservices/common/metrics"
	gapi "cloudservices/event/generated/grpc"
	"context"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/olivere/elastic"
	"github.com/prometheus/client_golang/prometheus"
)

// TODO move to proto
const (
	EventTypeKey    = "type"
	AlertEventType  = "ALERT"
	StatusEventType = "STATUS"
	MetricEventType = "METRIC"

	// Elasticsearch has this max limit on the result window
	ElasticSearchMaxItems = 10000

	staleEventThreshold = time.Hour

	// Event audience
	InfraEventAudience        = "INFRA"
	ProjectEventAudience      = "PROJECT"
	InfraProjectEventAudience = "INFRA_PROJECT"
)

var (
	// Example path /edge:$EDGE_ID/project:$PROJECT_ID/stream:$STREAM_ID/transform:$TRANSFORM_ID/status
	// is converted to /serviceDomain:ID/project:ID/dataPipeline:ID/function:ID/status
	pathCompReplacements = []string{
		// Pairs <Old>, <New>
		"/edge:", "/serviceDomain:",
		"/source:", "/dataSource:",
		"/stream:", "/dataPipeline:",
	}
	replacer = strings.NewReplacer(pathCompReplacements...)

	// DefaultEndTimeWindowSecs is the window for the default endtime
	DefaultEndTimeWindowSecs = int64(30 * 24 * 60 * 60)
)

// TransformPath transforms old paths to new paths
func TransformPath(path string) string {
	if strings.HasPrefix(path, "/edge:") {
		path = replacer.Replace(path)
	}
	return path
}

// Don't store any time series data in ES.
func isTimeSeries(event *gapi.Event) bool {
	return false
}

// SetSearchEndTimeMaybe sets the default search endtime if required
func SetSearchEndTimeMaybe(endTime *timestamp.Timestamp) *timestamp.Timestamp {
	currentEpochSecs := base.RoundedNow().Unix()
	defaultEndTimeSecs := currentEpochSecs - DefaultEndTimeWindowSecs
	if endTime == nil {
		endTime = &timestamp.Timestamp{Seconds: defaultEndTimeSecs}
	} else if defaultEndTimeSecs > endTime.GetSeconds() {
		endTime.Seconds = defaultEndTimeSecs
	}
	return endTime
}

func (server *apiServer) assignEventAudience(ctx context.Context, event *gapi.Event) error {
	policy, err := server.pathPolicyManager.GetPolicy(event.Path)
	if err != nil {
		glog.Warningf(base.PrefixRequestID(ctx, "Error in assigning policy for event path %s. Error: %s"), event.Path, err.Error())
		return err
	}
	if policy.Name != "" {
		event.Audience = policy.Name
	}
	// Override
	if event.Audience == InfraEventAudience || event.Audience == InfraProjectEventAudience {
		event.IsInfraEntity = true
	} else {
		event.IsInfraEntity = false
	}
	return nil
}

func (server *apiServer) UpsertEvents(request *gapi.UpsertEventsRequest, stream gapi.EventService_UpsertEventsServer) error {
	bulk := true
	return server.upsertEventsCommon(request, stream, bulk)
}

func (server *apiServer) upsertEventsCommon(request *gapi.UpsertEventsRequest, stream gapi.EventService_UpsertEventsServer, bulk bool) error {
	metrics.GRPCCallCount.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "method": "UpsertEvents"}).Inc()
	var bulkRequest *elastic.BulkService
	ctx := stream.Context()
	reqID := base.GetRequestID(ctx)
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	tenantID := authContext.TenantID
	projectIDs := auth.GetProjectIDs(authContext)
	isInfraAdmin := auth.IsInfraAdminRole(authContext)
	isEdge := auth.IsEdgeRole(authContext)

	glog.V(3).Infof("Request %s: IsInfraAdmin=%t, Projects=%s",
		reqID, isInfraAdmin, projectIDs)
	errMsgs := []string{}
	events := request.GetEvents()
	index, err := server.CreateSearchIndex(tenantID, nil)
	if err != nil {
		return err
	}
	if bulk {
		bulkRequest = server.search.Bulk()
	}
	for _, event := range events {
		var eventID string
		event.Path = TransformPath(event.Path)
		if isTimeSeries(event) {
			eventID = base.GetUUID()
		} else {
			// Generate the ID so that it can be overwritten
			eventID = *base.GetMD5Hash(event.Path)
		}
		err = server.assignEventAudience(ctx, event)
		if err != nil {
			glog.Warningf(base.PrefixRequestID(ctx, "Error in assigning audience for event path %s. Error: %s"), event.Path, err.Error())
		}
		event.Id = eventID
		// Extract project ID and source type from path
		path := strings.Trim(event.Path, "/")
		comps := strings.Split(path, "/")
		sourceType := []string{}
		svcDomainID := ""
		for i, comp := range comps {
			// comp can be [serviceDomain:svcDomainID] or [project:projectID] or [application:applicationID] etc.
			t := strings.SplitN(comp, ":", 2)
			sourceType = append(sourceType, t[0])
			if i == 0 &&
				sourceType[0] == "serviceDomain" &&
				len(t) == 2 {
				svcDomainID = t[1]
			}
			if i == 1 &&
				sourceType[0] == "serviceDomain" &&
				sourceType[1] == "project" &&
				len(t) == 2 {
				event.ProjectID = t[1]
			}
		}

		if svcDomainID != "" && isEdge {
			authEdgeID := auth.GetEdgeID(authContext)
			if svcDomainID != authEdgeID {
				return errcode.NewPermissionDeniedError("AuthContext service domain ID doesn't match with that in event's path")
			}
		} else if svcDomainID == "" && isEdge {
			return errcode.NewPermissionDeniedError("Service domain ID missing from event path")
		} else if isInfraAdmin {
			if event.ProjectID != "" && !event.IsInfraEntity {
				hit := false
				for _, projectID := range projectIDs {
					if event.ProjectID == projectID {
						hit = true
						break
					}
				}
				if !hit {
					return errcode.NewPermissionDeniedError("No project access")
				}
			}
		} else { // Yet to add project_user.
			return errcode.NewPermissionDeniedError("Current user is neither infra admin nor edge user")
		}
		if event.ProjectID == "" {
			event.ProjectID = "null"
		}
		event.SourceType = strings.Join(sourceType, ".")
		if bulk {
			bodyString, err := server.getSearchDocumentBodyString(index, tenantID, eventID, event)
			if err != nil {
				return err
			}
			indexReq := elastic.NewBulkIndexRequest().Index(index).Type("doc").Id(eventID).Doc(bodyString)
			bulkRequest = bulkRequest.Add(indexReq)
		} else {
			// Insert new document
			docID, err := server.PutSearchDocument(index, tenantID, eventID, event)
			if err == nil {
				event.Id = docID
				err = stream.Send(event)
				if err != nil {
					return err
				}
			} else {
				errMsgs = append(errMsgs, err.Error())
			}
		}
	}
	if bulk {
		bulkRequest = bulkRequest.Refresh("wait_for")
		bulkResponse, err := bulkRequest.Do(context.Background())
		if err != nil {
			return err
		}
		// Indexed returns information about indexed documents
		indexed := bulkResponse.Indexed()
		if len(indexed) != len(events) {
			// should not happen
			return errcode.NewInternalError("events and indexed count mismatch")
		}
		for i, brItem := range indexed {
			errDetails := brItem.Error
			if errDetails == nil {
				event := events[i]
				event.Id = brItem.Id
				err = stream.Send(event)
				if err != nil {
					return err
				}
			} else {
				errMsgs = append(errMsgs, errDetails.Reason)
			}
		}
	}
	if len(errMsgs) > 0 {
		err = errcode.NewInternalError(strings.Join(errMsgs, "\n"))
	}
	return err
}

func (server *apiServer) QueryEvents(request *gapi.QueryEventsRequest, stream gapi.EventService_QueryEventsServer) error {
	metrics.GRPCCallCount.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "method": "QueryEvents"}).Inc()
	ctx := stream.Context()
	reqID := base.GetRequestID(ctx)
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	projectIDs := auth.GetProjectIDs(authContext)
	isInfraAdmin := auth.IsInfraAdminRole(authContext)
	tenantID := authContext.TenantID
	glog.V(3).Infof("Request %s: %+v", reqID, request)
	glog.V(3).Infof("Request %s: IsInfraAdmin=%t, Projects=%s",
		reqID, isInfraAdmin, projectIDs)
	if len(request.GetPath()) == 0 {
		return errcode.NewBadRequestError("request.path")
	}
	if request.GetStart() < 0 {
		return errcode.NewBadRequestError("request.start")
	}
	if request.GetSize() < 0 || request.GetSize() > ElasticSearchMaxItems {
		// Do not put a limit
		request.Size = 0
	}
	endTime := SetSearchEndTimeMaybe(request.GetEndTime())
	keys := request.GetKeys()
	_type := keys[EventTypeKey]
	// ENG-284053 Don't query by type. ALERT(s) and STATUS(s) must override
	// each other. Collapsing happens for indivual queries. Hence we must
	// query all ALERT(s) and STATUS(s) in same query.
	if _type != "" && (_type == StatusEventType || _type == AlertEventType) {
		delete(keys, EventTypeKey)
	}
	now := base.RoundedNow()
	callback := func(ID string, doc interface{}) error {
		event := doc.(gapi.Event)
		// ENG-284053 Filter result by type.
		if _type != "" && event.Type != _type {
			return nil
		}
		if !strings.HasPrefix(event.SourceType, "serviceDomain.upgrade.") {
			// Edge upgrade lives longer (old way of upgrade).
			// Other events are refreshed frequently and can be discarded sooner
			goTime, err := ptypes.Timestamp(event.Timestamp)
			if err != nil {
				glog.Errorf("Error in timestamp conversion for event %s", event.Path)
				return nil
			}
			if now.Sub(goTime) > staleEventThreshold {
				if event.Properties == nil || event.Properties[events.TTL] == "" {
					glog.V(4).Infof(base.PrefixRequestID(ctx, "Discarding old event %+v"), event)
					return nil
				}
				if ttl, err := time.ParseDuration(event.Properties[events.TTL]); err != nil {
					glog.V(4).Infof("Parse TTL to duration error: %v, %v", event.Properties[events.TTL], err)
				} else {
					if time.Now().After(event.Timestamp.AsTime().Add(ttl)) {
						glog.V(4).Infof(base.PrefixRequestID(ctx, "Discarding old event %+v"), event)
						return nil
					} else {
						glog.V(4).Infof(base.PrefixRequestID(ctx, "More than %v old event found %+v"), staleEventThreshold, event)
					}
				}
			}
		}
		event.Id = ID
		err := stream.Send(&event)
		if err != nil {
			glog.Infof("Request %s: Error: %s", reqID, err.Error())
		}
		return err
	}
	searchCriteria := &SearchCriteria{
		TenantID:     tenantID,
		ProjectIDs:   projectIDs,
		IsInfraAdmin: isInfraAdmin,
		PathRegex:    request.GetPath(),
		Keys:         keys,
		Start:        int(request.GetStart()),
		Size:         int(request.GetSize()),
		SortKey:      "timestamp",
		Desc:         true,
		StartTime:    request.GetStartTime(),
		EndTime:      endTime,
		Callback:     callback,
		Model:        gapi.Event{},
	}
	return server.SearchDocuments(searchCriteria)
}

// DeleteEvents is used internally and not exposed in the API
func (server *apiServer) DeleteEvents(ctx context.Context, request *gapi.DeleteEventsRequest) (*gapi.DeleteEventsResponse, error) {
	metrics.GRPCCallCount.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "method": "DeleteEvents"}).Inc()
	reqID := base.GetRequestID(ctx)
	glog.Infof("Request %s: %+v", reqID, request)
	err := server.DeleteSearchDocuments(request.GetIndex())
	if err != nil {
		glog.Infof("Request %s: Error: %s", reqID, err.Error())
		return nil, err
	}
	return &gapi.DeleteEventsResponse{}, nil
}
