package api

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

const entityTypeEdgeInfo = "edgeinfo"

func init() {
	orderByHelper.Setup(entityTypeEdgeInfo, []string{"id", "version", "created_at", "updated_at", "edge_id", "num_cpu", "total_memory_kb", "total_storage_kb", "gpu_info", "cpu_usage", "memory_free_kb", "storage_free_kb", "gpu_usage", "edge_version", "edge_build_num", "kube_version", "os_version"})
}

func (dbAPI *dbObjectModelAPI) initEdgeInfo(ctx context.Context, tx *base.WrappedTx, edgeID string, now time.Time) error {
	return dbAPI.initEdgeDeviceInfo(ctx, tx, edgeID, now)
}

// SelectAllEdges select all edgeInfo for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllEdgesInfo(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.EdgeUsageInfo, error) {
	deviceInfos, err := dbAPI.SelectAllEdgeDevicesInfo(context, entitiesQueryParam)
	if err != nil {
		return []model.EdgeUsageInfo{}, err
	}
	usageInfos := make([]model.EdgeUsageInfo, 0, len(deviceInfos))
	for _, deviceInfo := range deviceInfos {
		usageInfos = append(usageInfos, deviceInfo.ToEdgeUsageInfo())
	}
	return usageInfos, nil
}

// SelectAllEdgesW select all edgesInfo for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgesInfoW(context context.Context, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParamV1(req)
	usageInfos, err := dbAPI.SelectAllEdgesInfo(context, entitiesQueryParam)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, usageInfos)
}

// getEdgesInfoWV2 select all edgesInfo for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) getEdgesInfoWV2(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	queryParam := model.GetEntitiesQueryParam(req)
	deviceInfos, totalCount, err := dbAPI.getEdgeDevicesInfoV2(context, projectID, "", req)
	if err != nil {
		return err
	}
	edgesUsageInfos := make([]model.EdgeUsageInfo, 0, len(deviceInfos))
	for _, deviceInfo := range deviceInfos {
		edgesUsageInfos = append(edgesUsageInfos, deviceInfo.ToEdgeUsageInfo())
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeEdgeInfo})
	r := model.EdgeInfoListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		EdgeUsageInfoList:         edgesUsageInfos,
	}
	return json.NewEncoder(w).Encode(r)
}

// SelectAllEdgesWV2 select all edgesInfo for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgesInfoWV2(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getEdgesInfoWV2(context, "", w, req)
}

// SelectAllEdgesForProject select all edge usage info for the given tenant + project
func (dbAPI *dbObjectModelAPI) SelectAllEdgesInfoForProject(context context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.EdgeUsageInfo, error) {
	deviceInfos, err := dbAPI.SelectAllEdgeDevicesInfoForProject(context, projectID, entitiesQueryParam)
	if err != nil {
		return []model.EdgeUsageInfo{}, err
	}
	usageInfos := make([]model.EdgeUsageInfo, 0, len(deviceInfos))
	for _, deviceInfo := range deviceInfos {
		usageInfos = append(usageInfos, deviceInfo.ToEdgeUsageInfo())
	}
	return usageInfos, nil
}

// SelectAllEdgesInfoForProjectW select all edges info for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgesInfoForProjectW(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParamV1(req)
	usageInfos, err := dbAPI.SelectAllEdgesInfoForProject(context, projectID, entitiesQueryParam)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, usageInfos)
}

// SelectAllEdgesInfoForProjectWV2 select all edges info for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgesInfoForProjectWV2(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getEdgesInfoWV2(context, projectID, w, req)
}

// GetEdge get a edgeInfo object in the DB
func (dbAPI *dbObjectModelAPI) GetEdgeInfo(context context.Context, id string) (model.EdgeUsageInfo, error) {
	usageInfo := model.EdgeUsageInfo{}
	deviceInfo, err := dbAPI.GetEdgeDeviceInfo(context, id)
	if err != nil {
		return usageInfo, err
	}
	return deviceInfo.ToEdgeUsageInfo(), err
}

// GetEdgeW get a edgeInfo object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetEdgeInfoW(context context.Context, id string, w io.Writer, req *http.Request) error {
	usageInfo, err := dbAPI.GetEdgeInfo(context, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, usageInfo)
}

// CreateEdge creates an edgeInfo object in the DB
// Note: POST /edgesinfo is not there, this method is not used. Only PUT /edges/{id}/info is exposed
// EdgeInfo is initially created when edge is created, see CreateEdge in edgeApi.go
func (dbAPI *dbObjectModelAPI) CreateEdgeInfo(context context.Context, i interface{} /* *model.EdgeUsageInfo */, callback func(context.Context, interface{}) error) (interface{}, error) {
	p, ok := i.(*model.EdgeUsageInfo)
	if !ok {
		return model.CreateDocumentResponse{}, errcode.NewInternalError("CreateEdgeInfo: type error")
	}
	doc := *p
	edgeDeviceInfo := doc.ToEdgeDeviceInfo()
	return dbAPI.CreateEdgeDeviceInfo(context, &edgeDeviceInfo, callback)
}

// CreateEdgeW creates an edgeInfo object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateEdgeInfoW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateEdgeInfo, &model.EdgeUsageInfo{}, w, r, callback)
}

// CreateEdgeWV2 creates an edgeInfo object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateEdgeInfoWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateEdgeInfo), &model.EdgeUsageInfo{}, w, r, callback)
}

// UpdateEdge update an edgeInfo object in the DB
func (dbAPI *dbObjectModelAPI) UpdateEdgeInfo(ctx context.Context, i interface{} /* *model.EdgeUsageInfo */, callback func(context.Context, interface{}) error) (interface{}, error) {
	var createCallback func(context.Context, interface{}) error
	if callback != nil {
		createCallback = func(ctx context.Context, in interface{}) error {
			doc := in.(model.CreateDocumentResponse)
			return callback(ctx, model.UpdateDocumentResponse{ID: doc.ID})
		}
	}
	resp := model.UpdateDocumentResponse{}
	createResp, err := dbAPI.CreateEdgeInfo(ctx, i, createCallback)
	if err == nil {
		resp.ID = createResp.(model.CreateDocumentResponse).ID
	}
	return resp, err
}

// UpdateEdgeW update an edgeInfo object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateEdgeInfoW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateEdgeInfo, &model.EdgeUsageInfo{}, w, r, callback)
}

// UpdateEdgeWV2 update an edgeInfo object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) UpdateEdgeInfoWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateEdgeInfo), &model.EdgeUsageInfo{}, w, r, callback)
}

// DeleteEdge delete a edge info object in the DB
func (dbAPI *dbObjectModelAPI) DeleteEdgeInfo(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	return dbAPI.DeleteEdgeDeviceInfo(context, id, callback)
}

// DeleteEdgeW delete a edge info object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteEdgeInfoW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteEdgeInfo, id, w, callback)
}

// DeleteEdgeWV2 delete a edge info object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteEdgeInfoWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteEdgeInfo), id, w, callback)
}
