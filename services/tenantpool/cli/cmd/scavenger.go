package cmd

import (
	"cloudservices/common/base"
	"cloudservices/tenantpool/config"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
)

const (
	checkColumnQuery          = `SELECT count(column_name) as count FROM information_schema.columns WHERE table_name = :table and column_name = :column`
	deleteEntityQueryTemplate = `DELETE FROM %s e USING tenant_model t WHERE e.%s = t.id AND t.name='Trial Tenant' AND t.external_id is null AND (:tenant_id = '' OR t.id = :tenant_id) AND (SELECT count(*) FROM user_model where tenant_id = t.id AND email not like '%%.ntnx-del') = 0 AND t.updated_at < :update_before`
)

type CheckColumnQuery struct {
	Table  string `json:"table" db:"table"`
	Column string `json:"column" db:"column"`
}

type DeleteEntityQueryParam struct {
	TenantID      string    `json:"tenantId" db:"tenant_id"`
	UpdatedBefore time.Time `json:"updatedBefore" db:"update_before"`
}

type Scavenger struct {
	*base.DBObjectModelAPI
}

func init() {
	sqlHost := strings.TrimSpace(os.Getenv("SQL_HOST"))
	if len(sqlHost) > 0 {
		*config.Cfg.SQLHost = sqlHost
	}
	sqlPort := strings.TrimSpace(os.Getenv("SQL_PORT"))
	if len(sqlPort) > 0 {
		port, err := strconv.Atoi(sqlPort)
		if err != nil {
			*config.Cfg.SQLPort = port
		}
	}
	sqlDB := strings.TrimSpace(os.Getenv("SQL_DB"))
	if len(sqlDB) > 0 {
		*config.Cfg.SQLDB = sqlDB
	}
	sqlPassword := strings.TrimSpace(os.Getenv("SQL_PASSWORD"))
	if len(sqlPassword) > 0 {
		*config.Cfg.SQLPassword = sqlPassword
	}
}

// NewScavenger instantiates Scavenger
func NewScavenger() *Scavenger {
	dbURL, err := base.GetDBURL(*config.Cfg.SQLDialect, *config.Cfg.SQLDB, *config.Cfg.SQLUser, *config.Cfg.SQLPassword, *config.Cfg.SQLHost, *config.Cfg.SQLPort, false)
	if err != nil {
		glog.Errorf("Failed to create book keeper instance. Error: %s", err.Error())
		panic(err)
	}
	dbAPI, err := base.NewDBObjectModelAPI(*config.Cfg.SQLDialect, dbURL, dbURL, nil)
	if err != nil {
		glog.Errorf("Failed to create db object model API instance. Error: %s", err.Error())
		panic(err)
	}
	return &Scavenger{DBObjectModelAPI: dbAPI}
}

// GenerateDeleteEntityQueries generates the delete queries starting from the independent tables
func (scavenger *Scavenger) GenerateDeleteEntityQueries(ctx context.Context, dependencies map[string]map[string]bool, callback func(table, query string) error) error {
	return base.TraverseDependencies(dependencies, func(deletableTables map[string]bool) error {
		for table := range deletableTables {
			var deleteEntityQuery string
			if table == "tenant_model" {
				deleteEntityQuery = fmt.Sprintf(deleteEntityQueryTemplate, table, "id")
			} else {
				rows, err := scavenger.NamedQuery(ctx, checkColumnQuery, CheckColumnQuery{Table: table, Column: "tenant_id"})
				if err != nil {
					glog.Errorf(base.PrefixRequestID(ctx, "Failed to execute query %s. Error: %s"), checkColumnQuery, err.Error())
					return err
				}
				columnCount := 0
				if rows.Next() {
					err = rows.Scan(&columnCount)
					if err != nil {
						glog.Errorf(base.PrefixRequestID(ctx, "Failed to execute query %s. Error: %s"), checkColumnQuery, err.Error())
						return err
					}
				}
				if columnCount > 0 {
					deleteEntityQuery = fmt.Sprintf(deleteEntityQueryTemplate, table, "tenant_id")
				}
			}
			if len(deleteEntityQuery) > 0 {
				// Retains the order
				err := callback(table, deleteEntityQuery)
				if err != nil {
					glog.Errorf(base.PrefixRequestID(ctx, "Failed to invoke callback with query %s. Error: %s"), deleteEntityQuery, err.Error())
					return err
				}
			}
		}
		return nil
	})
}

// Run is the entry to start the deletion
func (scavenger *Scavenger) Run(ctx context.Context, isDryRun bool, updatedBefore time.Duration, tenantID string) (int64, error) {
	dependencies, err := scavenger.GetTableDependencies(ctx)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get foreign key dependencies. Error: %s"), err.Error())
		return 0, nil
	}
	var tenantsAffected int64
	param := DeleteEntityQueryParam{UpdatedBefore: time.Now().Add(-updatedBefore), TenantID: tenantID}
	err = scavenger.DoInTxn(func(tx *base.WrappedTx) error {
		err := scavenger.GenerateDeleteEntityQueries(ctx, dependencies, func(table, deleteEntityQuery string) error {
			glog.Infof(base.PrefixRequestID(ctx, "Running query: %s"), deleteEntityQuery)
			result, err := tx.NamedExec(ctx, deleteEntityQuery, param)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to execute query %s. Error: %s"), deleteEntityQuery, err.Error())
				return err
			}
			if table == "tenant_model" {
				rowsAffected, err := result.RowsAffected()
				if err != nil {
					glog.Errorf(base.PrefixRequestID(ctx, "Failed to get affected rows for query %s. Error: %s"), deleteEntityQuery, err.Error())
					return err
				}
				tenantsAffected += rowsAffected
			}
			return nil
		})
		if err != nil {
			return err
		}
		if isDryRun {
			return base.ErrForceRollback
		}
		return nil
	})
	return tenantsAffected, err
}
