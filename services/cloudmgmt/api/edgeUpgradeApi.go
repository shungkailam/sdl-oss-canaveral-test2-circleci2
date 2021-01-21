package api

import (
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/meta"
	"cloudservices/common/model"
	"cloudservices/common/service"
	gapi "cloudservices/operator/generated/grpc"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/go-openapi/strfmt"
	"github.com/golang/glog"
	"github.com/jmoiron/sqlx/types"
	"google.golang.org/grpc"
)

func init() {
}

var nilVersion = base.StringPtr("v1.0.0")

// EdgeUpgradeDBO is DB object model for edgeUpgrade
type EdgeUpgradeDBO struct {
	model.BaseModelDBO
	Release            string          `json:"release"`
	CompatibleReleases *types.JSONText `json:"compatibleReleases"`
	Changelog          string          `json:"changelog"`
}

func (dbAPI *dbObjectModelAPI) getECRLogin() (string, error) {

	// Get auth token from aws
	svc := ecr.New(session.New(&aws.Config{
		Region:      aws.String(*config.Cfg.AWSRegion),
		Credentials: credentials.NewStaticCredentials(*config.Cfg.OtaAccessKey, *config.Cfg.OtaSecretKey, ""),
	}))
	input := &ecr.GetAuthorizationTokenInput{}
	result, err := svc.GetAuthorizationToken(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeServerException:
				return "", fmt.Errorf("%s: %s", ecr.ErrCodeServerException, aerr.Error())
			case ecr.ErrCodeInvalidParameterException:
				return "", fmt.Errorf("%s: %s", ecr.ErrCodeInvalidParameterException, aerr.Error())
			default:
				return "", fmt.Errorf(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			return "", fmt.Errorf(err.Error())
		}
	}

	// Extract base 64 decoded aws pass
	creds, err := base64.StdEncoding.DecodeString(*result.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		return "", fmt.Errorf(err.Error())
	}
	userPass := strings.Split(string(creds), ":")
	if len(userPass) != 2 {
		err = fmt.Errorf("userpass can not be extracted from creds: %s", creds)
		return "", fmt.Errorf(err.Error())
	}
	pass := userPass[1]
	repo := *result.AuthorizationData[0].ProxyEndpoint
	login := fmt.Sprintf("docker login -u AWS -p %s %s", pass, repo)

	return login, nil
}

// SelectAllEdgeUpgrades select all edgeUpgrades
func (dbAPI *dbObjectModelAPI) SelectAllEdgeUpgrades(ctx context.Context) ([]model.EdgeUpgradeCore, error) {

	upgrades := []model.EdgeUpgradeCore{}
	reqID := base.GetRequestID(ctx)
	request := &gapi.ListReleasesRequest{}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		releases, err := client.ListReleases(ctx, request)
		if err != nil {
			glog.Errorf("Request %s: Error: %s", reqID, err.Error())
			return err
		}
		//Note: The operator List Releases method returns n and n-1 release from build v1.15.0
		//So,to make the api backward compatible, return only latest version
		release := releases.Releases[0]
		upgrade := model.EdgeUpgradeCore{}
		upgrade.Changelog = release.Changelog
		upgrade.Release = release.Id
		upgrades = append(upgrades, upgrade)
		return nil
	}
	err := service.CallClient(ctx, service.OperatorService, handler)

	return upgrades, err
}

// SelectAllEdgeUpgradesW select all edgeUpgrades, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgeUpgradesW(context context.Context, w io.Writer, req *http.Request) error {
	upgrades, err := dbAPI.SelectAllEdgeUpgrades(context)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, upgrades)
}

// SelectAllEdgeUpgradesWV2 select all edgeUpgrades, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgeUpgradesWV2(context context.Context, w io.Writer, req *http.Request) error {
	upgrades, err := dbAPI.SelectAllEdgeUpgrades(context)
	if err != nil {
		return err
	}
	queryParam := model.GetEntitiesQueryParam(req)
	queryInfo := ListQueryInfo{
		StartPage:  base.PageToken(""),
		TotalCount: len(upgrades),
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &queryInfo)

	r := model.EdgeUpgradeListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		EdgeUpgradeCoreList:       upgrades,
	}
	return json.NewEncoder(w).Encode(r)
}

// SelectEdgeUpgradesByEdgeID select edgeUpgrades by ID
func (dbAPI *dbObjectModelAPI) SelectEdgeUpgradesByEdgeID(ctx context.Context, edgeid string) ([]model.EdgeUpgradeCore, error) {
	reqID := base.GetRequestID(ctx)

	edgeUpgrades := []model.EdgeUpgradeCore{}
	edgeInfo, err := dbAPI.GetEdgeInfo(ctx, edgeid)
	if err != nil {
		return edgeUpgrades, fmt.Errorf("Request %s SelectEdgeUpgradesByEdgeID: could not get edgeInfo %s", reqID, err)
	}
	// How do we handle old edges?
	if edgeInfo.EdgeVersion == nil {
		edgeInfo.EdgeVersion = nilVersion
	}

	upgrades := []model.EdgeUpgradeCore{}
	request := &gapi.ListCompatibleReleasesRequest{Id: *edgeInfo.EdgeVersion}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		releases, err := client.ListCompatibleReleases(ctx, request)
		if err != nil {
			glog.Errorf("Request %s: Error: %s", reqID, err.Error())
			return err
		}
		//Note: The operator ListReleases API returns n and n-1 release from build v1.15.0
		//So,to make this api backward compatible, return only latest version
		release := releases.Releases[0]
		upgrade := model.EdgeUpgradeCore{}
		upgrade.Changelog = release.Changelog
		upgrade.Release = release.Id
		upgrades = append(upgrades, upgrade)
		return nil
	}
	err = service.CallClient(ctx, service.OperatorService, handler)

	return upgrades, err

}

// SelectEdgeUpgradesByEdgeIDW select edgeUpgrades by ID, write output into writer
func (dbAPI *dbObjectModelAPI) SelectEdgeUpgradesByEdgeIDW(context context.Context, edgeid string, w io.Writer, req *http.Request) error {
	upgrades, err := dbAPI.SelectEdgeUpgradesByEdgeID(context, edgeid)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, upgrades)
}

func (dbAPI *dbObjectModelAPI) ExecuteEdgeUpgrade(ctx context.Context, i interface{} /* *model.EdgeUpgrade */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.ExecuteEdgeUpgrade)
	if !ok {
		return resp, fmt.Errorf("ExecuteEdgeUpgrade: type error")
	}
	doc := *p
	if doc.Release == "" {
		return resp, fmt.Errorf("ExecuteEdgeUpgrade: Release is required")
	}
	v := doc.Release
	vvv := strings.Split(v, ".")
	if len(vvv) != 3 {
		return resp, fmt.Errorf("ExecuteEdgeUpgrade: Release needs to be of the form v{major}.{minor}.{bugfix}")
	}
	// TODO revision version format
	// Only infra admin can update
	//tenantID := authContext.TenantID
	err = auth.CheckRBAC(
		authContext,
		meta.EntityEdgeUpgrade,
		meta.OperationCreate,
		auth.RbacContext{})

	if err != nil {
		return resp, err
	}

	// Use grpc to get the data for upgrading from operator
	executeEdgeUpgradeParam := model.ExecuteEdgeUpgradeData{}
	reqID := base.GetRequestID(ctx)
	request := &gapi.GetReleaseRequest{Id: doc.Release}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		release, err := client.GetRelease(ctx, request)
		if err != nil {
			glog.Errorf("Request %s: Error: %s", reqID, err.Error())
			return err
		}
		// Due to ENG-249164 GRPC max byte size issue, release.Data is not set
		data := strfmt.Base64(release.Data)
		executeEdgeUpgradeParam.UpgradeData = &data
		executeEdgeUpgradeParam.UpgradeURL = release.Url
		return nil
	}
	err = service.CallClient(ctx, service.OperatorService, handler)
	if err != nil {
		return resp, err
	}
	executeEdgeUpgradeParam.Release = doc.Release
	// Get docker creds for downloading the images from our account
	executeEdgeUpgradeParam.DockerLogin, err = dbAPI.getECRLogin()
	if err != nil {
		return resp, err
	}

	now := time.Now()
	epochInNanoSecs := now.UnixNano()
	executeEdgeUpgradeParam.Version = float64(epochInNanoSecs)
	executeEdgeUpgradeParam.UpdatedAt = now
	executeEdgeUpgradeParam.ID = base.GetUUID()
	upgradeStartMsg := model.EventUpsertRequest{}
	if doc.Force == false {
		// NOTE: This is a fix to reduce grpc calls, but does not check if all edges are valid for upgrade
		// Validate that the edges have valid versions before updating
		// Get the valid upgrades, returned by operator
		upgrades, err := dbAPI.SelectAllEdgeUpgrades(ctx)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get list edge upgrades. Error: %s"), err.Error())
			return resp, err
		}
		latestVersion := upgrades[0].Release
		if doc.Release != latestVersion {
			glog.Errorf(base.PrefixRequestID(ctx, "A newer upgrade is avilable requested: %s, available: %s"), doc.Release, latestVersion)
			return resp, errcode.NewBadRequestError("release")
		}
	}
	batchEdges := []string{}
	for idx, edgeID := range doc.EdgeIDs {
		batchEdges = append(batchEdges, edgeID)
		edgeUpgradeStartEvent := []model.Event{
			model.Event{
				Timestamp: now,
				Type:      "STATUS",
				Path:      fmt.Sprintf("/serviceDomain:%s/upgrade:%s:%s/event", edgeID, doc.Release, executeEdgeUpgradeParam.ID),
				State:     "Starting",
				Version:   "v1",
				Message:   "Sending upgrade message to the Edge",
			},
			model.Event{
				Timestamp: now,
				Type:      "STATUS",
				Path:      fmt.Sprintf("/serviceDomain:%s/upgrade:%s:%s/progress", edgeID, doc.Release, executeEdgeUpgradeParam.ID),
				State:     "Starting",
				Version:   "v1",
				Message:   "0%",
			},
		}
		upgradeStartMsg.Events = append(upgradeStartMsg.Events, edgeUpgradeStartEvent...)
		// NOTE: batching the events with maxEventsBatchSize to prevent overwhelming the eventserver
		// TODO: handle errors if any call fails
		if len(upgradeStartMsg.Events) == *config.Cfg.EventsMaxBatchSize || idx == len(doc.EdgeIDs)-1 {
			go func(upgradeStartMsg model.EventUpsertRequest, batchEdges []string) {
				newCtx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
				newCtx = context.WithValue(newCtx, base.RequestIDKey, reqID)
				_, err = dbAPI.UpsertEvents(newCtx, upgradeStartMsg, nil)
				if err != nil {
					glog.Errorf(base.PrefixRequestID(ctx, "Failed to insert edge upgrades in elasticsearch. Error: %s"), err.Error())
				}
				if callback != nil {
					for _, batchEdgeID := range batchEdges {
						executeEdgeUpgradeParam.EdgeID = batchEdgeID
						edgeUpgradeParam := executeEdgeUpgradeParam
						edgeInfo, err := dbAPI.GetEdgeInfo(ctx, batchEdgeID)
						if edgeInfo.EdgeVersion == nil || err != nil {
							// Use old version for upgrade as we need the data
							edgeInfo.EdgeVersion = nilVersion
						}
						feats, _ := GetFeaturesForVersion(*edgeInfo.EdgeVersion)
						if feats.URLupgrade == true {
							// Remove the data for newer edges
							var emptyStr strfmt.Base64
							edgeUpgradeParam.UpgradeData = &emptyStr
						}
						// Can leave out data here for older edges
						go callback(newCtx, &edgeUpgradeParam)
					}
				}
			}(upgradeStartMsg, batchEdges)
			upgradeStartMsg = model.EventUpsertRequest{}
			batchEdges = []string{}
		}
	}

	resp.ID = doc.Release
	return resp, nil
}

// ExecuteEdgeUpgradeW executes an  edge upgrade and writes output into writer
func (dbAPI *dbObjectModelAPI) ExecuteEdgeUpgradeW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.ExecuteEdgeUpgrade, &model.ExecuteEdgeUpgrade{}, w, r, callback)
}

// ExecuteEdgeUpgradeWV2 executes an  edge upgrade and writes output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) ExecuteEdgeUpgradeWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.ExecuteEdgeUpgrade), &model.ExecuteEdgeUpgrade{}, w, r, callback)
}
