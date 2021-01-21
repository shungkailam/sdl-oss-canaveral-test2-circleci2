package pkg

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"context"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/golang/glog"
)

const (
	logEntryQuery         = "select id, location from log_model where updated_at < :updated_before"
	deleteLogEntriesQuery = "delete from log_model where id in (:ids)"
)

var (
	AWSSession *session.Session
	DBAPI      *base.DBObjectModelAPI
)

// LogDBO for select SQL query
type LogDBO struct {
	ID       string `json:"id" db:"id"`
	Location string `json:"location" db:"location"`
}

// LogEntryQueryParam for the log SQL queries
type LogEntryQueryParam struct {
	UpdatedBefore time.Time `json:"updatedBefore" db:"updated_before"`
	IDs           []string  `json:"ids" db:"ids"`
}

// ConnectDB connects to DB and creates DBObjectModelAPI
func ConnectDB() error {
	dbURL, err := base.GetDBURL(*Cfg.SQLDialect, *Cfg.SQLDB, *Cfg.SQLUser, *Cfg.SQLPassword, *Cfg.SQLHost, *Cfg.SQLPort, *Cfg.DisableDBSSL)
	if err != nil {
		glog.Errorf("Failed to get DB URL. Error: %s", err.Error())
		return err
	}
	DBAPI, err = base.NewDBObjectModelAPI(*Cfg.SQLDialect, dbURL, dbURL, nil)
	if err != nil {
		glog.Errorf("Failed to create DB object model API instance. Error: %s", err.Error())
		return err
	}
	return nil
}

// InitAWS creates AWS session
func InitAWS() error {
	var err error
	AWSSession, err = session.NewSession(&aws.Config{
		Region: aws.String(*Cfg.AWSRegion)},
	)
	return err
}

// DeleteLogEntries is the entry method for starting the deletion of log records.
// Rows are fetched in the main go-routine and written to a channel.
// A set of multiple workers read off the channel (Producer-Consumer pattern)
func DeleteLogEntries(ctx context.Context, updatedBefore time.Time, cleaner func(context.Context, time.Time, map[string]*LogDBO) error) error {
	err := ConnectDB()
	if err != nil {
		return err
	}
	defer DBAPI.Close()
	err = InitAWS()
	if err != nil {
		return err
	}
	count := 0
	totalCount := 0
	pageSize := *Cfg.PageSize
	chanSize := pageSize * 2
	inputQueue := make(chan *LogDBO, chanSize)
	// Create the workers to delete from S3 and DB
	wg := createWorkers(ctx, updatedBefore, pageSize, inputQueue, cleaner)
	nextPageToken := base.StartPageToken
	for now := range time.Tick(time.Second) {
		glog.Infof("Checking at %+v", now)
		if nextPageToken == base.NilPageToken {
			if len(inputQueue) == 0 {
				glog.Infof("Closing input queue. Total processed %d", totalCount)
				close(inputQueue)
				break
			}
			continue
		}
		// Fetch may return the same next token when it cannot find enough free slots in the channel
		nextPageToken, count, err = fetchRows(ctx, updatedBefore, nextPageToken, pageSize, inputQueue)
		if err != nil {
			glog.Errorf("Failed to fetch rows. Error: %s", err.Error())
		} else {
			totalCount += count
		}
	}
	glog.Infof("Waiting for workers to shut down")
	wg.Wait()
	return nil
}

// DeleteVersionedS3Objects deletes all the S3 object versions for the keys in LogDBOs
func DeleteVersionedS3Objects(ctx context.Context, updatedBefore time.Time, logDBOs map[string]*LogDBO) error {
	svc := s3.New(AWSSession)
	objectIDs := []*s3.ObjectIdentifier{}
	var nextKeyMarker *string
	clonedLogDBOs := make(map[string]*LogDBO)
	for key, value := range logDBOs {
		clonedLogDBOs[key] = value
	}
	for {
		versionsOutput, err := svc.ListObjectVersionsWithContext(ctx, &s3.ListObjectVersionsInput{
			Bucket:    Cfg.S3Bucket,
			KeyMarker: nextKeyMarker,
		})
		if err != nil {
			glog.Errorf("Failed to list object versions. Error: %s", err.Error())
			return err
		}
		for _, deleteMarker := range versionsOutput.DeleteMarkers {
			key := deleteMarker.Key
			if key == nil {
				continue
			}
			if _, ok := clonedLogDBOs[*key]; !ok {
				if deleteMarker.IsLatest == nil || !*deleteMarker.IsLatest {
					continue
				}
				if deleteMarker.LastModified == nil || updatedBefore.Before(*deleteMarker.LastModified) {
					// Delete if it is too old and already deleted
					continue
				}
				// Add so that all the versions can be deleted
				clonedLogDBOs[*key] = &LogDBO{Location: *key}
			}
			glog.Infof("Deleting key: %s, version: %s", *key, *deleteMarker.VersionId)
			objectIDs = append(objectIDs, &s3.ObjectIdentifier{Key: key, VersionId: deleteMarker.VersionId})
		}
		for _, version := range versionsOutput.Versions {
			key := version.Key
			if key == nil {
				continue
			}
			if _, ok := clonedLogDBOs[*key]; !ok {
				continue
			}
			glog.Infof("Deleting key: %s, version: %s", *key, *version.VersionId)
			objectIDs = append(objectIDs, &s3.ObjectIdentifier{Key: key, VersionId: version.VersionId})
		}
		nextKeyMarker = versionsOutput.NextKeyMarker
		if nextKeyMarker == nil {
			break
		}
	}
	err := DeleteObjectsInBatches(ctx, objectIDs)
	if err != nil {
		glog.Errorf("Failed to delete S3 versioned objects. Error: %s", err.Error())
	}
	return err
}

// DeleteS3Objects deletes S3 objects for the keys in LogDBOs
func DeleteS3Objects(ctx context.Context, updatedBefore time.Time, logDBOs map[string]*LogDBO) error {
	objectIDs := []*s3.ObjectIdentifier{}
	for _, logDBO := range logDBOs {
		glog.Infof("Deleting key: %s", logDBO.Location)
		objectIDs = append(objectIDs, &s3.ObjectIdentifier{Key: aws.String(logDBO.Location)})
	}
	err := DeleteObjectsInBatches(ctx, objectIDs)
	if err != nil {
		glog.Errorf("Failed to delete S3 objects. Error: %s", err.Error())
	}
	return err
}

// DeleteDBRecords deletes the DB records
func DeleteDBRecords(ctx context.Context, updatedBefore time.Time, logDBOs map[string]*LogDBO) error {
	ids := []string{}
	for _, logDBO := range logDBOs {
		ids = append(ids, logDBO.ID)
	}
	_, err := DBAPI.DeleteIn(ctx, deleteLogEntriesQuery, LogEntryQueryParam{IDs: ids})
	if err != nil {
		glog.Errorf("Failed to delete log entries from DB. Error: %s", err.Error())
	}
	return err
}

// DeleteRecords deletes all the records from DB and S3
// Retry will be taken by the cron job
func DeleteRecords(ctx context.Context, updatedBefore time.Time, logDBOs map[string]*LogDBO) error {
	var err error
	if *Cfg.S3VersionEnabled {
		err = DeleteVersionedS3Objects(ctx, updatedBefore, logDBOs)
	} else {
		err = DeleteS3Objects(ctx, updatedBefore, logDBOs)
	}
	if err != nil {
		return err
	}
	err = DeleteDBRecords(ctx, updatedBefore, logDBOs)
	return err
}

// createWorkers creates workers to work on the entries fetched from DB.
// It returns the wait group to wait for workers to shutdown and exit gracefully
func createWorkers(ctx context.Context, updatedBefore time.Time, batchSize int, inputQueue <-chan *LogDBO, cleaner func(context.Context, time.Time, map[string]*LogDBO) error) *sync.WaitGroup {
	var wg sync.WaitGroup
	for i := 0; i < *Cfg.WorkerCount; i++ {
		wg.Add(1)
		go func(ctx context.Context) {
			defer wg.Done()
			logDBOs := map[string]*LogDBO{}
		loop:
			for {
				select {
				case logDBO, ok := <-inputQueue:
					if ok {
						logDBOs[logDBO.Location] = logDBO
						if len(logDBOs) >= batchSize {
							err := cleaner(ctx, updatedBefore, logDBOs)
							if err != nil {
								glog.Errorf("Failed to delete log entries. Error: %s", err.Error())
							}
							logDBOs = map[string]*LogDBO{}
						}
					} else {
						if len(logDBOs) > 0 {
							err := cleaner(ctx, updatedBefore, logDBOs)
							if err != nil {
								glog.Errorf("Failed to delete log entries. Error: %s", err.Error())
							}
						}
						break loop
					}
				}
			}
			glog.Infof("Worker exiting...")
		}(ctx)
	}
	return &wg
}

// fetchRows fetches log records from DB in small pages.
// It fetches when the queue/channel has enough room to fill all the records in the page.
func fetchRows(ctx context.Context, updatedBefore time.Time, nextPageToken base.PageToken, pageSize int, logDBOs chan<- *LogDBO) (base.PageToken, int, error) {
	var err error
	count := 0
	freeSize := cap(logDBOs) - len(logDBOs)
	if freeSize >= pageSize {
		glog.Infof("Fetching log records as freeSize(%d) >= pageSize(%d)", freeSize, pageSize)
		queryParam := LogEntryQueryParam{UpdatedBefore: updatedBefore}
		nextPageToken, err = DBAPI.PagedQueryEx(ctx, nextPageToken, pageSize, func(dbObjPtr interface{}) error {
			logDBO := dbObjPtr.(*LogDBO)
			logDBOs <- logDBO
			count++
			return nil
		}, logEntryQuery, queryParam, LogDBO{})
	}
	return nextPageToken, count, err
}

// DeleteObjectsInBatches deletes S3 objects in batches
func DeleteObjectsInBatches(ctx context.Context, objectIDs []*s3.ObjectIdentifier) error {
	svc := s3.New(AWSSession)
	errMsgs := []string{}
	// AWS has a limit of 1000
	batchSize := 990
	objectIDsLen := len(objectIDs)
	for startIdx := 0; startIdx < objectIDsLen; {
		endSize := startIdx + batchSize
		if endSize > objectIDsLen {
			endSize = objectIDsLen
		}
		batch := objectIDs[startIdx:endSize]
		startIdx = startIdx + endSize
		glog.Infof("Deleting %d objects", len(batch))
		delResult, err := svc.DeleteObjectsWithContext(ctx, &s3.DeleteObjectsInput{
			Bucket: Cfg.S3Bucket,
			Delete: &s3.Delete{
				Objects: objectIDs,
				Quiet:   aws.Bool(false),
			},
		})
		if err == nil {
			for _, delErr := range delResult.Errors {
				if delErr == nil || delErr.Code == nil {
					continue
				}
				if *delErr.Code == "NoSuchKey" {
					continue
				}
				errMsgs = append(errMsgs, *delErr.Code)
				glog.Errorf("Failed to delete some objects from S3. Error: %s", *delErr.Code)
			}
		} else {
			errMsgs = append(errMsgs, err.Error())
			glog.Errorf("Failed to delete objects from S3. Error: %s", err.Error())
		}
	}
	var err error
	if len(errMsgs) > 0 {
		err = errcode.NewInternalError(strings.Join(errMsgs, "[n] "))
	}
	return err
}
