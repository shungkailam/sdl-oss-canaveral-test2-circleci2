package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/golang/glog"
)

// S3UploadIterator implements BatchUploadIterator
type S3UploadIterator struct {
	ctx       context.Context
	ch        <-chan *Record
	s3Bucket  *string
	s3Prefix  *string
	s3Object  *s3manager.BatchUploadObject
	createdAt time.Time
	index     int
}

// NewS3UploadIterator returns an iterator which implements s3manager.BatchUploadIterator of AWS
func NewS3UploadIterator(ctx context.Context, createdAt time.Time, s3Bucket, s3Prefix *string, ch <-chan *Record) *S3UploadIterator {
	iterator := &S3UploadIterator{ctx: ctx, s3Bucket: s3Bucket, s3Prefix: s3Prefix, ch: ch, createdAt: createdAt}
	return iterator
}

// Next implements s3manager.BatchUploadIterator of AWS
func (iterator *S3UploadIterator) Next() bool {
	// Loop also checks if the channel is closed
	for record := range iterator.ch {
		columnValues := map[string]interface{}{}
		for i := range record.Columns {
			column := record.Columns[i]
			columnValues[column.ColumnName] = column.Value
		}
		content, err := json.Marshal(columnValues)
		if err != nil {
			continue
		}
		id, ok := columnValues["id"].(string)
		if !ok {
			glog.Warningf("Missing ID for record %+v", columnValues)
			continue
		}
		s3Key := iterator.getS3ObjectKey(record.Name, id)
		if glog.V(4) {
			glog.V(4).Infof("S3 content for key %s\n%+v", s3Key, string(content))
		}
		iterator.s3Object = &s3manager.BatchUploadObject{
			Object: &s3manager.UploadInput{
				Body:   bytes.NewBuffer(content),
				Bucket: iterator.s3Bucket,
				Key:    aws.String(s3Key),
			},
		}
		return true
	}
	return false
}

// Err implements s3manager.BatchUploadIterator of AWS
func (iterator *S3UploadIterator) Err() error {
	return nil
}

// UploadObject implements s3manager.BatchUploadIterator of AWS
func (iterator *S3UploadIterator) UploadObject() s3manager.BatchUploadObject {
	if iterator.s3Object == nil {
		return s3manager.BatchUploadObject{}
	}
	return *iterator.s3Object
}

func (iterator *S3UploadIterator) getS3ObjectKey(recordName string, suffix string) string {
	return fmt.Sprintf("%s/%s/%s/%s_%s.txt",
		*iterator.s3Prefix, recordName, getDatePathComponent(iterator.createdAt), recordName, suffix)
}

func getDatePathComponent(t time.Time) string {
	return fmt.Sprintf("year=%d/month=%02d/day=%02d", t.Year(), t.Month(), t.Day())
}
