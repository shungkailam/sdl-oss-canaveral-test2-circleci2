package websocket

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
)

// ModifyExecuteEdgeUpgradeData modifies the payload to remove or add the binary data
func ModifyExecuteEdgeUpgradeData(dbAPI api.ObjectModelAPI, edgeID, tenantID string, msg interface{}) (interface{}, error) {
	upgradeReqMap := msg.(map[string]interface{})
	reqID := upgradeReqMap["requestId"].(string)
	upgradeDocMap := upgradeReqMap["doc"].(map[string]interface{})
	ctx := base.GetAdminContext(reqID, tenantID)
	glog.Infof(base.PrefixRequestID(ctx, "Send message executeEdgeUpgrade: get data for upgrade %s\n"), edgeID)
	edgeInfo, err := dbAPI.GetEdgeInfo(ctx, edgeID)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get edge info for %s in executeEdgeUpgrade. Error: %s\n"), edgeID, err.Error())
		return nil, err
	}
	var ver string
	if edgeInfo.EdgeVersion == nil {
		ver = "v1.0.0"
	} else {
		ver = *edgeInfo.EdgeVersion
	}
	feats, err := api.GetFeaturesForVersion(ver)
	if err != nil {
		return nil, err
	}
	if feats.URLupgrade == true {
		glog.Infof(base.PrefixRequestID(ctx, "Send message executeEdgeUpgrade URL upgrade supported %s\n"), edgeID)
		// If the edge supports being upgraded by url
		upgradeDocMap["data"] = ""
	} else if upgradeDocMap["data"] == "" {
		// Get the data from the url
		glog.Infof(base.PrefixRequestID(ctx, "Send message executeEdgeUpgrade getting upgrade data %s\n"), edgeID)
		resp, err := http.Get(upgradeDocMap["upgradeURL"].(string))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		upgradeDocMap["data"] = data
	}
	return msg, nil
}

func handleUpgradeError(tenantID string, edgeID string, msgName string, msg interface{}, err error) error {
	switch msg.(type) {
	// type for redis cloudinstance
	case map[string]interface{}:
		upgradeReqMap := msg.(map[string]interface{})
		reqID := upgradeReqMap["requestId"].(string)
		tenantID := upgradeReqMap["tenantId"].(string)
		upgradeDocMap := upgradeReqMap["doc"].(map[string]interface{})
		releaseVersion := upgradeDocMap["release"].(string)
		releaseID := upgradeDocMap["id"].(string)
		ctx := context.WithValue(context.Background(), base.RequestIDKey, reqID)
		claims := jwt.MapClaims{
			"specialRole": "admin",
		}
		authContext := &base.AuthContext{TenantID: tenantID, Claims: claims}
		ctx = context.WithValue(ctx, base.AuthContextKey, authContext)
		event := model.UpgradeEvent{ID: releaseID, TenantID: tenantID, EdgeID: edgeID, Err: err,
			EventState: "Failed", ReleaseVersion: releaseVersion}
		eventerr := base.Publisher.Publish(ctx, &event)
		if eventerr != nil {
			return fmt.Errorf("Failed to publish event %v. Error: %s", event, err.Error())
		}
	// type for websocket send
	case api.ObjectRequest:
		upgradeObj := msg.(api.ObjectRequest)
		upgrademsg := upgradeObj.Doc.(*model.ExecuteEdgeUpgradeData)
		ctx := context.WithValue(context.Background(), base.RequestIDKey, upgradeObj.RequestID)
		claims := jwt.MapClaims{
			"specialRole": "admin",
		}
		authContext := &base.AuthContext{TenantID: upgradeObj.TenantID, Claims: claims}
		ctx = context.WithValue(ctx, base.AuthContextKey, authContext)
		event := model.UpgradeEvent{ID: upgrademsg.ID, TenantID: upgradeObj.TenantID, EdgeID: upgrademsg.EdgeID, Err: err,
			EventState: "Failed", ReleaseVersion: upgrademsg.Release}
		eventerr := base.Publisher.Publish(ctx, &event)
		if eventerr != nil {
			glog.Errorf("Failed to publish event %v. Error: %s", event, err.Error())
		}
	default:
		return fmt.Errorf("Interface of type %T not supported", msg)
	}
	return nil
}
