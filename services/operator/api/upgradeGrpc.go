package api

import (
	"cloudservices/common/base"
	"cloudservices/common/metrics"
	gapi "cloudservices/operator/generated/grpc"
	"cloudservices/operator/releases"
	"context"
	"os"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

// GetRelease is used to get release info including data
func (server *rpcServer) GetRelease(ctx context.Context, request *gapi.GetReleaseRequest) (*gapi.Release, error) {
	metrics.GRPCCallCount.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "method": "GetRelease"}).Inc()
	reqID := base.GetRequestID(ctx)
	glog.Infof("Request %s: %+v", reqID, request)
	release, err := releases.GetRelease(request.GetId())
	if err != nil {
		glog.Infof("Request %s: Error: %s", reqID, err.Error())
		return nil, err
	}
	protoRelease := &gapi.Release{}
	protoRelease.Id = release.ID
	protoRelease.Changelog = release.Changelog
	protoRelease.Data = []byte(release.Data)
	protoRelease.Url = release.URL
	return protoRelease, nil
}

// ListReleases is used to get release info including data
func (server *rpcServer) ListReleases(ctx context.Context, request *gapi.ListReleasesRequest) (*gapi.ListReleasesResponse, error) {
	metrics.GRPCCallCount.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "method": "ListReleases"}).Inc()
	reqID := base.GetRequestID(ctx)
	glog.Infof("Request %s: %+v", reqID, request)
	// Only return the latest release
	releases, err := releases.GetLatestRelease()
	if err != nil {
		glog.Infof("Request %s: Error: %s", reqID, err.Error())
		return nil, err
	}
	protoReleases := &gapi.ListReleasesResponse{}

	for _, release := range releases {
		protoRelease := &gapi.Release{}
		protoRelease.Id = release.ID
		protoRelease.Changelog = release.Changelog
		protoReleases.Releases = append(protoReleases.Releases, protoRelease)
	}

	return protoReleases, nil
}

// ListCompatibleReleases is used to get release info including data
func (server *rpcServer) ListCompatibleReleases(ctx context.Context, request *gapi.ListCompatibleReleasesRequest) (*gapi.ListReleasesResponse, error) {
	metrics.GRPCCallCount.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "method": "ListCompatibleReleases"}).Inc()
	glog.Infof(base.PrefixRequestID(ctx, "Requesting compatible releases %+v"), request)
	releasList, err := releases.GetLatestRelease()
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in listing compatible releases: %s"), err.Error())
		return nil, err
	}
	protoReleases := &gapi.ListReleasesResponse{}

	for _, release := range releasList {
		is, err := releases.IsSmaller(release.ID, request.GetId())
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert give release , check provided format. Error: %s"), err.Error())
			return protoReleases, err
		}
		if is == true {
			// Do not return if the given id is greater than the version provided
			continue
		}
		protoRelease := &gapi.Release{}
		protoRelease.Id = release.ID
		protoRelease.Changelog = release.Changelog
		protoReleases.Releases = append(protoReleases.Releases, protoRelease)
	}

	return protoReleases, nil
}

// GetReleaseHelmChart returns the current release helm chart
func (server *rpcServer) GetReleaseHelmChart(ctx context.Context, request *gapi.GetReleaseHelmChartRequest) (*gapi.GetReleaseHelmChartResponse, error) {
	metrics.GRPCCallCount.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "method": "GetReleaseHelmChart"}).Inc()
	release, err := releases.GetReleaseHelmChart(ctx)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting release helm chart: %s"), err.Error())
		return nil, err
	}
	resp := &gapi.GetReleaseHelmChartResponse{
		Release: &gapi.ReleaseHelmChart{
			Id:  release.ID,
			Url: release.URL,
		},
	}
	return resp, nil
}
