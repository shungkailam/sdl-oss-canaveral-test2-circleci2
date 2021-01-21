package api

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"fmt"
)

type ServiceDomainBindingService interface {
	Set(tx *base.WrappedTx, id string, binding *model.ServiceDomainBinding) error
	SetAll(tx *base.WrappedTx, bindings map[string]*model.ServiceDomainBinding) error
	Get(id string) (*model.ServiceDomainBinding, error)
	GetAll(ids []string) (map[string]*model.ServiceDomainBinding, error)
}

type dataDriverParamsServiceDomainBinding struct {
	dbAPI   *dbObjectModelAPI
	context context.Context
}

type paramBindingEdgeDBO struct {
	ID       int64  `json:"id" db:"id"`
	ParamsID string `json:"paramsId" db:"params_id"`
	EdgeID   string `json:"edgeId" db:"edge_id"`
	State    string `json:"state" db:"state"`
}

type paramsBindingEdgeSelectorDBO struct {
	model.CategoryInfo `json:"categoryInfo" db:"category_info"`
	ID                 int64  `json:"id" db:"id"`
	ParamsID           string `json:"paramsId" db:"params_id"`
	CategoryValueID    int64  `json:"categoryValueId" db:"category_value_id"`
}

type paramsBindingFilter struct {
	ParamsID string `json:"params_id" db:"params_id"`
}

func init() {
	queryMap["SelectDataDriverEdges"] = `SELECT * FROM data_driver_edge_model WHERE params_id = :params_id`
	queryMap["CreateDataDriverEdge"] = `INSERT INTO data_driver_edge_model (params_id, edge_id, state) VALUES (:params_id, :edge_id, :state)`
	queryMap["DeleteDataDriverEdges"] = "DELETE FROM data_driver_edge_model where params_id = :params_id"

	queryMap["SelectDataDriverEdgeSelectors"] = `SELECT data_driver_edge_selector_model.*, category_value_model.category_id "category_info.id", category_value_model.value "category_info.value"
	  FROM data_driver_edge_selector_model JOIN category_value_model ON data_driver_edge_selector_model.category_value_id = category_value_model.id WHERE data_driver_edge_selector_model.params_id = :params_id`
	queryMap["CreateDataDriverEdgeSelector"] = `INSERT INTO data_driver_edge_selector_model (params_id, category_value_id) VALUES (:params_id, :category_value_id)`
	queryMap["DeleteDataDriverEdgeSelectors"] = "DELETE FROM data_driver_edge_selector_model where params_id = :params_id"
}

func NewDataDriverConfigBinding(context context.Context, dbAPI *dbObjectModelAPI) ServiceDomainBindingService {
	return &dataDriverParamsServiceDomainBinding{
		dbAPI:   dbAPI,
		context: context,
	}
}

func NewDataDriverStreamBinding(context context.Context, dbAPI *dbObjectModelAPI) ServiceDomainBindingService {
	return &dataDriverParamsServiceDomainBinding{
		dbAPI:   dbAPI,
		context: context,
	}
}

func (service *dataDriverParamsServiceDomainBinding) Set(tx *base.WrappedTx, id string, binding *model.ServiceDomainBinding) error {
	// Deleting all bindings
	_, err := tx.NamedExec(service.context, queryMap["DeleteDataDriverEdges"], paramsBindingFilter{
		ParamsID: id,
	})
	if err != nil {
		return err
	}

	_, err = tx.NamedExec(service.context, queryMap["DeleteDataDriverEdgeSelectors"], paramsBindingFilter{
		ParamsID: id,
	})
	if err != nil {
		return err
	}

	if binding == nil {
		return nil
	}

	if len(binding.ServiceDomainIDs) > 0 { // edge-based
		for _, edgeId := range binding.ServiceDomainIDs {
			_, err = tx.NamedExec(service.context, queryMap["CreateDataDriverEdge"], &paramBindingEdgeDBO{
				ParamsID: id,
				EdgeID:   edgeId,
				State:    string(model.DeployEntityState),
			})
			if err != nil {
				return errcode.TranslateDatabaseError(id, err)
			}
		}
	} else { // category-based
		for _, categoryInfo := range binding.ServiceDomainSelectors {
			categoryId := categoryInfo.ID
			categoryValueDBOs, err := service.dbAPI.getCategoryValueDBOs(service.context, CategoryValueDBO{CategoryID: categoryId})
			if err != nil {
				return err
			}
			if len(categoryValueDBOs) == 0 {
				return errcode.NewRecordNotFoundError(categoryId)
			}
			valueFound := false
			for _, categoryValueDBO := range categoryValueDBOs {
				if categoryValueDBO.Value == categoryInfo.Value {
					_, err = tx.NamedExec(service.context, queryMap["CreateDataDriverEdgeSelector"], &paramsBindingEdgeSelectorDBO{
						ParamsID:        id,
						CategoryValueID: categoryValueDBO.ID,
					})
					if err != nil {
						return errcode.TranslateDatabaseError(id, err)
					}
					valueFound = true
					break
				}
			}
			if !valueFound {
				return errcode.NewRecordNotFoundError(fmt.Sprintf("%s:%s", categoryId, categoryInfo.Value))
			}
		}

		for _, edgeId := range binding.ExcludeServiceDomainIDs {
			_, err = tx.NamedExec(service.context, queryMap["CreateDataDriverEdge"], paramBindingEdgeDBO{
				ParamsID: id,
				EdgeID:   edgeId,
				State:    string(model.UndeployEntityState),
			})
			if err != nil {
				return errcode.TranslateDatabaseError(id, err)
			}
		}
	}
	return nil
}

func (service *dataDriverParamsServiceDomainBinding) SetAll(tx *base.WrappedTx, bindings map[string]*model.ServiceDomainBinding) error {
	for id, binding := range bindings {
		err := service.Set(tx, id, binding)
		if err != nil {
			return err
		}
	}
	return nil
}

func (service *dataDriverParamsServiceDomainBinding) Get(id string) (*model.ServiceDomainBinding, error) {
	edgesDBOs := []paramBindingEdgeDBO{}
	err := service.dbAPI.QueryIn(service.context, &edgesDBOs, queryMap["SelectDataDriverEdges"], paramsBindingFilter{
		ParamsID: id,
	})
	if err != nil {
		return nil, err
	}

	selectorDBOs := []paramsBindingEdgeSelectorDBO{}
	err = service.dbAPI.QueryIn(service.context, &selectorDBOs, queryMap["SelectDataDriverEdgeSelectors"], paramsBindingFilter{
		ParamsID: id,
	})
	if err != nil {
		return nil, err
	}

	res := model.ServiceDomainBinding{}
	if len(selectorDBOs) == 0 {
		res.ServiceDomainIDs = []string{}
		res.ExcludeServiceDomainIDs = []string{}
		for _, edgesDBO := range edgesDBOs {
			edgeID := edgesDBO.EdgeID
			if edgesDBO.State == string(model.DeployEntityState) {
				res.ServiceDomainIDs = append(res.ServiceDomainIDs, edgeID)
			} else {
				res.ExcludeServiceDomainIDs = append(res.ExcludeServiceDomainIDs, edgeID)
			}
		}
	} else {
		res.ExcludeServiceDomainIDs = []string{}
		for _, edgesDBO := range edgesDBOs {
			if edgesDBO.State == string(model.UndeployEntityState) {
				res.ExcludeServiceDomainIDs = append(res.ExcludeServiceDomainIDs, edgesDBO.EdgeID)
			}
		}

		res.ServiceDomainSelectors = []model.CategoryInfo{}
		for _, selector := range selectorDBOs {
			res.ServiceDomainSelectors = append(res.ServiceDomainSelectors, selector.CategoryInfo)
		}
	}
	return &res, nil
}

func (service *dataDriverParamsServiceDomainBinding) GetAll(ids []string) (map[string]*model.ServiceDomainBinding, error) {
	res := make(map[string]*model.ServiceDomainBinding)
	for _, id := range ids {
		binding, err := service.Get(id)
		if err != nil {
			return nil, err
		}
		res[id] = binding
	}
	return res, nil
}

func (dbAPI *dbObjectModelAPI) cleanupServiceDomainBinding(ctx context.Context, project *model.Project, binding *model.ServiceDomainBinding) error {
	if project.EdgeSelectorType == model.ProjectEdgeSelectorTypeExplicit {
		binding.ServiceDomainIDs = base.And(binding.ServiceDomainIDs, project.EdgeIDs)
		binding.ExcludeServiceDomainIDs = base.And(binding.ExcludeServiceDomainIDs, project.EdgeIDs)
		binding.ServiceDomainSelectors = nil
	} else {
		binding.ServiceDomainIDs = []string{}
		binding.ServiceDomainSelectors = model.CategoryAnd(binding.ServiceDomainSelectors, project.EdgeSelectors)

		excludeEdgeIDsMap := map[string]bool{}
		for _, excludeEdgeID := range binding.ExcludeServiceDomainIDs {
			excludeEdgeIDsMap[excludeEdgeID] = false
		}

		edgeClusterIDLabelsList, err := dbAPI.SelectEdgeClusterIDLabels(ctx)
		if err != nil {
			return err
		}

		// Cleanup categories we do not have in out project
		for _, edgeClusterIDLabels := range edgeClusterIDLabelsList {
			if model.CategoryMatch(edgeClusterIDLabels.Labels, binding.ServiceDomainSelectors) {
				if _, ok := excludeEdgeIDsMap[edgeClusterIDLabels.ID]; ok {
					// ID is valid
					excludeEdgeIDsMap[edgeClusterIDLabels.ID] = true
				}
			}
		}

		binding.ExcludeServiceDomainIDs = []string{}
		for excludeEdgeID, valid := range excludeEdgeIDsMap {
			if valid {
				binding.ExcludeServiceDomainIDs = append(binding.ExcludeServiceDomainIDs, excludeEdgeID)
			}
		}
	}
	return nil
}
