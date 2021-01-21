package model

import (
	"cloudservices/common/errcode"
	"strings"
)

// CategoryInfo - a choice of a value from a category.
// swagger:model CategoryInfo
type CategoryInfo struct {
	//
	// The category ID.
	// For example: the ID for the Airport category.
	//
	// required: true
	ID string `json:"id" db:"id"`
	//
	// An allowed value to choose for the category.
	// For example:  SFO, SJC, LAX, and so on.
	//
	// required: true
	Value string `json:"value" db:"value"`
}

// Category is object model for Category.
//
// Similar to labels for Kubernetes.
// Category logically groups edges, data sources, and other items.
// Applying a category to an entity applies any values and attributes associated with the category to the entity.
//
// swagger:model Category
type Category struct {
	// required: true
	BaseModel
	//
	// Unique category name.
	// For example: Airport, Terminal, Floor, Environment, Department, and so on.
	//
	// required: true
	Name string `json:"name" validate:"range=1:200"`
	// Purpose of the category.
	// required: true
	Purpose string `json:"purpose" validate:"range=0:200"`
	//
	// All allowed category values. For example:
	//   SFO, ORD, LAX, ...
	//   1, 2, 3, ...
	//   Production, Dev, Test, ...
	//   Sales, HR, Eng, ...
	//
	// required: true
	Values []string `json:"values" validate:"range=1"`
}

// CategoryUsage captures usage for a category or category value
type CategoryUsage struct {
	// IDs of edges using this category
	// required: true
	EdgeIDs []string `json:"edgeIds"`
	// IDs of projects using this category
	// required: true
	ProjectIDs []string `json:"projectIds"`
	// IDs of applications using this category
	// required: true
	ApplicationIDs []string `json:"applicationIds"`
	// IDs of data pipelines using this category
	// required: true
	DataPipelineIDs []string `json:"dataPipelineIds"`
	// IDs of data sources using this category
	// required: true
	DataSourceIDs []string `json:"dataSourceIds"`
}

func NewEmptyCategoryUsage() *CategoryUsage {
	return &CategoryUsage{
		EdgeIDs:         []string{},
		ProjectIDs:      []string{},
		ApplicationIDs:  []string{},
		DataPipelineIDs: []string{},
		DataSourceIDs:   []string{},
	}
}

// CategoryUsageInfo captures usage info for a category
// swagger:model CategoryUsageInfo
type CategoryUsageInfo struct {
	// ID of the category
	// required: true
	ID string `json:"id"`
	// required: true
	CategoryUsage
}

// CategoryDetailUsageInfo captures category usage info details
type CategoryDetailUsageInfo struct {
	// ID of the category
	// required: true
	ID string `json:"id"`
	// UsageMap map of category value to its usage
	// required: true
	UsageMap map[string]*CategoryUsage `json:"usageMap"`
}

// CategoryCreateParam is Category used as API parameter
// swagger:parameters CategoryCreate CategoryCreateV2
type CategoryCreateParam struct {
	// A description of the category creation request.
	// in: body
	// required: true
	Body *Category `json:"body"`
}

// CategoryUpdateParam is Category used as API parameter
// swagger:parameters CategoryUpdate CategoryUpdateV2 CategoryUpdateV3
type CategoryUpdateParam struct {
	// in: body
	// required: true
	Body *Category `json:"body"`
}

// Ok
// swagger:response CategoryGetResponse
type CategoryGetResponse struct {
	// in: body
	// required: true
	Payload *Category
}

// Ok
// swagger:response CategoryUsageGetResponse
type CategoryUsageGetResponse struct {
	// in: body
	// required: true
	Payload *CategoryDetailUsageInfo
}

// Ok
// swagger:response CategoryListResponse
type CategoryListResponse struct {
	// in: body
	// required: true
	Payload *[]Category
}

// Ok
// swagger:response CategoryListResponseV2
type CategoryListResponseV2 struct {
	// in: body
	// required: true
	Payload *CategoryListResponsePayload
}

// payload for CategoryListResponseV2
type CategoryListResponsePayload struct {
	// required: true
	EntityListResponsePayload
	// list of categories
	// required: true
	CategoryList []Category `json:"result"`
}

// Ok
// swagger:response CategoryUsageListResponse
type CategoryUsageListResponse struct {
	// in: body
	// required: true
	Payload *[]CategoryUsageInfo
}

// swagger:parameters CategoryList CategoryListV2 CategoryGet CategoryGetV2 CategoryCreate CategoryCreateV2 CategoryUpdate CategoryUpdateV2 CategoryUpdateV3 CategoryDelete CategoryDeleteV2 CategoryUsageList CategoryUsageGet
// in: header
type categoryAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ObjectRequestBaseCategory is used as websocket Category message
// swagger:model ObjectRequestBaseCategory
type ObjectRequestBaseCategory struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc Category `json:"doc"`
}

type CategoriesByID []Category

func (a CategoriesByID) Len() int           { return len(a) }
func (a CategoriesByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a CategoriesByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

// CategoryMatch checks if the labels match the selectors
// Values within the same category are considered with 'OR' semantics
// Values across different categories are considered with 'AND' semantics
func CategoryMatch(labels []CategoryInfo, selectors []CategoryInfo) bool {
	if len(labels) == 0 || len(selectors) == 0 {
		return false
	}
	matchedCats := map[string]bool{}
	allCats := map[string]bool{}
	for _, selector := range selectors {
		allCats[selector.ID] = true
		if matchedCats[selector.ID] {
			continue
		}
		for _, label := range labels {
			if label.ID == selector.ID && label.Value == selector.Value {
				matchedCats[selector.ID] = true
				break
			}
		}
	}
	return len(matchedCats) == len(allCats)
}

func CategoryAnd(cats1 []CategoryInfo, cats2 []CategoryInfo) []CategoryInfo {
	result := []CategoryInfo{}
	if len(cats1) == 0 {
		return cats2
	}
	for _, cat := range cats1 {
		idMatch := false
		valueMatch := false
		for _, cat2 := range cats2 {
			if cat.ID == cat2.ID {
				idMatch = true
				if cat.Value == cat2.Value {
					valueMatch = true
					break
				}
			}
		}
		if !idMatch || (idMatch && valueMatch) {
			result = append(result, cat)
		}
	}
	for _, cat2 := range cats2 {
		idMatch := false
		for _, cat := range cats1 {
			if cat.ID == cat2.ID {
				idMatch = true
				break
			}
		}
		if !idMatch {
			result = append(result, cat2)
		}
	}
	return result
}

func ValidateCategory(cat *Category) error {
	if cat == nil {
		return errcode.NewBadRequestError("Category")
	}

	h := make(map[string]string)
	for _, v := range cat.Values {
		trimmed := strings.TrimSpace(v)
		lower := strings.ToLower(trimmed)
		h[lower] = trimmed
	}
	delete(h, "")

	filteredValues := make([]string, 0, len(h))
	for _, v := range h {
		filteredValues = append(filteredValues, v)
	}

	newLen := len(filteredValues)
	if newLen == 0 || newLen != len(cat.Values) {
		return errcode.NewBadRequestError("Values")
	}

	cat.Values = filteredValues

	cat.Name = strings.TrimSpace(cat.Name)

	return nil
}
