package main

import (
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/errcode"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/golang/glog"
)

func uploadToS3(file io.Reader, s3Key string) error {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(*config.Cfg.AWSRegion)},
	)
	if err != nil {
		glog.Errorf("Error connecting to s3, %s", err)
		return err
	}
	// To do return md5 sum
	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(sess)
	bucketName := "sherlock-code-coverage"
	// Upload code coverage files
	upParams := &s3manager.UploadInput{
		Bucket: &bucketName,
		Key:    &s3Key,
		Body:   file,
	}
	_, err = uploader.Upload(upParams)
	if err != nil {
		glog.Errorf("Unable to upload file to s3, %s", err)
		err = errcode.NewInternalError(fmt.Sprintf("%s", err))
		return err
	}
	return nil
}

func main() {
	filePath := flag.String("filePath", "/tmp/cloudmgmt.cov", "path to file.")
	flag.Parse()

	fmt.Println("Upload code coverage file to S3.")
	file, err := os.Open(*filePath)
	if err != nil {
		glog.Fatalf("Failed to open coverage file: %s", err)
	}
	ts := time.Now().Format(time.RFC3339)
	dnsTs := strings.Replace(ts, ":", "-", -1)
	s3Key := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name())) + "-" + dnsTs + ".html"
	err = uploadToS3(file, s3Key)
	if err != nil {
		glog.Fatalf("Unable to upload coverage file to s3, %s", err)
	}
}
