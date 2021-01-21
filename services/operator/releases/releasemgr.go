package releases

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/operator/common"
	"cloudservices/operator/config"
	"cloudservices/operator/generated/operator/models"
	"cloudservices/operator/generated/operator/restapi/operations/edge"
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/golang/glog"
)

const (
	MaxListReleaseSize = 3

	// <repo>/importer-<version>.tgz
	ReleaseHelmChartURLFormat = "%s/importer-%s.tgz"
)

type Release struct {
	major  int
	minor  int
	bugfix int
}

func (ar *Release) isLarger(br *Release) bool {
	if ar.major > br.major {
		return true
	} else if ar.major < br.major {
		return false
	}
	if ar.minor > br.minor {
		return true
	} else if ar.minor < br.minor {
		return false
	}

	if ar.bugfix > br.bugfix {
		return true
	}
	return false

}

func releaseToString(release *Release) string {
	return (*config.Cfg.S3Prefix + "v" + strconv.Itoa(release.major) + "/" + strconv.Itoa(release.minor) + "/" + strconv.Itoa(release.bugfix) + "/" +
		"v" + strconv.Itoa(release.major) + "." + strconv.Itoa(release.minor) + "." + strconv.Itoa(release.bugfix) + ".tar.gz")
}

func releaseToChangeLogKey(release *Release) string {
	return (*config.Cfg.S3Prefix + "v" + strconv.Itoa(release.major) + "/" + strconv.Itoa(release.minor) + "/" + strconv.Itoa(release.bugfix) + "/" +
		"Changelog.txt")
}

func releaseToFolderKey(release *Release) string {
	return (*config.Cfg.S3Prefix + "v" + strconv.Itoa(release.major) + "/" + strconv.Itoa(release.minor) + "/" + strconv.Itoa(release.bugfix))
}

// releaseToPackages returns the path to packages containing the files extracted from the release tar.gz file
func releaseToPackages(release *Release) string {
	return (*config.Cfg.S3Prefix + "v" + strconv.Itoa(release.major) + "/" + strconv.Itoa(release.minor) + "/" + strconv.Itoa(release.bugfix) + "/packages")
}

func releaseToKey(release *Release) string {
	return ("v" + strconv.Itoa(release.major) + "." + strconv.Itoa(release.minor) + "." + strconv.Itoa(release.bugfix))
}

func stringToRelease(str string) (*Release, error) {
	pair := strings.Split(str, "/")
	// any number before dots
	re := regexp.MustCompile("[0-9]+")
	// get version from the last part(file name) in the s3 key url
	ver := re.FindAllString(pair[len(pair)-1], -1)

	rel := &Release{}
	var err error
	if len(ver) < 3 {
		// Can skip for changelog as we can not extract version from changelog
		//glog.Warningln("stringToRelease: Could not decode release ", str)
		err := errors.New("stringToRelease: Could not decode release ")
		return nil, err
	}
	rel.major, err = strconv.Atoi(ver[0])
	if err != nil {
		glog.Errorf("stringToRelease: Could not decode release %s ", ver[0])
		return nil, err
	}
	rel.minor, err = strconv.Atoi(ver[1])
	if err != nil {
		glog.Errorf("stringToRelease: Could not decode release %s", ver[1])
		return nil, err
	}
	rel.bugfix, err = strconv.Atoi(ver[2])
	if err != nil {
		glog.Errorf("stringToRelease: Could not decode release %s", ver[2])
		return nil, err
	}
	return rel, err
}

// presignURL returns the presigned URL of the S3 key
func presignURL(s3Key string) (string, error) {
	sess := common.GetAWSSession()
	svc := s3.New(sess)
	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(*config.Cfg.S3Bucket),
		Key:    aws.String(s3Key),
	})
	url, err := req.Presign(1 * time.Hour)
	if err != nil {
		glog.Errorf("Error in signing URL S3 key %s. Error: %s", s3Key, err.Error())
		return "", errcode.NewInternalError(fmt.Sprintf("Unable to get the presigned URL for %s", s3Key))
	}
	return url, nil
}

// uploadToS3 uploads the file to S3 with S3 key.
// If the packages is set, the file is extracted and the contents are placed in a sibling folder packages
func uploadToS3(file io.Reader, s3Key, packages string) error {
	sess := common.GetAWSSession()
	s3Bucket := *config.Cfg.S3Bucket
	// To do return md5 sum
	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(sess)
	svc := s3.New(sess)
	// Upload input parameters
	upParams := &s3manager.UploadInput{
		Bucket: &s3Bucket,
		Key:    &s3Key,
		Body:   file,
	}
	_, err := uploader.Upload(upParams)
	if err != nil {
		glog.Errorf("Unable to upload object %s to S3. Error: %s", s3Key, err.Error())
		return errcode.NewInternalError(err.Error())
	}
	packages = strings.TrimSpace(packages)
	if packages != "" {
		ctx, cancelFn := context.WithCancel(context.Background())
		defer cancelFn()
		glog.Infof("Deleting existing packages if any in %s", packages)
		err = base.DeleteS3Objects(ctx, sess, s3Bucket, packages, true)
		if err != nil {
			glog.Errorf("Unable to clean existing packages %s from S3. Error: %s", packages, err.Error())
			return errcode.NewInternalError(err.Error())
		}
		glog.Infof("Extracting object %s into %s folder", s3Key, packages)
		// Extract the object and upload the contents to packages folder
		// Read the object from S3
		response, err := svc.GetObjectWithContext(ctx, &s3.GetObjectInput{
			Bucket: &s3Bucket,
			Key:    &s3Key,
		})
		if err != nil {
			glog.Errorf("Unable to read object %s from S3. Error: %s", s3Key, err.Error())
			return errcode.NewInternalError(err.Error())
		}
		reader := response.Body
		defer reader.Close()
		err = base.UploadTarContentsToS3(ctx, sess, s3Bucket, packages, reader)
		if err != nil {
			glog.Errorf("Unable to extract and upload object %s to S3. Error: %s", s3Key, err.Error())
			return errcode.NewInternalError(err.Error())
		}
	}
	return nil
}

// IsSmaller This function internally calls isLarger and returns the value
func IsSmaller(a string, b string) (bool, error) {
	l, err := isLarger(a, b)
	return !l, err
}

func isLarger(a string, b string) (bool, error) {

	ar, err := stringToRelease(a)
	if err != nil {
		glog.Errorf("isLarger: Could not compare %s", err)
		return false, err
	}
	br, err := stringToRelease(b)
	if err != nil {
		glog.Errorf("isLarger: Could not compare %s", err)
		return false, err
	}

	if ar.major > br.major {
		return true, nil
	} else if ar.major < br.major {
		return false, nil
	}

	if ar.minor > br.minor {
		return true, nil
	} else if ar.minor < br.minor {
		return false, nil
	}

	if ar.bugfix > br.bugfix {
		return true, nil
	}

	return false, nil
}

func DownloadChangeLog(changeLogKey string) string {
	sess := common.GetAWSSession()
	downloader := s3manager.NewDownloader(sess)
	// get the changelog
	clogParams := &s3.GetObjectInput{
		Bucket: config.Cfg.S3Bucket,
		Key:    &changeLogKey,
	}
	clogData := &aws.WriteAtBuffer{}
	_, err := downloader.Download(clogData, clogParams)
	if err != nil {
		glog.Errorf("Unable to get changelog info from s3, %s", err)
		return ""
	}
	return string(clogData.Bytes())
}

// GetLatestRelease returns the latest release as per s3.
func GetLatestRelease() (models.ReleaseList, error) {
	var releaseListModel models.ReleaseList
	var releases []*Release
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	sess := common.GetAWSSession()

	// Create S3 service client
	svc := s3.New(sess)
	s3bucket := *config.Cfg.S3Bucket
	s3prefix := *config.Cfg.S3Prefix
	request := &s3.ListObjectsInput{
		Bucket:    aws.String(s3bucket),
		Prefix:    aws.String(s3prefix),
		Delimiter: aws.String("/packages"),
	}
	err := svc.ListObjectsPagesWithContext(ctx, request, func(page *s3.ListObjectsOutput, lastPage bool) bool {
		for _, item := range page.Contents {
			// skip folders and empty objects
			if *item.Size == int64(0) {
				continue
			}
			// Version is in the file name
			if !strings.HasSuffix(*item.Key, ".tar.gz") {
				continue
			}
			rel, err := stringToRelease(*item.Key)
			if err != nil {
				// do not stop if object has wrong type thus always show some release
				// do we want to error out if there is an error in bucket format?
				glog.Errorf("String to Release Object failed: %s", err.Error())
				continue
			}
			releases = append(releases, rel)
		}
		// continue or not
		return !lastPage
	})
	if err != nil {
		glog.Errorf("Unable to list objects, %s", err)
		return nil, errcode.NewInternalError(err.Error())
	}
	//Note: This shouldnot happen in production.
	if len(releases) == 0 {
		glog.Warningln("Didn't find any release")
		//Note: To make it backward compatible ,load default release
		releaseListModel = append(releaseListModel, &models.Release{
			ID: "v1.0.0",
		})
		return releaseListModel, nil
	}
	sort.Slice(releases[:], func(i, j int) bool {
		return releases[i].isLarger(releases[j])
	})
	releaseCount := 0
	lastMajor := -1
	lastMinor := -1
	for _, release := range releases {
		if releaseCount >= MaxListReleaseSize {
			break
		}
		// They always come in non-increasing order
		if lastMajor == release.major && lastMinor == release.minor {
			continue
		}
		releaseKey := releaseToKey(release)
		changeLogKey := releaseToChangeLogKey(release)
		changeLogStr := DownloadChangeLog(changeLogKey)
		releaseListModel = append(releaseListModel, &models.Release{
			ID:        releaseKey,
			Changelog: changeLogStr,
		})
		lastMajor = release.major
		lastMinor = release.minor
		releaseCount++
	}
	return releaseListModel, nil
}

// UploadRelease uses the params to determine the file name and uploads the file to s3
func UploadRelease(params edge.UploadReleaseParams) (string, error) {

	releaseList, err := GetLatestRelease()
	if err != nil {
		glog.Errorf("Unable to get latest release, %s", err)
		return "", errcode.NewInternalError(fmt.Sprintf("Unable to get latest release, %s", err))
	}
	releaseObj, err := stringToRelease(releaseList[0].ID)
	if err != nil {
		glog.Errorf("Unable to convert the last release to string, %s", err)
		return "", errcode.NewInternalError(fmt.Sprintf("Unable to convert the last release to string, %s", err))
	}
	switch params.UpgradeType {
	case "major":
		releaseObj.major++
		releaseObj.minor = 0
		releaseObj.bugfix = 0
	case "minor":
		releaseObj.minor++
		releaseObj.bugfix = 0
	case "bugfix":
		releaseObj.bugfix++
	default:
		err := errcode.NewBadRequestError("UpgradeType")
		glog.Errorf("Only major, minor or bugfix are allowed types")
		return "", err
	}
	// e.g <s3 prefix>/v1/15/0/v1.15.0.tar.gz
	s3Key := releaseToString(releaseObj)
	// e.g <s3 prefix>/v1/15/0/Changelog.txt
	changeLogKey := releaseToChangeLogKey(releaseObj)
	// e.g v1.15.0
	opKey := releaseToKey(releaseObj)
	// e.g <s3 prefix>/v1/15/0/packages
	packages := releaseToPackages(releaseObj)

	err = uploadToS3(params.UpgradeFiles, s3Key, packages)
	if err != nil {
		return "", errcode.NewInternalError(fmt.Sprintf("Unable to upload file to s3, %s", err))
	}
	if params.Changelog == nil {
		return "", errcode.NewBadRequestExError("Changelog", "Changelog string cannot be empty")
	}

	// Upload the change log
	reader := strings.NewReader(*params.Changelog)
	err = uploadToS3(reader, changeLogKey, "")
	if err != nil {
		return "", errcode.NewInternalError(fmt.Sprintf("Unable to upload changelog to s3, %s", err))
	}

	return opKey, nil

}

// UpdateRelease uses the params to determine the file name and uploads the file to s3
func UpdateRelease(params edge.UpdateReleaseParams) (string, error) {
	// inter conversion to convert to correct format
	releaseObj, err := stringToRelease(params.ReleaseID)
	if err != nil {
		glog.Errorf("Unable to get release, please check id, %s", err)
		return "", errcode.NewInternalError(fmt.Sprintf("Unable to get release, please check id: %s", err))
	}
	s3Key := releaseToString(releaseObj)
	changeLogKey := releaseToChangeLogKey(releaseObj)
	opKey := releaseToKey(releaseObj)
	packages := releaseToPackages(releaseObj)

	err = uploadToS3(params.UpgradeFiles, s3Key, packages)
	if err != nil {
		glog.Errorf("Unable to upload file to s3, %s", err)
		return "", errcode.NewInternalError(fmt.Sprintf("Unable to upload file to s3, %s", err))
	}
	// Upload the change log
	if params.Changelog != nil {
		reader := strings.NewReader(*params.Changelog)
		err := uploadToS3(reader, changeLogKey, "")
		if err != nil {
			glog.Errorf("Unable to changelog to s3, %s", err)
			return "", errcode.NewInternalError(fmt.Sprintf("Unable to upload file to s3, %s", err))
		}
	}
	return opKey, nil
}

// DeleteRelease deletes the release from the s3 bucket
func DeleteRelease(params edge.DeleteReleaseParams) (string, error) {
	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()
	// inter conversion to convert to correct format
	releaseObj, err := stringToRelease(params.ReleaseID)
	if err != nil {
		glog.Errorf("Unable to get release, please check id, %s", err)
		return "", err
	}
	sess := common.GetAWSSession()
	s3bucket := *config.Cfg.S3Bucket
	s3Key := releaseToFolderKey(releaseObj)
	err = base.DeleteS3Objects(ctx, sess, s3bucket, s3Key, true)
	if err != nil {
		glog.Errorf("Unable to delete release %s from S3. Error: %s", s3Key, err.Error())
		return "", errcode.NewInternalError(err.Error())
	}
	opKey := releaseToKey(releaseObj)
	return opKey, nil
}

// GetRelease gets the release info the s3 bucket
func GetRelease(key string) (models.ReleaseData, error) {
	var release models.ReleaseData
	sess := common.GetAWSSession()
	// inter conversion to convert to correct format
	releaseObj, err := stringToRelease(key)
	if err != nil {
		glog.Errorf("Unable to get release, please check id, %s", err)
		return release, errcode.NewInternalError(fmt.Sprintf("Unable to get release, please check id: %s", err))
	}
	s3Key := releaseToString(releaseObj)
	changeLogKey := releaseToChangeLogKey(releaseObj)
	opKey := releaseToKey(releaseObj)

	// Create a downloader with the session and default options
	downloader := s3manager.NewDownloader(sess)
	// Download input parameters
	// Commenting out due to ENG-249164 GRPC max byte size issue
	/*
		dnParams := &s3.GetObjectInput{
			Bucket: config.Cfg.S3Bucket,
			Key:    &s3Key,
		}

		data := &aws.WriteAtBuffer{}

		// Perform the download.
		// We can use the below result to get s3 bucket url etc..
		// result, err = uploader.Upload(upParams)
		_, err = downloader.Download(data, dnParams)
		if err != nil {
			glog.Errorf("Unable to get release info from s3 %#v, %s", dnParams, err)
			return release, errcode.NewInternalError(fmt.Sprintf("%s", err))
		}
	*/

	// get changelog
	clogParams := &s3.GetObjectInput{
		Bucket: config.Cfg.S3Bucket,
		Key:    &changeLogKey,
	}

	clogData := &aws.WriteAtBuffer{}
	_, err = downloader.Download(clogData, clogParams)
	if err != nil {
		glog.Errorf("Unable to get changelog info from s3, %s", err)
		//do not error out
	} else {
		release.Changelog = string(clogData.Bytes())
	}

	release.ID = opKey
	// Get the presigned url for the data
	url, err := presignURL(s3Key)
	if err != nil {
		return release, err
	}
	release.URL = url
	return release, nil
}

// GetAllReleases returns the latest release as per s3
func GetAllReleases() (models.ReleaseList, error) {
	var releaseListModel models.ReleaseList
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	sess := common.GetAWSSession()

	// Create S3 service client
	svc := s3.New(sess)
	s3bucket := *config.Cfg.S3Bucket
	s3prefix := *config.Cfg.S3Prefix
	request := &s3.ListObjectsInput{
		Bucket:    aws.String(s3bucket),
		Prefix:    aws.String(s3prefix),
		Delimiter: aws.String("/packages"),
	}

	err := svc.ListObjectsPagesWithContext(ctx, request, func(page *s3.ListObjectsOutput, lastPage bool) bool {
		for _, item := range page.Contents {
			var releaseModel models.Release
			// skip folders and empty objects
			if *item.Size == int64(0) {
				continue
			}
			// Version is in the file name
			if !strings.HasSuffix(*item.Key, ".tar.gz") {
				continue
			}
			rel, err := stringToRelease(*item.Key)
			if err != nil {
				//not a valid release, continue
				continue
			}
			key := releaseToKey(rel)
			releaseModel.ID = key
			releaseListModel = append(releaseListModel, &releaseModel)
		}
		return !lastPage
	})
	if err != nil {
		glog.Errorf("Unable to list objects, %s", err)
		return releaseListModel, errcode.NewInternalError(fmt.Sprintf("%s", err))
	}
	return releaseListModel, nil
}

// GetReleaseHelmChart returns the release data for the helm chart
func GetReleaseHelmChart(ctx context.Context) (models.ReleaseData, error) {
	resp := models.ReleaseData{
		ID:  *config.Cfg.ReleaseHelmChartVersion,
		URL: fmt.Sprintf(ReleaseHelmChartURLFormat, *config.Cfg.ReleaseHelmChartRepo, *config.Cfg.ReleaseHelmChartVersion),
	}
	return resp, nil
}
