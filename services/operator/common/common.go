package common

import (
	"cloudservices/operator/config"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	awscredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

var sessionLoader sync.Once

var awsSession *session.Session
var awsSessionOTA *session.Session

func isMinio() bool {
	return *config.Cfg.ObjectStorageEngine == "minio"
}

func loadAWSSessions() {
	sessionLoader.Do(func() {
		var err error
		if isMinio() {
			awsSession, err = session.NewSession(&aws.Config{
				Credentials:      credentials.NewStaticCredentials(*config.Cfg.MinioAccessKey, *config.Cfg.MinioSecretKey, ""),
				Endpoint:         aws.String(*config.Cfg.MinioURL),
				Region:           aws.String("us-west-2"),
				DisableSSL:       aws.Bool(true),
				S3ForcePathStyle: aws.Bool(true),
			})
			awsSessionOTA = awsSession
		} else {
			// Create AWS session and AWS S3 client
			awsSession, err = session.NewSession(&aws.Config{
				Region: aws.String(*config.Cfg.AWSRegion)},
			)
			if err == nil {
				awsSessionOTA, err = session.NewSession(&aws.Config{
					Region:      aws.String(*config.Cfg.AWSRegion),
					Credentials: awscredentials.NewStaticCredentials(*config.Cfg.OTAAccessKey, *config.Cfg.OTASecretKey, ""),
				})
			}
		}
		if err != nil {
			panic(err)
		}
	})
}

// GetAWSSession returns the AWS session
func GetAWSSession() *session.Session {
	loadAWSSessions()
	return awsSession
}

// GetOTAAWSSession returns the OTA AWS session
func GetOTAAWSSession() *session.Session {
	loadAWSSessions()
	return awsSessionOTA
}
