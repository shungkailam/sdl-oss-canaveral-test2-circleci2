package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
)

func getAggregateRoutes(dbAPI api.ObjectModelAPI) []routeHandle {
	return []routeHandle{
		{
			method: "POST",
			path:   "/v1/common/aggregates",
			// swagger:route POST /v1/common/aggregates CommonAggregates
			//
			// Perform an aggregate query. ntnx:ignore
			//
			// Performs and returns the results of the aggregate query.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: CommonAggregatesResponse
			//       default: APIError
			handle: CreateAggregateHandler(dbAPI),
		},
		{
			method: "POST",
			path:   "/v1/common/nestedAggregates",
			// swagger:route POST /v1/common/nestedAggregates CommonNestedAggregates
			//
			// Perform a nested aggregate query. ntnx:ignore
			//
			// Performs and returns the results of the nested aggregate query.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: CommonNestedAggregatesResponse
			//       default: APIError
			handle: CreateNestedAggregateHandler(dbAPI),
		},
	}
}

func getTableName(tpe string) (string, error) {
	switch tpe {
	case "category":
		return "CategoryModel", nil
	case "cloudcreds":
		return "CloudCredsModel", nil
	case "datasource":
		return "DataSourceModel", nil
	case "datastream":
		return "DataStreamModel", nil
	case "edge":
		return "EdgeModel", nil
	case "edgecert":
		return "EdgeCertModel", nil
	case "project":
		return "ProjectModel", nil
	case "script":
		return "ScriptModel", nil
	case "sensor":
		return "SensorModel", nil
	case "tenant":
		return "TenantModel", nil
	case "user":
		return "UserModel", nil

	}
	return "", errors.New("Unknown type")
}

func CreateAggregateHandler(dbAPI api.ObjectModelAPI) httprouter.Handle {
	return getContext(dbAPI, CheckAuth(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		decoder := json.NewDecoder(r.Body)
		doc := model.AggregateSpec{}
		err := decoder.Decode(&doc)
		if err == nil {
			tableName, err := getTableName(doc.Type)
			if err == nil {
				err = dbAPI.GetAggregate(r.Context(), tableName, doc.Field, w)
				if err == nil {
					glog.Infof("[200] Aggregate: tenantID=%s, docType=%s, field=%s\n", ap.TenantID, doc.Type, doc.Field)
					return
				}
			}
		}
		handleResponse(w, r, err, "Aggregate: tenantID=%s, docType=%s, field=%s", ap.TenantID, doc.Type, doc.Field)
	}))
}

func CreateNestedAggregateHandler(dbAPI api.ObjectModelAPI) httprouter.Handle {
	return getContext(dbAPI, CheckAuth(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `[]`)
	}))
}
