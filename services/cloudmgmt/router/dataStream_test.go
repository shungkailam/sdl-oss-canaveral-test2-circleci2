package router_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net/http"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	DATA_STREAMS_PATH     = "/v1/datastreams"
	DATA_STREAMS_PATH_NEW = "/v1.0/datapipelines"
)

// create datastream
func createDataStream(netClient *http.Client, datastream *model.DataStream, token string) (model.CreateDocumentResponse, string, error) {
	resp, reqID, err := createEntity(netClient, DATA_STREAMS_PATH, *datastream, token)
	if err == nil {
		datastream.ID = resp.ID
	}
	return resp, reqID, err
}

// update datastream
func updateDataStream(netClient *http.Client, datastreamID string, datastream model.DataStream, token string) (model.UpdateDocumentResponse, string, error) {
	return updateEntity(netClient, fmt.Sprintf("%s/%s", DATA_STREAMS_PATH, datastreamID), datastream, token)
}

// get datastreams
func getDataStreams(netClient *http.Client, token string) ([]model.DataStream, error) {
	datastreams := []model.DataStream{}
	err := doGet(netClient, DATA_STREAMS_PATH, token, &datastreams)
	return datastreams, err
}

func getDataStreamsNew(netClient *http.Client, token string, pageIndex int, pageSize int) (model.DataStreamListPayload, error) {
	response := model.DataStreamListPayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", DATA_STREAMS_PATH_NEW, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}

func getDataStreamsForProject(netClient *http.Client, projectID string, token string) ([]model.DataStream, error) {
	datastreams := []model.DataStream{}
	err := doGet(netClient, PROJECTS_PATH+"/"+projectID+"/datastreams", token, &datastreams)
	return datastreams, err
}

// delete datastream
func deleteDataStream(netClient *http.Client, datastreamID string, token string) (model.DeleteDocumentResponse, string, error) {
	return deleteEntity(netClient, DATA_STREAMS_PATH, datastreamID, token)
}

// get datastream by id
func getDataStreamByID(netClient *http.Client, datastreamID string, token string) (model.DataStream, error) {
	datastream := model.DataStream{}
	err := doGet(netClient, DATA_STREAMS_PATH+"/"+datastreamID, token, &datastream)
	return datastream, err
}

func TestDataStream(t *testing.T) {
	t.Parallel()
	t.Log("running TestDataStream test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")

	// Teardown
	defer func() {
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		dbAPI.DeleteUser(ctx, user.ID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test DataStream", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		category := model.Category{
			Name:    "test-cat",
			Purpose: "",
			Values:  []string{"v1", "v2"},
		}
		_, _, err := createCategory(netClient, &category, token)
		require.NoError(t, err)
		categoryID := category.ID

		cloudcreds := model.CloudCreds{
			Name:        "aws-cloud-creds-name",
			Type:        "AWS",
			Description: "aws-cloud-creds-desc",
			AWSCredential: &model.AWSCredential{
				AccessKey: "foo",
				Secret:    "bar",
			},
			GCPCredential: nil,
		}
		_, _, err = createCloudCreds(netClient, &cloudcreds, token)
		require.NoError(t, err)
		cloudCredsID := cloudcreds.ID

		dockerProfileName := "aws-registry-name"
		dockerProfileDesc := "aws-registry-desc"

		// DockerProfile object, leave ID blank and let create generate it
		dockerprofile := model.DockerProfile{
			Name:         dockerProfileName,
			Type:         "AWS",
			Server:       "a.b.c.d.e.f",
			CloudCredsID: cloudCredsID,
			Description:  dockerProfileDesc,
			UserName:     "username",
			Email:        "test@example.com",
			Credentials:  "{\"AccessKeyId\":\"AWS-Access\",\"SecretAccessKey\":\"AWS-SecretAccessKey\",\"Account\":\"AWS-test\",\"Region\":\"us-west-2\",\"Server\":\"aws-server\",\"User\":\"aws-user\",\"Pwd\":\"aws-pwd\",\"Email\":\"aws-email\"}",
		}
		_, _, err = createDockerProfile(netClient, &dockerprofile, token)
		require.NoError(t, err)
		dockerProfileID := dockerprofile.ID

		project := makeExplicitProject(tenantID, []string{cloudcreds.ID}, []string{dockerprofile.ID}, []string{user.ID}, nil)
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		projectID := project.ID

		dockerfile := "docker file"
		scriptruntime := model.ScriptRuntime{
			ScriptRuntimeCore: model.ScriptRuntimeCore{
				Name:            "script-runtime-name",
				Description:     "script runtime desc",
				Language:        "python",
				Builtin:         false,
				DockerRepoURI:   "docker-repo-uri",
				DockerProfileID: dockerProfileID,
				Dockerfile:      dockerfile,
			},
			ProjectID: projectID,
		}

		_, _, err = createScriptRuntime(netClient, &scriptruntime, token)
		require.NoError(t, err)

		scriptName := "script name"
		scriptType := "Transformation"
		scriptLanguage := "Python"
		scriptEnvrionment := "python tensorflow"
		scriptCode := "def main: print"
		// scriptCodeUpdated := "def main: print 'hello'"

		// Script object, leave ID blank and let create generate it
		script := model.Script{
			ScriptCore: model.ScriptCore{
				Name:        scriptName,
				Type:        scriptType,
				Language:    scriptLanguage,
				Environment: scriptEnvrionment,
				Code:        scriptCode,
				Builtin:     false,
				ProjectID:   projectID,
				RuntimeID:   scriptruntime.ID,
			},
			Params: []model.ScriptParam{},
		}

		_, _, err = createScript(netClient, &script, token)
		require.NoError(t, err)

		var size float64 = 1000000
		dataStreamName := "data-streams-name"
		dataStreamDataType := "Custom"
		// dataStreamDataTypeUpdated := "Image"

		datastream := model.DataStream{
			Name:     dataStreamName,
			DataType: dataStreamDataType,
			Origin:   "DataSource",
			OriginSelectors: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: "v1",
				},
			},
			OriginID:         "",
			Destination:      "Cloud",
			CloudType:        "AWS",
			CloudCredsID:     cloudCredsID,
			AWSCloudRegion:   "us-west-2",
			GCPCloudRegion:   "",
			EdgeStreamType:   "",
			AWSStreamType:    "Kafka",
			GCPStreamType:    "",
			Size:             size,
			EnableSampling:   false,
			SamplingInterval: 0,
			TransformationArgsList: []model.TransformationArgs{
				{
					TransformationID: script.ID,
					Args:             []model.ScriptParamValue{},
				},
			},
			DataRetention: []model.RetentionInfo{},
			ProjectID:     projectID,
		}

		_, _, err = createDataStream(netClient, &datastream, token)
		require.NoError(t, err)

		datastreams, err := getDataStreams(netClient, token)
		require.NoError(t, err)
		t.Logf("got datastreams: %+v", datastreams)
		if len(datastreams) != 1 {
			t.Fatalf("expected datastream count to be 1, got %d", len(datastreams))
		}

		dstreams, err := getDataStreamsForProject(netClient, projectID, token)
		require.NoError(t, err)
		if len(dstreams) != 1 {
			t.Fatalf("expected datastream count to be 1, got %d", len(datastreams))
		}

		if !reflect.DeepEqual(dstreams[0], datastreams[0]) {
			t.Fatalf("expect datastream to equal, but %+v != %+v", dstreams[0], datastreams[0])
		}

		datastreamID := datastream.ID
		datastream.ID = ""
		datastream.Name = fmt.Sprintf("%s-updated", datastream.Name)
		ur, _, err := updateDataStream(netClient, datastreamID, datastream, token)
		require.NoError(t, err)
		if ur.ID != datastreamID {
			t.Fatal("update datastream id mismatch")
		}

		resp, _, err := deleteDataStream(netClient, datastreamID, token)
		require.NoError(t, err)
		if resp.ID != datastreamID {
			t.Fatal("delete datastream id mismatch")
		}

		resp, _, err = deleteProject(netClient, projectID, token)
		require.NoError(t, err)
		if resp.ID != projectID {
			t.Fatal("delete project id mismatch")
		}

		resp, _, err = deleteCloudCreds(netClient, cloudCredsID, token)
		require.NoError(t, err)
		if resp.ID != cloudCredsID {
			t.Fatal("delete cloud creds id mismatch")
		}

		resp, _, err = deleteCategory(netClient, categoryID, token)
		require.NoError(t, err)
		if resp.ID != categoryID {
			t.Fatal("delete category id mismatch")
		}

	})

}

func TestDataStreamPaging(t *testing.T) {
	t.Parallel()
	t.Log("running TestDataStreamPaging test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")

	rand1 := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Teardown
	defer func() {
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		dbAPI.DeleteUser(ctx, user.ID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test DataStreamPaging", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		category := model.Category{
			Name:    "test-cat",
			Purpose: "",
			Values:  []string{"v1", "v2"},
		}
		_, _, err := createCategory(netClient, &category, token)
		require.NoError(t, err)
		categoryID := category.ID

		cloudcreds := model.CloudCreds{
			Name:        "aws-cloud-creds-name",
			Type:        "AWS",
			Description: "aws-cloud-creds-desc",
			AWSCredential: &model.AWSCredential{
				AccessKey: "foo",
				Secret:    "bar",
			},
			GCPCredential: nil,
		}
		_, _, err = createCloudCreds(netClient, &cloudcreds, token)
		require.NoError(t, err)
		cloudCredsID := cloudcreds.ID

		dockerProfileName := "aws-registry-name"
		dockerProfileDesc := "aws-registry-desc"

		// DockerProfile object, leave ID blank and let create generate it
		dockerprofile := model.DockerProfile{
			Name:         dockerProfileName,
			Type:         "AWS",
			Server:       "a.b.c.d.e.f",
			CloudCredsID: cloudCredsID,
			Description:  dockerProfileDesc,
			UserName:     "username",
			Email:        "test@example.com",
			Credentials:  "{\"AccessKeyId\":\"AWS-Access\",\"SecretAccessKey\":\"AWS-SecretAccessKey\",\"Account\":\"AWS-test\",\"Region\":\"us-west-2\",\"Server\":\"aws-server\",\"User\":\"aws-user\",\"Pwd\":\"aws-pwd\",\"Email\":\"aws-email\"}",
		}
		_, _, err = createDockerProfile(netClient, &dockerprofile, token)
		require.NoError(t, err)
		dockerProfileID := dockerprofile.ID

		project := makeExplicitProject(tenantID, []string{cloudcreds.ID}, []string{dockerprofile.ID}, []string{user.ID}, nil)
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		projectID := project.ID

		dockerfile := "docker file"
		scriptruntime := model.ScriptRuntime{
			ScriptRuntimeCore: model.ScriptRuntimeCore{
				Name:            "script-runtime-name",
				Description:     "script runtime desc",
				Language:        "python",
				Builtin:         false,
				DockerRepoURI:   "docker-repo-uri",
				DockerProfileID: dockerProfileID,
				Dockerfile:      dockerfile,
			},
			ProjectID: projectID,
		}

		_, _, err = createScriptRuntime(netClient, &scriptruntime, token)
		require.NoError(t, err)

		scriptName := "script name"
		scriptType := "Transformation"
		scriptLanguage := "Python"
		scriptEnvrionment := "python tensorflow"
		scriptCode := "def main: print"
		// scriptCodeUpdated := "def main: print 'hello'"

		// Script object, leave ID blank and let create generate it
		script := model.Script{
			ScriptCore: model.ScriptCore{
				Name:        scriptName,
				Type:        scriptType,
				Language:    scriptLanguage,
				Environment: scriptEnvrionment,
				Code:        scriptCode,
				Builtin:     false,
				ProjectID:   projectID,
				RuntimeID:   scriptruntime.ID,
			},
			Params: []model.ScriptParam{},
		}

		_, _, err = createScript(netClient, &script, token)
		require.NoError(t, err)

		// randomly create some datastreams
		n := 1 + rand1.Intn(11)
		t.Logf("creating %d datastreams...", n)
		for i := 0; i < n; i++ {
			var size float64 = 1000000
			dataStreamName := fmt.Sprintf("data-streams-name-%s", base.GetUUID())
			dataStreamDataType := "Custom"
			// dataStreamDataTypeUpdated := "Image"

			datastream := model.DataStream{
				Name:     dataStreamName,
				DataType: dataStreamDataType,
				Origin:   "DataSource",
				OriginSelectors: []model.CategoryInfo{
					{
						ID:    categoryID,
						Value: "v1",
					},
				},
				OriginID:         "",
				Destination:      "Cloud",
				CloudType:        "AWS",
				CloudCredsID:     cloudCredsID,
				AWSCloudRegion:   "us-west-2",
				GCPCloudRegion:   "",
				EdgeStreamType:   "",
				AWSStreamType:    "Kafka",
				GCPStreamType:    "",
				Size:             size,
				EnableSampling:   false,
				SamplingInterval: 0,
				TransformationArgsList: []model.TransformationArgs{
					{
						TransformationID: script.ID,
						Args:             []model.ScriptParamValue{},
					},
				},
				DataRetention: []model.RetentionInfo{},
				ProjectID:     projectID,
			}

			_, _, err = createDataStream(netClient, &datastream, token)
			require.NoError(t, err)
		}

		datastreams, err := getDataStreams(netClient, token)
		require.NoError(t, err)
		if len(datastreams) != n {
			t.Fatalf("expected datastreams count to be %d, but got %d", n, len(datastreams))
		}
		sort.Sort(model.DataStreamsByID(datastreams))

		pageSize := 1 + rand1.Intn(n)
		nPages := (n + pageSize - 1) / pageSize
		dss := []model.DataStream{}
		nRemain := n
		t.Logf("fetch %d datastreams using paging api with page size %d, %d pages total...", n, pageSize, nPages)
		for i := 0; i < nPages; i++ {
			ndss, err := getDataStreamsNew(netClient, token, i, pageSize)
			require.NoError(t, err)
			if ndss.PageIndex != i {
				t.Fatalf("expected page index to be %d, but got %d", i, ndss.PageIndex)
			}
			if ndss.PageSize != pageSize {
				t.Fatalf("expected page size to be %d, but got %d", pageSize, ndss.PageSize)
			}
			if ndss.TotalCount != n {
				t.Fatalf("expected total count to be %d, but got %d", n, ndss.TotalCount)
			}
			nexp := nRemain
			if nexp > pageSize {
				nexp = pageSize
			}
			if len(ndss.DataStreamList) != nexp {
				t.Fatalf("expected result count to be %d, but got %d", nexp, len(ndss.DataStreamList))
			}
			nRemain -= pageSize
			for _, app := range ndss.DataStreamList {
				dss = append(dss, app)
			}
		}

		// verify paging api gives same result as old api
		for i := range dss {
			if !reflect.DeepEqual(datastreams[i], dss[i]) {
				t.Fatalf("expect datastream equal, but %+v != %+v", datastreams[i], dss[i])
			}
		}
		t.Log("get datastreams from paging api gives same result as old api")

		// delete datastreams
		for _, ds := range datastreams {
			resp, _, err := deleteDataStream(netClient, ds.ID, token)
			require.NoError(t, err)
			if resp.ID != ds.ID {
				t.Fatal("delete datastream id mismatch")
			}
		}

		resp, _, err := deleteProject(netClient, projectID, token)
		require.NoError(t, err)
		if resp.ID != projectID {
			t.Fatal("delete project id mismatch")
		}

		resp, _, err = deleteCloudCreds(netClient, cloudCredsID, token)
		require.NoError(t, err)
		if resp.ID != cloudCredsID {
			t.Fatal("delete cloud creds id mismatch")
		}

		resp, _, err = deleteCategory(netClient, categoryID, token)
		require.NoError(t, err)
		if resp.ID != categoryID {
			t.Fatal("delete category id mismatch")
		}

	})

}
