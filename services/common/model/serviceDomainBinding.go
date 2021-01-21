package model

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"fmt"
	"github.com/thoas/go-funk"
	"strings"
)

type ServiceDomainBinding struct {
	//
	// Service domains listed according to ID where the data driver config is deployed.
	// Only relevant if the parent project EdgeSelectorType value is set to Explicit.
	//
	// required: false
	ServiceDomainIDs []string `json:"serviceDomainIds,omitempty"`
	//
	// Service domains to be excluded from the data driver config deployment.
	//
	// required: false
	ExcludeServiceDomainIDs []string `json:"excludeServiceDomainIds,omitempty"`
	//
	// Select service domains according to CategoryInfo.
	// Only relevant if the parent project EdgeSelectorType value is set to Category.
	//
	// required: false
	ServiceDomainSelectors []CategoryInfo `json:"serviceDomainSelectors,omitempty"`
}

func ValidateServiceDomainBinding(model *ServiceDomainBinding, project *Project) error {
	if model == nil {
		return errcode.NewBadRequestError("ServiceDomainBinding")
	}

	if project.EdgeSelectorType == ProjectEdgeSelectorTypeCategory {
		model.ServiceDomainIDs = nil

		if len(model.ServiceDomainSelectors) == 0 {
			return errcode.NewBadRequestError("ServiceDomainSelectors")
		}
	} else {
		model.ServiceDomainIDs = base.Unique(model.ServiceDomainIDs)
		if len(model.ServiceDomainIDs) == 0 {
			return errcode.NewBadRequestError("ServiceDomainIDs")
		}

		// doc.EdgeIDs must be a subset of project.EdgeIDs
		badEdgeIDs := []string{}
		for _, edgeID := range project.EdgeIDs {
			if !funk.Contains(project.EdgeIDs, edgeID) {
				badEdgeIDs = append(badEdgeIDs, edgeID)
			}
		}
		if len(badEdgeIDs) != 0 {
			msg := fmt.Sprintf("Service domain with IDs %s are not part of the project", strings.Join(badEdgeIDs, ", "))
			return errcode.NewBadRequestExError("ServiceDomainIDs", msg)
		}
		model.ServiceDomainSelectors = nil
	}

	return nil
}
