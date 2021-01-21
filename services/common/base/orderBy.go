package base

import (
	"cloudservices/common/errcode"
	"cloudservices/common/filter"
	"fmt"
	"strings"

	"github.com/golang/glog"
)

// PageQueryParameter is the interface for page params
type PageQueryParameter interface {
	GetPageIndex() int
	GetPageSize() int
}

// FilterAndOrderByParameter is the interface for filter params
type FilterAndOrderByParameter interface {
	GetFilter() string
	GetOrderBy() []string
}

// QueryParameter is the interface for query params
type QueryParameter interface {
	PageQueryParameter
	FilterAndOrderByParameter
}

// PageQueryParam is the implementation of PageQueryParameter
type PageQueryParam struct {
	PageIndex int `json:"pageIndex"`
	PageSize  int `json:"pageSize"`
}

// FilterAndOrderByParam is the implementation of FilterAndOrderByParameter
type FilterAndOrderByParam struct {
	Filter  string   `json:"filter"`
	OrderBy []string `json:"orderBy"`
}

func (pageQueryParam *PageQueryParam) GetPageIndex() int {
	return pageQueryParam.PageIndex
}

func (pageQueryParam *PageQueryParam) GetPageSize() int {
	return pageQueryParam.PageSize
}

func (filterAndOrderByParam *FilterAndOrderByParam) GetFilter() string {
	return filterAndOrderByParam.Filter
}

func (filterAndOrderByParam *FilterAndOrderByParam) GetOrderBy() []string {
	return filterAndOrderByParam.OrderBy
}

type OrderByHelper struct {
	// logical to real field/key mapping
	orderByMap map[string]map[string]string
	// contains all logical fields/keys
	orderByKeys map[string][]string
}

func NewOrderByHelper() *OrderByHelper {
	return &OrderByHelper{
		orderByMap:  make(map[string]map[string]string),
		orderByKeys: make(map[string][]string),
	}
}

func (obh *OrderByHelper) Setup(entityType string, keys []string) {
	obh.orderByKeys[entityType] = make([]string, 0, len(keys))
	km := make(map[string]string)
	for _, key := range keys {
		tokens := strings.SplitN(key, ":", 2)
		logicalKey := strings.TrimSpace(tokens[0])
		realKey := logicalKey
		if len(tokens) > 1 {
			realKey = strings.TrimSpace(tokens[1])
		}
		km[logicalKey] = realKey
		obh.orderByKeys[entityType] = append(obh.orderByKeys[entityType], logicalKey)
	}
	obh.orderByMap[entityType] = km
}

// Can be optimized later
func (obh *OrderByHelper) ContainsLogicalKey(entityType string) bool {
	for key, val := range obh.orderByMap[entityType] {
		if key != val {
			return true
		}
	}
	return false
}

func (obh *OrderByHelper) GetOrderByKeys(entityType string) []string {
	return obh.orderByKeys[entityType]
}

func (obh *OrderByHelper) GetLogicalKeyMap(entityType string) map[string]string {
	return obh.orderByMap[entityType]
}

// BuildOrderByClause build order by clause for the given entityType and orderBy list
// Return defaultOrderBy if no key map defined for the entityType or if the orderBy list is empty
// Each entry in orderBy if of the form: key [asc | desc]
// Return: ORDER BY <key 1> [DESC], ..., <key n> [DESC]
//         for each key in keyMap
// Currently keys not in keyMap for the entityType are ignored.
func (obh *OrderByHelper) BuildOrderByClause(entityType string, orderBy []string, defaultOrderBy string, defaultTableAlias string, tableAliasMapping map[string]string) (obc string, err error) {
	obc = defaultOrderBy

	// log for debugging
	if glog.V(5) {
		defer func() {
			glog.V(5).Infof("BuildOrderByClause: entityType=%s, orderBy=%+v, return=%s\n", entityType, orderBy, obc)
		}()
	}

	parts := []string{}
	keyMap := obh.orderByMap[entityType]
	if keyMap == nil {
		err = errcode.NewBadRequestExError("orderBy", fmt.Sprintf("Unsupported orderBy entity type: %s", entityType))
		return
	}
	for _, ob := range orderBy {
		fields := strings.Fields(ob)
		n := len(fields)
		if n == 0 {
			continue
		}
		key := strings.ToLower(fields[0])
		realKey, ok := keyMap[key]
		if !ok {
			err = errcode.NewBadRequestExError("orderBy", fmt.Sprintf("Unsupported orderBy key: %s", fields[0]))
			return
		}
		alias := defaultTableAlias
		if tableAliasMapping != nil {
			if val, ok := tableAliasMapping[realKey]; ok {
				alias = val
			}
		}
		if len(alias) > 0 {
			realKey = fmt.Sprintf("%s.%s", alias, realKey)
		}
		if n > 1 && strings.ToLower(fields[1]) == "desc" {
			parts = append(parts, fmt.Sprintf("%s DESC", realKey))
		} else {
			parts = append(parts, realKey)
		}
	}
	if len(parts) == 0 {
		return
	}
	return fmt.Sprintf("ORDER BY %s", strings.Join(parts, ", ")), err
}

func (obh *OrderByHelper) HasKey(entityType string, key string) bool {
	m := obh.orderByMap[entityType]
	if m == nil {
		return false
	}
	_, ok := m[key]
	return ok
}

func (obh *OrderByHelper) ValidateFilter(filterExpr string, entityType string) (*filter.Expression, error) {
	m := obh.orderByMap[entityType]
	if m == nil {
		return nil, errcode.NewBadRequestExError("filter", fmt.Sprintf("Unsupported filter entity type: %s", entityType))
	}
	expr, err := filter.Parse(filterExpr)
	if err != nil {
		return nil, err
	}
	return expr, filter.ValidateFilter(expr, m)
}

// GetFilterAndOrderBy gets the filter and order SQL sub-clause
func (obh *OrderByHelper) GetFilterAndOrderBy(entityType string, queryParam QueryParameter, defaultOrderBy string) (string, error) {
	return obh.GetFilterAndOrderByWithTableAlias(entityType, queryParam, defaultOrderBy, "", nil)
}

// GetFilterAndOrderByWithTableAlias gets the filter and order SQL sub-clause with table alias substitution
func (obh *OrderByHelper) GetFilterAndOrderByWithTableAlias(entityType string, queryParam FilterAndOrderByParameter, defaultOrderBy, defaultAlias string, aliasMapping map[string]string) (string, error) {
	filterAndOrderBy := defaultOrderBy
	if queryParam != nil {
		orderBy := queryParam.GetOrderBy()
		if len(orderBy) != 0 {
			var err error
			filterAndOrderBy, err = obh.BuildOrderByClause(entityType, orderBy, defaultOrderBy, defaultAlias, aliasMapping)
			if err != nil {
				return "", err
			}
		}
		filterStr := queryParam.GetFilter()
		if filterStr != "" {
			expr, err := obh.ValidateFilter(filterStr, entityType)
			if err != nil {
				return "", err
			}
			if len(defaultAlias) > 0 || (aliasMapping != nil && len(aliasMapping) > 0) || obh.ContainsLogicalKey(entityType) {
				filterStr = filter.TransformFields(expr, obh.GetLogicalKeyMap(entityType), defaultAlias, aliasMapping)
			}
			filterAndOrderBy = fmt.Sprintf("AND %s %s", filterStr, filterAndOrderBy)
		}
	}
	return filterAndOrderBy, nil
}

// BuildQuery builds the complete SQL query from the template using the query params
func (obh *OrderByHelper) BuildQuery(entityType, queryTemplate string, queryParam FilterAndOrderByParameter, defaultOrderBy string) (string, error) {
	return obh.BuildQueryWithTableAlias(entityType, queryTemplate, queryParam, defaultOrderBy, "", nil)
}

// BuildQueryWithTableAlias builds the complete SQL query from the template using the query params with table alias substitution
func (obh *OrderByHelper) BuildQueryWithTableAlias(entityType, queryTemplate string, queryParam FilterAndOrderByParameter, defaultOrderBy, defaultAlias string, aliasMapping map[string]string) (string, error) {
	filterAndOrderBy, err := obh.GetFilterAndOrderByWithTableAlias(entityType, queryParam, defaultOrderBy, defaultAlias, aliasMapping)
	if err != nil {
		return "", err
	}
	queryTemplate = strings.TrimSpace(queryTemplate)
	if !strings.HasSuffix(queryTemplate, " %s") {
		queryTemplate = queryTemplate + " %s"
	}
	return fmt.Sprintf(queryTemplate, filterAndOrderBy), nil
}

// BuildPagedQuery build query with pagination params
func (obh *OrderByHelper) BuildPagedQuery(entityType, queryTemplate string, queryParam QueryParameter, defaultOrderBy string) (string, PageQueryParameter, error) {
	query, err := obh.BuildQuery(entityType, queryTemplate, queryParam, defaultOrderBy)
	if err != nil {
		return "", nil, err
	}
	query, pageQueryParam := GetPagedQuery(query, queryParam)
	return query, pageQueryParam, nil
}

// BuildPagedQueryWithTableAlias returns the modified query with pagination filter params
func (obh *OrderByHelper) BuildPagedQueryWithTableAlias(entityType, queryTemplate string, queryParam QueryParameter, defaultOrderBy, defaultAlias string, aliasMapping map[string]string) (string, PageQueryParameter, error) {
	query, err := obh.BuildQueryWithTableAlias(entityType, queryTemplate, queryParam, defaultOrderBy, defaultAlias, aliasMapping)
	if err != nil {
		return "", nil, err
	}
	query, pageQueryParam := GetPagedQuery(query, queryParam)
	return query, pageQueryParam, nil
}

// GetPagedQuery returns the modified query and the affective pagination filter params
func GetPagedQuery(query string, queryParam PageQueryParameter) (string, PageQueryParameter) {
	pageIndex := 0
	pageSize := MaxRowsLimit
	if queryParam != nil {
		pageIndex = queryParam.GetPageIndex()
		pageSize = queryParam.GetPageSize()
		if pageIndex < 0 {
			pageIndex = 0
		}
		if pageSize <= 0 {
			pageSize = MaxRowsLimit
		}
	}
	pageQueryParam := &PageQueryParam{PageIndex: pageIndex, PageSize: pageSize}
	return fmt.Sprintf("%s OFFSET %d LIMIT %d", query, pageIndex*pageSize, pageSize), pageQueryParam
}
