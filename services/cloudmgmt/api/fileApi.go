package api

import (
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/golang/glog"
)

var (
	// MaxUploadBytes is the max upload size cap
	MaxUploadBytes int64 = 1025 * 1024
)

// GetFile gets the content of a file with the path from S3
func (dbAPI *dbObjectModelAPI) GetFile(ctx context.Context, path string, w http.ResponseWriter, r *http.Request) error {
	svc := s3.New(awsSession)
	// Prefix does not work with leading
	path = strings.TrimLeft(path, "/")
	input := &s3.GetObjectInput{
		Bucket: config.Cfg.FileS3Bucket,
		Key:    aws.String(path),
	}
	ifMatch := HeaderValue(r.Header, "If-Match")
	if len(ifMatch) > 0 {
		input.IfMatch = aws.String(ifMatch)
	}
	ifModifiedSince := HeaderValue(r.Header, "If-Modified-Since")
	if len(ifModifiedSince) > 0 {
		t, err := http.ParseTime(ifModifiedSince)
		if err == nil {
			input.IfModifiedSince = aws.Time(t)
		}
	}
	ifNoneMatch := HeaderValue(r.Header, "If-None-Match")
	if len(ifNoneMatch) > 0 {
		input.IfNoneMatch = aws.String(ifNoneMatch)
	}
	ifUnmodifiedSince := HeaderValue(r.Header, "If-Unmodified-Since")
	if len(ifUnmodifiedSince) > 0 {
		t, err := http.ParseTime(ifUnmodifiedSince)
		if err == nil {
			input.IfUnmodifiedSince = aws.Time(t)
		}
	}

	output, err := svc.GetObjectWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "NotModified" {
				w.WriteHeader(http.StatusNotModified)
				return nil
			}
			if aerr.Code() == "NoSuchKey" || aerr.Code() == "NoSuchBucket" {
				w.WriteHeader(http.StatusNotFound)
				return errcode.NewRecordNotFoundError(path)
			}
		}
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get file %s. Error: %s"), path, err.Error())
		return err
	}
	if output.ETag != nil {
		w.Header().Set("ETag", *output.ETag)
	}
	if output.CacheControl != nil {
		w.Header().Set("Cache-Control", *output.CacheControl)
	}
	if output.Body != nil {
		_, err := io.Copy(w, output.Body)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to copy content of the file %s to output stream. Error: %s"), path, err.Error())
			return errcode.NewInternalError(err.Error())
		}
	}
	return nil
}

// ListFiles lists files in S3 for a folder
func (dbAPI *dbObjectModelAPI) ListFiles(ctx context.Context, path string, w http.ResponseWriter, r *http.Request) error {
	svc := s3.New(awsSession)
	files := []string{}
	// Prefix does not work with leading
	path = strings.TrimLeft(path, "/")
	// Delimiter being /, add the trailing / to pick some chars in between
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}
	params := &s3.ListObjectsInput{
		Bucket:    config.Cfg.FileS3Bucket,
		Prefix:    aws.String(path),
		Delimiter: aws.String("/"),
	}
	err := svc.ListObjectsPagesWithContext(ctx, params,
		func(page *s3.ListObjectsOutput, lastPage bool) bool {
			for _, value := range page.Contents {
				files = append(files, *value.Key)
			}
			// continue or  not
			return !lastPage
		})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to list files for path %s. Error: %s"), path, err.Error())
		return err
	}
	return json.NewEncoder(w).Encode(files)
}

// CreateFile creates a file with the path in S3
func (dbAPI *dbObjectModelAPI) CreateFile(ctx context.Context, path string, w http.ResponseWriter, r *http.Request, callback func(context.Context, interface{}) error) error {
	// Prefix does not work with leading
	path = strings.TrimLeft(path, "/")
	if len(path) == 0 || len(path) > 1024 {
		return fmt.Errorf(base.PrefixRequestID(ctx, "Invalid path %s"), path)
	}
	// Only single file for the path in the URI
	file, _, err := r.FormFile("file")
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get file %s from form. Error: %s"), path, err.Error())
		return errcode.NewBadRequestError("file")
	}
	defer file.Close()
	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(awsSession)

	// Upload the file to S3.
	_, err = uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket:       config.Cfg.FileS3Bucket,
		Key:          aws.String(path),
		Body:         base.NewReaderWrapperWithLimit(file, MaxUploadBytes),
		CacheControl: aws.String("Cache-Control:max-age=86400"),
	})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to upload file %s to S3. Error: %s"), path, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return errcode.NewInternalError(err.Error())
	}
	if callback != nil {
		go callback(ctx, path)
	}
	return nil
}

// DeleteFile deletes a file with the path from S3
func (dbAPI *dbObjectModelAPI) DeleteFile(ctx context.Context, path string, w http.ResponseWriter, r *http.Request, callback func(context.Context, interface{}) error) error {
	err := dbAPI.DeleteFiles(ctx, path, true)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete file %s from S3. Error: %s"), path, err.Error())
		return err
	}
	if callback != nil {
		go callback(ctx, path)
	}
	return nil
}

// DeleteFiles deletes files with the option to delete recursively
func (dbAPI *dbObjectModelAPI) DeleteFiles(ctx context.Context, path string, isRecursive bool) error {
	svc := s3.New(awsSession)
	// Prefix does not work with leading
	path = strings.TrimLeft(path, "/")
	// Delimiter being /, add the trailing / to pick some chars in between
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}
	params := &s3.ListObjectsInput{
		Bucket:    config.Cfg.FileS3Bucket,
		Prefix:    aws.String(path),
		Delimiter: aws.String("/"),
	}
	keys := []string{}
	commonPrefixes := []string{}
	err := svc.ListObjectsPagesWithContext(ctx, params,
		func(page *s3.ListObjectsOutput, lastPage bool) bool {
			for _, commonPrefix := range page.CommonPrefixes {
				commonPrefixes = append(commonPrefixes, *commonPrefix.Prefix)
			}
			for _, value := range page.Contents {
				keys = append(keys, *value.Key)
			}
			return lastPage
		})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to list files with path prefix %s from S3. Error: %s"), path, err.Error())
		return err
	}
	errMsgs := []string{}
	for _, key := range keys {
		input := &s3.DeleteObjectInput{
			Bucket: config.Cfg.FileS3Bucket,
			Key:    aws.String(key),
		}
		glog.V(3).Infof(base.PrefixRequestID(ctx, "Deleting %s"), key)
		_, err := svc.DeleteObjectWithContext(ctx, input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() == "NoSuchKey" {
					continue
				}
			}
			errMsgs = append(errMsgs, err.Error())
			glog.Warningf(base.PrefixRequestID(ctx, "Failed to delete file %s from S3. Error: %s"), key, err.Error())
			// Ignore error
		}
	}
	if isRecursive && len(commonPrefixes) > 0 {
		for _, commonPrefix := range commonPrefixes {
			err = dbAPI.DeleteFiles(ctx, commonPrefix, isRecursive)
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					if aerr.Code() == "NoSuchKey" {
						continue
					}
				}
				errMsgs = append(errMsgs, err.Error())
				glog.Warningf(base.PrefixRequestID(ctx, "Failed to delete for prefix %s from S3. Error: %s"), commonPrefix, err.Error())
				// Ignore error
			}
		}
	}
	// Now delete the parent key
	// This could have been deleted as S3 deletes by key (path)
	path = strings.TrimRight(path, "/")
	input := &s3.DeleteObjectInput{
		Bucket: config.Cfg.FileS3Bucket,
		Key:    aws.String(path),
	}
	glog.V(3).Infof(base.PrefixRequestID(ctx, "Deleting %s"), path)
	_, err = svc.DeleteObjectWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != "NoSuchKey" {
			errMsgs = append(errMsgs, err.Error())
			glog.Warningf(base.PrefixRequestID(ctx, "Failed to delete file %s from S3. Error: %s"), path, err.Error())
		}
		// Ignore error
	}
	if len(errMsgs) > 0 {
		err = errcode.NewInternalError(strings.Join(errMsgs, "[n] "))
	}
	return err
}

// PurgeFiles deletes all files with the tenant ID and entity ID
func (dbAPI *dbObjectModelAPI) PurgeFiles(ctx context.Context, tenantID string, id string) error {
	svc := s3.New(awsSession)
	deleteFn := func(baseDir string) error {
		params := &s3.ListObjectsInput{
			Bucket:    config.Cfg.FileS3Bucket,
			Prefix:    aws.String(fmt.Sprintf("%s/%s/", baseDir, tenantID)),
			Delimiter: aws.String(fmt.Sprintf("%s/", id)),
		}
		commonPrefixes := []string{}
		err := svc.ListObjectsPagesWithContext(ctx, params,
			func(page *s3.ListObjectsOutput, lastPage bool) bool {
				for _, commonPrefix := range page.CommonPrefixes {
					commonPrefixes = append(commonPrefixes, *commonPrefix.Prefix)
				}
				return lastPage
			})
		if err != nil {
			return err
		}
		for _, commonPrefix := range commonPrefixes {
			err = dbAPI.DeleteFiles(ctx, commonPrefix, true)
			if err != nil {
				glog.Warningf(base.PrefixRequestID(ctx, "Failed to delete for prefix %s from S3. Error: %s"), commonPrefix, err.Error())
				// Ignore error
			}
		}
		return nil
	}
	errMsgs := []string{}
	err := deleteFn("public")
	if err != nil {
		errMsgs = append(errMsgs, err.Error())
	}
	err = deleteFn("private")
	if err != nil {
		errMsgs = append(errMsgs, err.Error())
	}
	if len(errMsgs) > 0 {
		err = errcode.NewInternalError(strings.Join(errMsgs, "[n] "))
	}
	return err
}

// HeaderValue extracts value of a key from the header
func HeaderValue(header http.Header, key string) string {
	if v := header[key]; len(v) > 0 {
		return v[0]
	}
	return ""
}
