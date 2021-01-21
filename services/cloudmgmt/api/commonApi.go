package api

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"fmt"
	"io"

	"github.com/iancoleman/strcase"
)

func init() {
	queryMap["GetAggregate"] = `SELECT "%s" AS "key", COUNT("id") AS "doc_count" FROM "%s" WHERE "%s".tenant_id = '%s' GROUP BY "key"`
}

// GetAggregate perform aggregate query on the selected field
func (dbAPI *dbObjectModelAPI) GetAggregate(context context.Context, tableName string, fieldName string, w io.Writer) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	tenantID := authContext.TenantID
	db := dbAPI.GetDB()
	// TODO: Temporary the table hes not been changed in the UI request
	snakeTableName := strcase.ToSnake(tableName)
	snakeFieldName := strcase.ToSnake(fieldName)
	query := fmt.Sprintf(queryMap["GetAggregate"], snakeFieldName, snakeTableName, snakeTableName, tenantID)
	rows, err := db.Queryx(query)
	if err != nil {
		return errcode.TranslateDatabaseError(fieldName, err)
	}
	defer rows.Close()
	results := []model.AggregateInfo{}
	for rows.Next() {
		p := model.AggregateInfo{}
		err = rows.StructScan(&p)
		if err != nil {
			return errcode.TranslateDatabaseError(fieldName, err)
		}
		results = append(results, p)
	}
	return base.DispatchPayload(w, results)
}
