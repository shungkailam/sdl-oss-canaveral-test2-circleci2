package api

import (
	"cloudservices/common/base"
	"cloudservices/operator/generated/operator/models"
	"cloudservices/operator/generated/operator/restapi/operations/edge"
	"cloudservices/operator/releases"
	"fmt"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/golang/glog"
)

func (server *APIServer) UploadReleaseHandler(params edge.UploadReleaseParams) middleware.Responder {
	reqID := base.GetUUID()
	if &params.UpgradeFiles == nil {
		glog.Errorf("Request %s: Upgrade files empty", reqID)
		errStr := "Upgrade files empty"
		retErr := &models.Error{Message: &errStr}
		return edge.NewUploadReleaseDefault(http.StatusInternalServerError).WithPayload(retErr)
	}
	releaseKey, err := releases.UploadRelease(params)
	if err != nil {
		glog.Errorf("Request %s: Failed to upload release %s", reqID, err)
		errStr := fmt.Sprintf("Failed to upload release: %s", err.Error())
		retErr := &models.Error{Message: &errStr}
		return edge.NewUploadReleaseDefault(http.StatusInternalServerError).WithPayload(retErr)
	}
	glog.Infof("Request %s: Uploaded release: %s\n", reqID, releaseKey)
	return edge.NewUploadReleaseOK().WithPayload(releaseKey)
}

func (server *APIServer) GetReleaseHandler(params edge.GetReleaseParams) middleware.Responder {
	reqID := base.GetUUID()
	release, err := releases.GetRelease(params.ReleaseID)
	if err != nil {
		glog.Errorf("Request %s: Failed to get release %s", reqID, err)
		errStr := fmt.Sprintf("Failed to get release: %s", err.Error())
		retErr := &models.Error{Message: &errStr}
		return edge.NewGetReleaseDefault(http.StatusInternalServerError).WithPayload(retErr)
	}
	glog.Infof("Request %s: Get release: %s success\n", reqID, release.ID)

	return edge.NewGetReleaseOK().WithPayload(&release)
}

func (server *APIServer) UpdateReleaseHandler(params edge.UpdateReleaseParams) middleware.Responder {
	reqID := base.GetUUID()
	releaseKey, err := releases.UpdateRelease(params)
	if err != nil {
		glog.Errorf("Request %s: Failed to update release %s", reqID, err)
		errStr := fmt.Sprintf("Failed to update release: %s", err.Error())
		retErr := &models.Error{Message: &errStr}
		return edge.NewUpdateReleaseDefault(http.StatusInternalServerError).WithPayload(retErr)
	}
	glog.Infof("Request %s: Update release: %s success\n", reqID, releaseKey)

	return edge.NewUpdateReleaseOK().WithPayload(releaseKey)
}

func (server *APIServer) DeleteReleaseHandler(params edge.DeleteReleaseParams) middleware.Responder {
	reqID := base.GetUUID()
	releaseKey, err := releases.DeleteRelease(params)
	if err != nil {
		glog.Errorf("Request %s: Failed to delete release %s", reqID, err)
		errStr := fmt.Sprintf("Failed to delete release: %s", err.Error())
		retErr := &models.Error{Message: &errStr}
		return edge.NewDeleteReleaseDefault(http.StatusInternalServerError).WithPayload(retErr)
	}

	glog.Infof("Request %s: Delete release: %s success", reqID, releaseKey)
	return edge.NewDeleteReleaseOK().WithPayload(releaseKey)
}

func (server *APIServer) ListReleasesHandler(params edge.ListReleasesParams) middleware.Responder {
	reqID := base.GetUUID()
	// for now we list the latest one
	releaseList, err := releases.GetAllReleases()
	if err != nil {
		glog.Errorf("Request %s: Failed to list releases %s", reqID, err)
		errStr := fmt.Sprintf("Failed to list releases: %s", err.Error())
		retErr := &models.Error{Message: &errStr}
		return edge.NewListReleasesDefault(http.StatusInternalServerError).WithPayload(retErr)
	}
	glog.Infof("Request %s: Available releases: success: {", reqID)
	for v := range releaseList {
		glog.Infof("%+v", releaseList[v])
	}
	glog.Infof("}")
	return edge.NewListReleasesOK().WithPayload(releaseList)
}

func (server *APIServer) ListCompatibleReleasesHandler(params edge.ListCompatibleReleasesParams) middleware.Responder {
	// First check if release id is valid,
	// skipping for now
	// _, err := releases.GetRelease(params.ReleaseID)
	// if err != nil {
	// 	glog.Errorf("Failed to list releases %s", err)
	// 	errStr := fmt.Sprintf("Failed to list releases:", err)
	// 	retErr := &models.Error{Message: &errStr}
	// 	return edge.NewListCompatibleReleasesDefault(http.StatusInternalServerError).WithPayload(retErr)
	// }

	// for now we list the latest one
	reqID := base.GetUUID()
	releaseList, err := releases.GetLatestRelease()
	if err != nil {
		glog.Errorf("Request %s: Failed to list releases %s", reqID, err.Error())
		errStr := fmt.Sprintf("Failed to list releases: %s", err.Error())
		retErr := &models.Error{Message: &errStr}
		return edge.NewListCompatibleReleasesDefault(http.StatusInternalServerError).WithPayload(retErr)
	}
	trimmedReleases := models.ReleaseList{}
	glog.Infof("Request %s: Compatible releases for a given version, success: {", reqID)
	for _, v := range releaseList {
		is, err := releases.IsSmaller(v.ID, params.ReleaseID)
		if err != nil {
			glog.Errorf("Request %s: Failed to convert give release , check provided format %s", reqID, err.Error())
			errStr := fmt.Sprintf("Failed to convert give release , check provided format: %s", err.Error())
			retErr := &models.Error{Message: &errStr}
			return edge.NewListCompatibleReleasesDefault(http.StatusInternalServerError).WithPayload(retErr)
		}
		if is == true {
			// Do not return if the given id is greater than the version provided
			continue
		}
		glog.Infof("%+v", v)
		trimmedReleases = append(trimmedReleases, v)
	}
	glog.Infof("}")
	return edge.NewListCompatibleReleasesOK().WithPayload(trimmedReleases)
}
