package api

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"

	"context"
	"fmt"

	"github.com/golang/glog"
)

func init() {
	queryMap["SelectDataIfcClaimByTopic"] = `SELECT data_source_id, data_stream_id, tenant_id, topic FROM data_source_topic_claim WHERE tenant_id = :tenant_id AND data_source_id = :data_source_id AND topic = :topic`
	queryMap["SelectDataIfcClaimByDataSourceID"] = `SELECT data_source_id, data_stream_id, tenant_id, topic FROM data_source_topic_claim WHERE tenant_id = :tenant_id AND data_source_id = :data_source_id`
	queryMap["CreateDataIfcClaim"] = `INSERT INTO  data_source_topic_claim(data_source_id, data_stream_id, application_id, topic, tenant_id) VALUES (:data_source_id, :data_stream_id, :application_id, :topic, :tenant_id)`
}

// DataIfcTopicClaimDBO encapsulates the topic claim from a data stream or application to a given data source
// In our current impl, we allow only one out data Ifc to be claimed by a data stream
type DataIfcTopicClaimDBO struct {
	DataSourceID  string  `json:"dataSourceID" db:"data_source_id"`
	DataStreamID  *string `json:"dataStreamID" db:"data_stream_id"`
	ApplicationID *string `json:"applicationID" db:"application_id"`
	Topic         string  `json:"topic" db:"topic"`
	TenantID      string  `json:"tenantID" db:"tenant_id"`
}

// insertDataIfcClaim inserts a data source claim object in DB
func (dbAPI *dbObjectModelAPI) insertDataIfcClaim(ctx context.Context, tx *base.WrappedTx, claim *DataIfcTopicClaimDBO) error {
	_, err := tx.NamedExec(ctx, queryMap["CreateDataIfcClaim"], claim)
	if err != nil {
		return fmt.Errorf("failed to create claim %+v. %s", *claim, err.Error())
	}
	return nil
}

// fetchDataIfcTopicClaim fetches data stream topic claims for the data Ifc based on the given query string
func (dbAPI *dbObjectModelAPI) fetchDataIfcTopicClaim(context context.Context, query string, claim *DataIfcTopicClaimDBO) ([]DataIfcTopicClaimDBO, error) {
	rtnClaims := []DataIfcTopicClaimDBO{}
	err := dbAPI.QueryIn(context, &rtnClaims, query, claim)
	if err != nil {
		glog.Errorf("failed to find data source claim for data ifc topic: %s", err.Error())
		return nil, errcode.NewInternalError(fmt.Sprintf("failed to find data stream claim for data ifc topic: %s", err.Error()))
	}
	glog.V(5).Infof(base.PrefixRequestID(context, "fetched data ifc claims for query: %s, %v"), query, rtnClaims)
	return rtnClaims, nil
}

// fetchDataIfcTopicClaim fetches data stream topic claims for the data Ifc based on the given query string inside a transaction
func (dbAPI *dbObjectModelAPI) fetchDataIfcTopicClaimInTxn(context context.Context, tx *base.WrappedTx, query string, claim *DataIfcTopicClaimDBO) ([]DataIfcTopicClaimDBO, error) {
	if claim == nil {
		return nil, nil
	}
	rtnClaims := []DataIfcTopicClaimDBO{}
	err := base.QueryTxn(context, tx, &rtnClaims, query, claim)
	if err != nil {
		glog.Errorf("failed to find data source claim for %+v: %s", *claim, err.Error())
		return nil, errcode.NewInternalError(fmt.Sprintf("failed to find data stream claim for data ifc topic: %s", err.Error()))
	}
	glog.V(5).Infof(base.PrefixRequestID(context, "fetched data ifc claims for query: %s, %v"), query, rtnClaims)
	return rtnClaims, nil
}

func (dbAPI *dbObjectModelAPI) deleteIfcClaimByEntityID(context context.Context, tx *base.WrappedTx, entityID, entityType string) error {
	var err error
	switch entityType {
	case entityTypeDataStream:
		_, err = base.DeleteTxn(context, tx, "data_source_topic_claim", map[string]interface{}{"data_stream_id": entityID})
	case entityTypeApplication:
		_, err = base.DeleteTxn(context, tx, "data_source_topic_claim", map[string]interface{}{"application_id": entityID})
	default:
		return fmt.Errorf("unsupported entity type %s", entityType)
	}
	if err != nil {
		return errcode.TranslateDatabaseError(entityID, err)
	}
	return nil
}

// unclaimDataIfcTopic removes the claim of the given entity from the data source field/topic
// FIXME: Adding field_id as an FK to Claim table would only require one delete query with the help of casdcade  delete.
// Note: This method is not transaction safe and caller has to make sure to manage txns.
func (dbAPI *dbObjectModelAPI) unclaimDataIfcTopic(context context.Context, tx *base.WrappedTx, endpoint *model.DataIfcEndpoint, entityType, entityID string) error {
	if endpoint == nil {
		return nil
	}

	glog.V(5).Infof(base.PrefixRequestID(context, "unclaiming data Ifc topic for %s %s and endpoint: %v"), entityType, entityID, *endpoint)

	// Delete data source field created for this entity
	deleteParams := DataSourceFieldDeleteParams{DataSourceID: endpoint.ID,
		FieldType: model.IfcOutFieldType, Name: &endpoint.Name, MQTTTopic: &endpoint.Value,
	}
	err := dbAPI.deleteDataSourceFieldByParams(context, tx, deleteParams)
	if err != nil {
		return fmt.Errorf("failed to delete data source field entry with params: %+v. %s", deleteParams, err.Error())
	}

	glog.V(5).Infof(base.PrefixRequestID(context, "successfully deleted data source fields  by param %v"), deleteParams)

	// Clean up the entry for the entity in the claims table
	err = dbAPI.deleteIfcClaimByEntityID(context, tx, entityID, entityType)
	if err != nil {
		return fmt.Errorf("failed to delete claim for %s %s. %s", entityType, entityID, err.Error())
	}

	glog.V(5).Infof(base.PrefixRequestID(context, "successfully deleted Ifc claim for  %s %s"), entityType, entityID)
	return nil
}

// claimDataIfcTopic adds data fields to a data source and records the claim for the given entity
// Note: This method is not transaction safe and caller has to make sure to manage txns.
func (dbAPI *dbObjectModelAPI) claimDataIfcTopic(context context.Context, tx *base.WrappedTx, endpoint *model.DataIfcEndpoint, entityType, entityID, tenantID string) error {
	if endpoint == nil {
		return nil
	}
	glog.V(5).Infof(base.PrefixRequestID(context, "claiming data Ifc topic for %s %s and endpoint: %v"), entityType, entityID, *endpoint)

	// Fetch fields in this transaction to account for any deletions that might have happened within the txn
	dataSourceFieldDBOs := []DataSourceFieldDBO{}
	err := base.QueryTxn(context, tx, &dataSourceFieldDBOs, queryMap["SelectDataSourceFields"], endpoint)
	if err != nil {
		return err
	}

	fieldExists := false
	for _, field := range dataSourceFieldDBOs {
		if field.Name == endpoint.Name {
			if field.MQTTTopic == endpoint.Value {
				fieldExists = true
				break
			} else {
				return errcode.NewBadRequestExError("DataIfcEndpoints",
					fmt.Sprintf("topic %s already exists and is not claimed by %s %s. Existing Topic/field: %+v", endpoint.Value, entityType, entityID, field),
				)
			}
		}
	}

	if !fieldExists {
		// Update the source field info for the data interface
		dataSourceInfo := model.DataSourceFieldInfo{model.DataSourceFieldInfoCore{Name: endpoint.Name, FieldType: model.IfcOutFieldType}, endpoint.Value}
		glog.V(5).Infof(base.PrefixRequestID(context, "creating data source field %v for %s %s"), dataSourceInfo, entityType, entityID)
		err = dbAPI.createDataSourceField(context, tx, dataSourceInfo, endpoint.ID)
		if err != nil {
			return errcode.NewInternalError(fmt.Sprintf("failed to add topic %s to data interface %s : %+v", endpoint.Value, endpoint.ID, err))
		}
	}

	// claims should be fetched in transaction to take the deleted claims into account which might have been
	// done  as part of unclaimTopic call
	claims, err := dbAPI.fetchDataIfcTopicClaimInTxn(context, tx, queryMap["SelectDataIfcClaimByTopic"],
		&DataIfcTopicClaimDBO{TenantID: tenantID, DataSourceID: endpoint.ID, Topic: endpoint.Value},
	)
	if err != nil {
		return err
	}

	if len(claims) > 1 {
		glog.Errorf(base.PrefixRequestID(context, "unexpected number of claims found for topic %s on data source %s"), endpoint.Value, endpoint.ID)
		return errcode.NewInternalError(fmt.Sprintf("unexpected number of claims found for topic %s on data source %s", endpoint.Value, endpoint.ID))
	}

	// Check if the claim exists, and if it exists, make sure it is bound to the same app/datastream
	if len(claims) == 1 {
		claim := claims[0]
		switch entityType {
		case entityTypeDataStream:
			if claim.DataStreamID == nil || *claim.DataStreamID != entityID {
				// TODO: Change this to a 409
				return errcode.NewPreConditionFailedError(fmt.Sprintf("topic %s on data source %s is already taken. please try another topic", endpoint.Value, endpoint.ID))
			}
		case entityTypeApplication:
			if claim.ApplicationID == nil || *claim.ApplicationID != entityID {
				// TODO: Change this to a 409
				return errcode.NewPreConditionFailedError(fmt.Sprintf("topic %s on data source %s is already taken. please try another topic", endpoint.Value, endpoint.ID))
			}
		default:
			return fmt.Errorf("cannot claim topics for entity of type %q", entityType)
		}

		// the claim is already bound to the same app/datastream, nothing to do
		return nil
	}

	glog.V(5).Infof(base.PrefixRequestID(context, "creating data ifc claim for %s %s and endpoint %v"), entityType, entityID, *endpoint)

	switch entityType {
	case entityTypeDataStream:
		err = dbAPI.insertDataIfcClaim(context, tx, &DataIfcTopicClaimDBO{TenantID: tenantID, DataSourceID: endpoint.ID, DataStreamID: &entityID, Topic: endpoint.Value})
	case entityTypeApplication:
		err = dbAPI.insertDataIfcClaim(context, tx, &DataIfcTopicClaimDBO{TenantID: tenantID, DataSourceID: endpoint.ID, ApplicationID: &entityID, Topic: endpoint.Value})
	default:
		err = fmt.Errorf("cannot claim topics for entity of type %q", entityType)
	}

	// We don't expect any error to occur at this step
	if err != nil {
		glog.Error(err.Error())
		return errcode.NewInternalError(fmt.Sprintf("failed to update data interface: %s", err.Error()))
	}

	glog.V(5).Infof(base.PrefixRequestID(context, "successfully claimed data ifc claim for %s %s"), entityType, entityID)
	return nil
}
