package pkg

import (
	"cloudservices/common/base"
	"context"
	"time"

	"github.com/golang/glog"
)

const (
	// Delete previous batches which are older than a specified updated time
	// deleteExpiredBatchesQuery  = `delete from software_update_batch_model where updated_at < :updated_at and id not in (select distinct on(svc_domain_id) batch_id from software_update_service_domain_model order by svc_domain_id, created_at desc)`

	// Delete previous batches which are older than a specified update time and total number of batches crosses a max number of batches.
	// Partition the rows by svc_domain_id to increment the count within each svc_domain_id and select the count > the max batches
	deleteExpiredBatchesQuery = `delete from software_update_batch_model where id in (select s.batch_id from (select svc_domain_id, batch_id, row_number() over (partition by svc_domain_id) as rno from software_update_service_domain_model where updated_at < :updated_at order by svc_domain_id, created_at desc) s where s.rno > :max_batches)`
)

var (
	DBAPI *base.DBObjectModelAPI
)

// BatchQueryParam is the query DBO
type BatchQueryParam struct {
	UpdatedAt  time.Time `json:"updatedAt" db:"updated_at"`
	MaxBatches int       `json:"maxBatches" db:"max_batches"`
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

// DeleteExpiredBatches deletes the expired batches which are previous batches with updated timestamp lesser than a checkpoint time
func DeleteExpiredBatches(ctx context.Context) (int64, error) {
	err := ConnectDB()
	if err != nil {
		return 0, err
	}
	defer DBAPI.Close()
	checkpointTime := base.RoundedNow().Add(-time.Duration(*Cfg.ElapsedDays))
	maxBatches := *Cfg.MaximumBatches
	if maxBatches <= 0 {
		// Keep at least one if it is zero
		maxBatches = 1
	}
	result, err := DBAPI.NamedExec(ctx, deleteExpiredBatchesQuery, &BatchQueryParam{UpdatedAt: checkpointTime, MaxBatches: maxBatches})
	if err != nil {
		glog.Errorf("Failed to delete expired software update batches. Error: %s", err.Error())
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		glog.Errorf("Failed to get number of deleted batches. Error: %s", err.Error())
		return 0, err
	}
	glog.Infof("Deleted %d expired batches", rows)
	return rows, nil
}
