package base_test

import (
	"cloudservices/common/base"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

func compareFileContents(t *testing.T, first string, second string) {
	firstData, err := ioutil.ReadFile(first)
	require.NoError(t, err)
	secondData, err := ioutil.ReadFile(second)
	require.NoError(t, err)
	if !reflect.DeepEqual(firstData, secondData) {
		t.Fatalf("contents mismatch:\n%s -->%s\n%s -->%s\n", first, string(firstData), second, string(secondData))
	}
}

func IsEmptyDir(name string) (bool, error) {
	isEmpty := true
	err := filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !isEmpty {
			return nil
		}
		if !info.IsDir() {
			isEmpty = false
		}
		return nil
	})
	return isEmpty, err
}

func TestUploadTarContentsToS3(t *testing.T) {
	ctx := context.Background()
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	require.NoError(t, err)
	_, filename, _, _ := runtime.Caller(0)
	currentDir := path.Dir(filename)
	s3Bucket := "khogen-upgrade-test"
	s3Prefix := base.GetUUID()
	reader, err := os.Open(fmt.Sprintf("%s/testdata/test.tgz", currentDir))
	require.NoError(t, err)

	defer reader.Close()
	err = base.UploadTarContentsToS3(ctx, awsSession, s3Bucket, s3Prefix, reader)
	require.NoError(t, err)
	s3ObjectPath := fmt.Sprintf("s3://%s/%s", s3Bucket, s3Prefix)
	downloadDir := fmt.Sprintf("/tmp/%s", s3Prefix)
	cmd := exec.Command("sh", "-c", fmt.Sprintf("mkdir -p %s", downloadDir))
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	// Clean up
	defer exec.Command("sh", "-c", fmt.Sprintf("rm -rf %s", downloadDir)).Run()
	cmd = exec.Command("sh", "-c", fmt.Sprintf("aws s3 sync %s %s", s3ObjectPath, downloadDir))
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	compareFileContents(t, fmt.Sprintf("%s/f1/first.txt", downloadDir), fmt.Sprintf("%s/testdata/f1/first.txt", currentDir))
	compareFileContents(t, fmt.Sprintf("%s/second.txt", downloadDir), fmt.Sprintf("%s/testdata/second.txt", currentDir))
	err = base.DeleteS3Objects(ctx, awsSession, s3Bucket, s3Prefix, true)
	require.NoError(t, err)
	cmd = exec.Command("sh", "-c", fmt.Sprintf("aws s3 sync --delete %s %s", s3ObjectPath, downloadDir))
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	isEmpty, err := IsEmptyDir(downloadDir)
	require.NoError(t, err)
	if !isEmpty {
		t.Fatalf("directory %s must be empty after the sync", downloadDir)
	}
}
