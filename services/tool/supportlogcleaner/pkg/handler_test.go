package pkg_test

import (
	"cloudservices/common/base"
	"cloudservices/tool/supportlogcleaner/pkg"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func TestSupportLogCleaner(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	err := pkg.ConnectDB()
	require.NoError(t, err)
	// Before 3 days
	updatedBefore := time.Now().Add(-time.Hour * 24 * 3)
	actualRecordCount := getRecordCount(t, pkg.DBAPI, updatedBefore)
	pkg.DBAPI.Close()
	t.Logf("Actual record count: %d", actualRecordCount)
	recordCount := 0
	err = pkg.DeleteLogEntries(ctx, updatedBefore, func(ctx context.Context, updatedBefore time.Time, logDBOs map[string]*pkg.LogDBO) error {
		recordCount += len(logDBOs)
		return nil
	})
	require.NoError(t, err)
	if recordCount != actualRecordCount {
		t.Fatalf("Expected %d records, found %d records", actualRecordCount, recordCount)
	}
}

func TestDeleteVersionedS3Objects(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	err := pkg.InitAWS()
	require.NoError(t, err)
	updatedBefore := time.Now().Add(-time.Hour * 24 * 15)
	uuid := base.GetUUID()
	s3ObjectKey := fmt.Sprintf("v1-test/%s/test1.txt", uuid)
	s3ObjectKeyPrefix := fmt.Sprintf("v1-test/%s/", uuid)
	svc := s3.New(pkg.AWSSession)
	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket: pkg.Cfg.S3Bucket,
		Key:    aws.String(s3ObjectKey),
		Body:   strings.NewReader("test"),
	})
	require.NoError(t, err)
	_, err = svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: pkg.Cfg.S3Bucket,
		Key:    aws.String(s3ObjectKey),
	})
	require.NoError(t, err)
	listObjectVersionsInput := &s3.ListObjectVersionsInput{
		Bucket:  pkg.Cfg.S3Bucket,
		MaxKeys: aws.Int64(100),
		Prefix:  aws.String(s3ObjectKeyPrefix),
	}
	versionsOutput, err := svc.ListObjectVersions(listObjectVersionsInput)
	require.NoError(t, err)
	if len(versionsOutput.DeleteMarkers) != 1 {
		t.Fatalf("Expected 1 delete marker, found %d", len(versionsOutput.DeleteMarkers))
	}
	if len(versionsOutput.Versions) != 1 {
		t.Fatalf("Expected 1 version, found %d", len(versionsOutput.Versions))
	}
	err = pkg.DeleteVersionedS3Objects(ctx, updatedBefore, map[string]*pkg.LogDBO{
		s3ObjectKey: {
			Location: s3ObjectKey,
		},
	})
	require.NoError(t, err)
	versionsOutput, err = svc.ListObjectVersions(listObjectVersionsInput)
	require.NoError(t, err)
	if len(versionsOutput.DeleteMarkers) != 0 {
		t.Fatalf("Expected 0 delete marker, found %d, %+v", len(versionsOutput.DeleteMarkers), versionsOutput)
	}
	if len(versionsOutput.Versions) != 0 {
		t.Fatalf("Expected 0 version, found %d, %+v", len(versionsOutput.Versions), versionsOutput)
	}
}

func getRecordCount(t *testing.T, dbAPI *base.DBObjectModelAPI, updatedBefore time.Time) int {
	rows, err := dbAPI.NamedQuery(context.Background(), "select count(*) from log_model where updated_at < :updated_before", pkg.LogEntryQueryParam{UpdatedBefore: updatedBefore})
	require.NoError(t, err)
	count := 0
	if rows.Next() {
		rows.Scan(&count)
	}
	return count
}
