package base

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/golang/glog"
	funk "github.com/thoas/go-funk"
)

const (
	// AWS has a limit of 1000
	s3BatchDeleteSize = 990
)

// SyncTarFileIterator is used to upload the contents of a tar file
// to Amazon S3. It implements BatchUploadIterator. This implementation
// is inspired by
// https://github.com/aws/aws-sdk-go/blob/master/example/service/s3/sync/sync.go
// https://medium.com/@skdomino/taring-untaring-files-in-go-6b07cf56bc07
type SyncTarFileIterator struct {
	bucket     string
	prefix     string
	gzipReader *gzip.Reader
	tarReader  *tar.Reader
	header     *tar.Header
	err        error
}

// NewSyncTarFileIterator creates an instance of SyncTarFileIterator
func NewSyncTarFileIterator(ctx context.Context, bucket, prefix string, reader io.Reader) (*SyncTarFileIterator, error) {
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}
	tarReader := tar.NewReader(gzipReader)
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}
	return &SyncTarFileIterator{
		bucket:     bucket,
		prefix:     prefix,
		gzipReader: gzipReader,
		tarReader:  tarReader,
	}, nil
}

// Next will determine whether or not there is any remaining files to
// be uploaded
func (iter *SyncTarFileIterator) Next() bool {
	for {
		iter.header, iter.err = iter.tarReader.Next()
		switch {
		// if no more files are found return
		case iter.err == io.EOF:
			return false
		// return any other error
		case iter.err != nil:
			return false
		// if the header is nil, just skip it (not sure how this happens)
		case iter.header == nil:
			continue
		case iter.header.Typeflag != tar.TypeReg:
			continue
		default:
			tokens := strings.Split(iter.header.Name, "/")
			if strings.HasPrefix(tokens[len(tokens)-1], "._") {
				// Apple qurantine report
				continue
			}
			return true
		}
	}
}

// Err returns any error when os.Open is called.
func (iter *SyncTarFileIterator) Err() error {
	return iter.err
}

// Close frees up resources
func (iter *SyncTarFileIterator) Close() {
	if iter.gzipReader != nil {
		iter.gzipReader.Close()
		iter.gzipReader = nil
	}
}

// UploadObject returns a new upload input for the current file in the iterator
func (iter *SyncTarFileIterator) UploadObject() s3manager.BatchUploadObject {
	bucket := iter.bucket
	prefix := iter.prefix
	key := iter.header.Name
	tarReader := iter.tarReader

	extension := filepath.Ext(key)
	mimeType := mime.TypeByExtension(extension)

	if mimeType == "" {
		mimeType = "binary/octet-stream"
	}

	input := s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(prefix + key),
		Body:        tarReader,
		ContentType: &mimeType,
	}

	glog.Infof("Uploading %+v", key)

	return s3manager.BatchUploadObject{
		Object: &input,
	}
}

// UploadTarContentsToS3 extracts the contents of a tar file (reader) into a S3 prefix as the root folder
func UploadTarContentsToS3(ctx context.Context, awsSession *session.Session, bucket, prefix string, reader io.Reader) error {
	svc := s3manager.NewUploader(awsSession)
	iter, err := NewSyncTarFileIterator(ctx, bucket, prefix, reader)
	if err != nil {
		return err
	}
	defer iter.Close()
	if err := svc.UploadWithIterator(ctx, iter); err != nil {
		return err
	}
	return nil
}

// DeleteS3Objects deletes S3 objects with the given prefix and option to delete recursively
func DeleteS3Objects(ctx context.Context, awsSession *session.Session, bucket, prefix string, isRecursive bool) error {
	svc := s3.New(awsSession)
	// Delimiter being /, add the trailing / to pick some chars in between
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}
	params := &s3.ListObjectsInput{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
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
			// continue or not
			return !lastPage
		})
	if err != nil {
		glog.Errorf(PrefixRequestID(ctx, "Failed to list objects with prefix %s from S3. Error: %s"), prefix, err.Error())
		return err
	}
	objectIDs := funk.Map(keys, func(key string) *s3.ObjectIdentifier {
		return &s3.ObjectIdentifier{Key: aws.String(key)}
	}).([]*s3.ObjectIdentifier)
	err = deleteS3ObjectsInBatches(ctx, awsSession, bucket, objectIDs)
	if err != nil {
		return nil
	}
	if isRecursive && len(commonPrefixes) > 0 {
		for _, commonPrefix := range commonPrefixes {
			err = DeleteS3Objects(ctx, awsSession, bucket, commonPrefix, isRecursive)
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					if aerr.Code() == "NoSuchKey" {
						continue
					}
				}
				return err
			}
		}
	}
	return nil
}

// deleteS3ObjectsInBatches deletes S3 objects in batches
func deleteS3ObjectsInBatches(ctx context.Context, awsSession *session.Session, bucket string, objectIDs []*s3.ObjectIdentifier) error {
	svc := s3.New(awsSession)
	objectIDsLen := len(objectIDs)
	for startIdx := 0; startIdx < objectIDsLen; {
		endSize := startIdx + s3BatchDeleteSize
		if endSize > objectIDsLen {
			endSize = objectIDsLen
		}
		objectIDsInBatch := objectIDs[startIdx:endSize]
		startIdx = startIdx + endSize
		delResult, err := svc.DeleteObjectsWithContext(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &s3.Delete{
				Objects: objectIDsInBatch,
				Quiet:   aws.Bool(false),
			},
		})
		if err != nil {
			glog.Errorf(PrefixRequestID(ctx, "Failed to delete objects from S3 bucket %s. Error: %s"), bucket, err.Error())
			return err
		}
		for _, err := range delResult.Errors {
			if err == nil || err.Code == nil {
				continue
			}
			if *err.Code == "NoSuchKey" {
				continue
			}
			glog.Errorf(PrefixRequestID(ctx, "Failed to delete some objects from S3 bucket %s. Error: %s"), bucket, *err.Code)
			return fmt.Errorf("%s: %s", *err.Code, err.String())
		}
	}
	return nil
}
