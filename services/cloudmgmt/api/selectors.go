package api

import (
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"

	"github.com/golang/glog"

	"fmt"
	"strings"
)

func init() {
	queryMap["CreateDataStreamOriginSelector"] = `INSERT INTO data_stream_origin_model (data_stream_id, category_value_id) VALUES (:data_stream_id, :category_value_id)`
	queryMap["CreateApplicationOriginSelector"] = `INSERT INTO application_origin_model (application_id, category_value_id) VALUES (:application_id, :category_value_id)`
}

// BaseOriginSelectorDBO is the base model for origin selectors
type BaseOriginSelectorDBO struct {
	model.CategoryInfo `json:"categoryInfo" db:"category_info"`
	ID                 int64 `json:"id" db:"id"`
	CategoryValueID    int64 `json:"categoryValueId" db:"category_value_id"`
}

// DataStreamOriginSelectorDBO is the DB object model for Origin selectors of  data streams
type DataStreamOriginSelectorDBO struct {
	BaseOriginSelectorDBO
	DataStreamID string `json:"dataStreamId" db:"data_stream_id"`
}

// ApplicationOriginSelectorDBO is the DB model for origin selector of applications
type ApplicationOriginSelectorDBO struct {
	BaseOriginSelectorDBO
	ApplicationID string `json:"applictionId" db:"application_id"`
}

// createOriginSelectors creates selectors for the given categories and entity
func (dbAPI *dbObjectModelAPI) createOriginSelectors(ctx context.Context, tx *base.WrappedTx, categories []model.CategoryInfo, entityType, entityID string) error {
	if !*config.Cfg.EnableAppOriginSelectors && entityType == entityTypeApplication || len(categories) == 0 {
		return nil
	}

	catIDValuePairs, dbCatIDValuePairs := make(map[string]bool), make(map[string]bool)
	IDs := make([]string, 0, len(categories))
	for _, sel := range categories {
		IDs = append(IDs, sel.ID)
		catIDValuePairs[fmt.Sprintf("%s:%s", sel.ID, sel.Value)] = true
	}

	categoryValueDBOs, err := dbAPI.getCategoryValueDBOsByCategoryIds(ctx, IDs)
	if err != nil {
		return err
	}

	if len(categoryValueDBOs) == 0 {
		glog.Errorf("no category values selected for IDs: %s", strings.Join(IDs, ","))
		return errcode.NewRecordNotFoundError(strings.Join(IDs, ","))
	}

	// validate that all category values exist
	for _, catValue := range categoryValueDBOs {
		dbCatIDValuePairs[fmt.Sprintf("%s:%s", catValue.CategoryID, catValue.Value)] = true
	}

	for catValPair := range catIDValuePairs {
		if !dbCatIDValuePairs[catValPair] {
			return errcode.NewRecordNotFoundError(fmt.Sprintf(catValPair))
		}
	}

	for _, catValue := range categoryValueDBOs {
		if !catIDValuePairs[fmt.Sprintf("%s:%s", catValue.CategoryID, catValue.Value)] {
			glog.V(5).Infof("skipping %s:%s as it is not in the requested category value pairs", catValue.CategoryID, catValue.Value)
			continue
		}
		var err error
		switch entityType {
		case entityTypeDataStream:
			dataStreamOriginSelector := DataStreamOriginSelectorDBO{DataStreamID: entityID}
			dataStreamOriginSelector.CategoryValueID = catValue.ID
			_, err = tx.NamedExec(ctx, queryMap["CreateDataStreamOriginSelector"], &dataStreamOriginSelector)
		case entityTypeApplication:
			appOriginSelector := ApplicationOriginSelectorDBO{ApplicationID: entityID}
			appOriginSelector.CategoryValueID = catValue.ID
			_, err = tx.NamedExec(ctx, queryMap["CreateApplicationOriginSelector"], &appOriginSelector)
		default:
			return fmt.Errorf("unsupported entity type for origin selectors: %s", entityType)
		}
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx,
				"Error occurred while creating origin selectors for %s:%s . Error: %s"),
				entityType, entityID, err.Error(),
			)
			return errcode.TranslateDatabaseError(entityID, err)
		}
		glog.V(5).Infof("successfully created origin selector: %+v for %s %s", catValue, entityType, entityID)
	}

	return nil
}
