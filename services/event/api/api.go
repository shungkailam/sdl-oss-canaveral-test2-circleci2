package api

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/policy"
	"cloudservices/event/config"
	gapi "cloudservices/event/generated/grpc"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"

	"google.golang.org/grpc"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/golang/glog"

	"github.com/olivere/elastic"
	"github.com/sha1sum/aws_signing_client"
)

const (
	DefaultSearchSize = 100

	DefaultSearchMappings = `
	{
		"settings":{
			"analysis": {
				"analyzer": {
					"path_analyzer": {
						"type" : "custom",
						"tokenizer": "path_hierarchy"
					}
				},
				"normalizer": {
					"my_normalizer": {
					  "type": "custom",
					  "char_filter": [],
					  "filter": ["lowercase", "asciifolding"]
					}
				}
			},
			"number_of_shards":  2,
			"number_of_replicas": 1
		},
		"mappings": {
			"doc": {
			  "properties": {
				"path": {
				  "type": "text",
				  "analyzer": "path_analyzer",
				  "search_analyzer": "path_analyzer"
				},
				"message" : {
					"type":       "text",
					"index": false
				},
				"projectID" : {
					"type":       "keyword",
					"normalizer": "my_normalizer"
				},
				"timestamp": {
					"type": "date",
					"doc_values": true
				},
				"id": {
					"type": "keyword",
					"doc_values": true
				},
				"properties" : {
					"type":  "object",
					"enabled": false
				},
				"metrics" : {
					"type": "object",
					"enabled": false
				},
				"isInfraEntity" : {
					"type": "boolean",
					"index": true,
					"doc_values" : false
				},
				"audience" :{
					"type" : "keyword"
				}
			  }
			}
		}
	}`

	// Update version for each change to index or its mappings
	Version = "v5"
)

// SearchCriteria holds the search parameter for the search API
type SearchCriteria struct {
	TenantID     string
	ProjectIDs   []string
	IsInfraAdmin bool
	PathRegex    string
	Keys         map[string]string
	DocType      string
	SortKey      string
	Desc         bool
	Start        int
	Size         int
	StartTime    *timestamp.Timestamp // Inclusive
	EndTime      *timestamp.Timestamp // Exclusive
	Callback     func(ID string, doc interface{}) error
	Model        interface{}
}

// ObjectModelAPI captures all object model APIs
type APIServer interface {
	Register(gServer *grpc.Server)
	Close() error
	gapi.EventServiceServer
}

// So fare index is only used to serialize access too indices
// during create and update.
type Index struct {
	mtx         sync.Mutex
	initialized bool
}

type apiServer struct {
	search            *elastic.Client
	indices           sync.Map // map name to index
	pathPolicyManager *policy.Manager
}

func getHTTPClient(options ...func(*http.Client) error) (*http.Client, error) {
	httpClient := &http.Client{}
	for _, option := range options {
		if err := option(httpClient); err != nil {
			return nil, err
		}
	}
	return httpClient, nil
}

// withAWS AWS requires HTTP requests to be signed
func withAWS(httpClient *http.Client) error {
	// No lock for now
	// Create AWS session
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(*config.Cfg.AWSRegion)},
	)
	if err != nil {
		return err
	}
	signer := v4.NewSigner(awsSession.Config.Credentials)
	_, err = aws_signing_client.New(signer, httpClient, "es", *awsSession.Config.Region)
	return err
}

// noSSL skip SSL verification in httpClient
func noSSL(httpClient *http.Client) error {
	httpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return nil
}

// NewAPIServer creates a APIServer based on sql DB
// Note: api with the same config should be shared
// to minimize resource usage
func NewAPIServer() (APIServer, error) {

	var httpClient *http.Client
	var err error
	isAWS := strings.HasSuffix(*config.Cfg.SearchURL, "amazonaws.com")

	if isAWS {
		httpClient, err = getHTTPClient(withAWS)
	} else {
		// see if SSL enabled
		if *config.Cfg.DisableESSSL {
			httpClient, err = getHTTPClient(noSSL)
		} else {
			httpClient, err = getHTTPClient()
		}
	}
	if err != nil {
		return nil, err
	}

	// Create elasticSearch client
	search, err := elastic.NewClient(
		elastic.SetURL(*config.Cfg.SearchURL),
		elastic.SetHttpClient(httpClient),
		elastic.SetSniff(false),
	)
	if err != nil {
		return nil, err
	}

	pathPolicyManager := policy.NewManager()
	err = pathPolicyManager.LoadPolicies(eventPathPolicies)
	if err != nil {
		return nil, err
	}
	apiServer := &apiServer{search: search, pathPolicyManager: pathPolicyManager}
	return apiServer, err
}

func (server *apiServer) Register(gServer *grpc.Server) {
	gapi.RegisterEventServiceServer(gServer, server)
}

// CreateSearchIndex creates the search index in elasticSearch if it does not exist
func (server *apiServer) CreateSearchIndex(index string, mappings *string) (indexWithNamespace string, err error) {
	indexWithNamespace = GetSearchIndexWithNamespace(index)
	// Look up or create index
	val, _ := server.indices.LoadOrStore(indexWithNamespace, &Index{})
	idx := val.(*Index)
	idx.mtx.Lock()
	defer func() {
		if err != nil {
			// remove entry on failure
			server.indices.Delete(indexWithNamespace)
		} else {
			// mark index as initialized
			idx.initialized = true
		}
		idx.mtx.Unlock()
	}()

	// Already create/updated
	if idx.initialized == true {
		return indexWithNamespace, nil
	}
	search := server.search
	indexAliasWithNamespace := GetSearchIndexAliasWithNamespace(index)
	indexCreateService := search.CreateIndex(indexWithNamespace)
	if mappings == nil {
		mappings = aws.String(DefaultSearchMappings)
	}
	indexCreateService = indexCreateService.BodyJson(*mappings)
	glog.Infof("Create index %s", indexWithNamespace)
	_, err = indexCreateService.Do(context.Background())
	if err != nil {
		if e, ok := err.(*elastic.Error); ok {
			if !strings.Contains(e.Details.Type, "already_exists_exception") {
				glog.Errorf("Failed to create index %s. Error: %s", index, err.Error())
				return "", errcode.TranslateSearchError("Index", err)
			}
		}
	}
	// Idempotent
	_, err = search.Alias().Add(indexWithNamespace, indexAliasWithNamespace).Do(context.Background())
	if err != nil {
		glog.Errorf("Failed to add index %s to alias. Error: %s", index, err.Error())
		return "", errcode.TranslateSearchError("Index", err)
	}
	return indexWithNamespace, nil
}

func (server *apiServer) getSearchDocumentBodyString(index string, tenantID string, id string, doc interface{}) (string, error) {
	if len(index) == 0 {
		glog.Errorf("Error upserting document %v. Index is not set", doc)
		return "", errcode.NewBadRequestError("Index")
	}
	if len(id) == 0 {
		glog.Errorf("Error upserting document %v. ID is not set", doc)
		return "", errcode.NewBadRequestError("ID")
	}
	data, err := base.ConvertToJSON(doc)
	if err != nil {
		glog.Errorf("Error upserting document %v. ID %s, Error: %s", doc, id, err.Error())
		return "", err
	}
	return string(data), nil
}

func (server *apiServer) PutSearchDocument(index string, tenantID string, id string, doc interface{}) (string, error) {
	bodyString, err := server.getSearchDocumentBodyString(index, tenantID, id, doc)
	if err != nil {
		return "", err
	}
	search := server.search
	indexService := search.
		Index().
		Index(index).
		// Type is going away. Multiple types are not supported in 6.0
		Type("doc").
		Id(id).
		BodyString(bodyString).
		Refresh("wait_for")
	indexResponse, err := indexService.Do(context.Background())
	if err != nil {
		glog.Errorf("Error upserting document %v. ID %s, Error: %s", doc, id, err.Error())
		return "", errcode.TranslateSearchError("Index", err)
	}
	return indexResponse.Id, err
}

func (server *apiServer) SearchDocuments(criteria *SearchCriteria) error {
	search := server.search
	modelType := reflect.TypeOf(criteria.Model)
	if modelType.Kind() == reflect.Ptr {
		return errors.New("Model param must be a value")
	}
	indexAliasWithNamespace := GetSearchIndexAliasWithNamespace(criteria.TenantID)
	glog.Infof("Query keys %v, regex: %s on index alias: %s", criteria.Keys, criteria.PathRegex, indexAliasWithNamespace)
	query := elastic.NewBoolQuery()
	query = query.Filter(elastic.NewRegexpQuery("path", criteria.PathRegex))
	// Filter does not add to the score
	for k, v := range criteria.Keys {
		// The values are anyways stored in lower case
		query = query.Filter(elastic.NewTermQuery(k, strings.ToLower(v)))
	}

	values := make([]interface{}, 0)
	// Project-level events have project ID set
	for _, projectID := range criteria.ProjectIDs {
		values = append(values, projectID)
	}
	// Infrastructure-level events don't have project ID set
	if criteria.IsInfraAdmin {
		values = append(values, "null")
		boolQuery := elastic.NewBoolQuery()
		// all the infra scoped entities are also accessible
		// null filter can be removed because projectID = "null" has isInfraEntity set to true
		// but it is added for backward compatibility
		// TODO remove isInfraEntity later
		// Any one or more of the boolean conditions must satisfy
		query = query.Filter(boolQuery.Should(elastic.NewTermsQuery("audience", InfraEventAudience, InfraProjectEventAudience), elastic.NewTermsQuery("isInfraEntity", true), elastic.NewTermsQuery("projectID", values...)))
	} else if len(values) == 0 {
		return nil
	} else {
		boolQuery := elastic.NewBoolQuery()
		// False boolean fields are omitted in the document. So existence check is required
		// Either not existent or false
		// TODO remove isInfraEntity later
		// Any one of the boolean conditions AND the project ID filter must satisfy
		boolQuery = boolQuery.Should(elastic.NewTermsQuery("audience", ProjectEventAudience, InfraProjectEventAudience), elastic.NewBoolQuery().MustNot(elastic.NewExistsQuery("isInfraEntity"), elastic.NewTermsQuery("isInfraEntity", false)))
		query = query.Filter(elastic.NewTermsQuery("projectID", values...), boolQuery)
	}

	var rangeQuery *elastic.RangeQuery
	if criteria.StartTime != nil {
		goTime, err := ptypes.Timestamp(criteria.StartTime)
		if err != nil {
			glog.Errorf("Invalid start time. Error: %s", err.Error())
			return errcode.NewBadRequestError("startTime")
		}
		rangeQuery = elastic.NewRangeQuery("timestamp").To(goTime).IncludeUpper(true)
	}
	if criteria.EndTime != nil {
		goTime, err := ptypes.Timestamp(criteria.EndTime)
		if err != nil {
			glog.Errorf("Invalid end time. Error: %s", err.Error())
			return errcode.NewBadRequestError("endTime")
		}
		if rangeQuery == nil {
			rangeQuery = elastic.NewRangeQuery("timestamp").From(goTime).IncludeLower(false)
		} else {
			rangeQuery = rangeQuery.From(goTime).IncludeLower(false)
		}
	}
	if rangeQuery != nil {
		query = query.Filter(rangeQuery)
	}
	if glog.V(5) {
		src, _ := query.Source()
		ba, _ := json.MarshalIndent(src, " ", " ")
		glog.V(5).Infof("Running query %s", string(ba))
	}
	searchService := search.Search().
		Index(indexAliasWithNamespace).
		// Type is going away. Multiple types are not supported in 6.0
		Type("doc").
		Query(query).
		// Pick the latest across indexes
		Collapse(elastic.NewCollapseBuilder("id")).
		From(criteria.Start).
		Pretty(true)
	if len(criteria.SortKey) > 0 {
		searchService = searchService.Sort(criteria.SortKey, !criteria.Desc)
	}
	if criteria.Size == 0 {
		criteria.Size = DefaultSearchSize
	}
	searchService = searchService.Size(criteria.Size)
	searchResult, err := searchService.Do(context.Background())
	if err != nil {
		glog.Errorf("Error querying for keys: %v, path regex: %s. Error: %s", criteria.Keys, criteria.PathRegex, err.Error())
		if e, ok := err.(*elastic.Error); ok {
			if strings.Contains(e.Details.Type, "index_not_found_exception") {
				glog.Warningf("Index %s is not found", indexAliasWithNamespace)
				return nil
			}
		}
		return errcode.TranslateSearchError("regex", err)
	}
	if searchResult.Hits == nil || searchResult.Hits.Hits == nil || len(searchResult.Hits.Hits) == 0 {
		return nil
	}
	for _, hit := range searchResult.Hits.Hits {
		v := reflect.New(modelType).Elem()
		if hit.Source != nil {
			if err := base.ConvertFromJSON(*hit.Source, v.Addr().Interface()); err == nil {
				err = criteria.Callback(hit.Id, v.Interface())
				if err != nil {
					glog.Errorf("Error in callback for keys: %v, path regex: %s. Error: %s", criteria.Keys, criteria.PathRegex, err.Error())
					return err
				}
			} else {
				glog.Warningf("Error in unmarshalling document. Error: %+v", err)
			}
		}
	}

	return nil
}

func (server *apiServer) DeleteSearchDocuments(index string) error {
	search := server.search
	_, err := search.DeleteIndex(index).Do(context.Background())
	if err != nil {
		glog.Errorf("Error in deleting index %s. Error: %s", index, err.Error())
		return errcode.TranslateSearchError("Index", err)
	}
	return nil
}

func (server *apiServer) Close() error {
	server.search.Stop()
	return nil
}

// GetSearchIndexWithNamespace returns the timestamped index
// Format <namespace>.<index>.2018-08-01
func GetSearchIndexWithNamespace(index string) string {
	env := config.Cfg.Environment
	if env == nil {
		env = base.StringPtr("dev")
	}
	currTime := time.Now().UTC()
	timestamp := currTime.Format("2006-01")
	return strings.ToLower(fmt.Sprintf("%s.%s.%s.%s",
		*env, Version, index, timestamp))
}

// GetSearchIndexAliasWithNamespace returns the alias for the index
// Format <namespace>.<index>
func GetSearchIndexAliasWithNamespace(index string) string {
	env := config.Cfg.Environment
	if env == nil {
		env = base.StringPtr("dev")
	}
	return strings.ToLower(fmt.Sprintf("%s.%s.alias", *env, index))
}
