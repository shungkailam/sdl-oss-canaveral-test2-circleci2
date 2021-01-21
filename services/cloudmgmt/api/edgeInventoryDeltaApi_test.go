package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func verifyCategories(t *testing.T, created []model.EntityVersionMetadata, resp []model.Category) {
	if len(created) != len(resp) {
		t.Fatalf("expect category count to match: %d vs %d", len(created), len(resp))
	}
	m := map[string]*model.EntityVersionMetadata{}
	for i := range created {
		m[created[i].ID] = &created[i]
	}
	for _, c := range resp {
		evm := m[c.ID]
		if evm == nil || evm.UpdatedAt != c.UpdatedAt {
			t.Fatalf("expect category to match: %+v vs %+v", *evm, c)
		}
	}
}
func verifyProjects(t *testing.T, created []model.EntityVersionMetadata, resp []model.Project) {
	if len(created) != len(resp) {
		t.Fatalf("expect project count to match: %d vs %d", len(created), len(resp))
	}
	m := map[string]*model.EntityVersionMetadata{}
	for i := range created {
		m[created[i].ID] = &created[i]
	}
	for _, c := range resp {
		evm := m[c.ID]
		if evm == nil || evm.UpdatedAt != c.UpdatedAt {
			t.Fatalf("expect project to match: %+v vs %+v", *evm, c)
		}
	}
}
func verifyCloudProfiles(t *testing.T, created []model.EntityVersionMetadata, resp []model.CloudCreds) {
	if len(created) != len(resp) {
		t.Fatalf("expect cloud profile count to match: %d vs %d", len(created), len(resp))
	}
	m := map[string]*model.EntityVersionMetadata{}
	for i := range created {
		m[created[i].ID] = &created[i]
	}
	for _, c := range resp {
		evm := m[c.ID]
		if evm == nil || evm.UpdatedAt != c.UpdatedAt {
			t.Fatalf("expect cloud profile to match: %+v vs %+v", *evm, c)
		}
	}
}
func verifyContainerRegistries(t *testing.T, created []model.EntityVersionMetadata, resp []model.ContainerRegistry) {
	if len(created) != len(resp) {
		t.Fatalf("expect container registry count to match: %d vs %d", len(created), len(resp))
	}
	m := map[string]*model.EntityVersionMetadata{}
	for i := range created {
		m[created[i].ID] = &created[i]
	}
	for _, c := range resp {
		evm := m[c.ID]
		if evm == nil || evm.UpdatedAt != c.UpdatedAt {
			t.Fatalf("expect container registry to match: %+v vs %+v", *evm, c)
		}
	}
}
func verifyScripts(t *testing.T, created []model.EntityVersionMetadata, resp []model.Script) {
	if len(created) != len(resp) {
		t.Fatalf("expect script count to match: %d vs %d", len(created), len(resp))
	}
	m := map[string]*model.EntityVersionMetadata{}
	for i := range created {
		m[created[i].ID] = &created[i]
	}
	for _, c := range resp {
		evm := m[c.ID]
		if evm == nil || evm.UpdatedAt != c.UpdatedAt {
			t.Fatalf("expect script to match: %+v vs %+v", *evm, c)
		}
	}
}
func verifyScriptRuntimes(t *testing.T, created []model.EntityVersionMetadata, resp []model.ScriptRuntime) {
	if len(created) != len(resp) {
		t.Fatalf("expect script runtime count to match: %d vs %d", len(created), len(resp))
	}
	m := map[string]*model.EntityVersionMetadata{}
	for i := range created {
		m[created[i].ID] = &created[i]
	}
	for _, c := range resp {
		evm := m[c.ID]
		if evm == nil || evm.UpdatedAt != c.UpdatedAt {
			t.Fatalf("expect script runtime to match: %+v vs %+v", *evm, c)
		}
	}
}
func verifyMLModels(t *testing.T, created []model.EntityVersionMetadata, resp []model.MLModel) {
	if len(created) != len(resp) {
		t.Fatalf("expect ml model count to match: %d vs %d", len(created), len(resp))
	}
	m := map[string]*model.EntityVersionMetadata{}
	for i := range created {
		m[created[i].ID] = &created[i]
	}
	for _, c := range resp {
		evm := m[c.ID]
		if evm == nil || evm.UpdatedAt != c.UpdatedAt {
			t.Fatalf("expect ml model to match: %+v vs %+v", *evm, c)
		}
	}
}
func verifyDataStreams(t *testing.T, created []model.EntityVersionMetadata, resp []model.DataStream) {
	if len(created) != len(resp) {
		t.Fatalf("expect data stream count to match: %d vs %d", len(created), len(resp))
	}
	m := map[string]*model.EntityVersionMetadata{}
	for i := range created {
		m[created[i].ID] = &created[i]
	}
	for _, c := range resp {
		evm := m[c.ID]
		if evm == nil || evm.UpdatedAt != c.UpdatedAt {
			t.Fatalf("expect data stream to match: %+v vs %+v", *evm, c)
		}
	}
}
func verifyApplications(t *testing.T, created []model.EntityVersionMetadata, resp []model.Application) {
	if len(created) != len(resp) {
		t.Fatalf("expect application count to match: %d vs %d", len(created), len(resp))
	}
	m := map[string]*model.EntityVersionMetadata{}
	for i := range created {
		m[created[i].ID] = &created[i]
	}
	for _, c := range resp {
		evm := m[c.ID]
		if evm == nil || evm.UpdatedAt != c.UpdatedAt {
			t.Fatalf("expect application to match: %+v vs %+v", *evm, c)
		}
	}
}
func verifyDataSources(t *testing.T, created []model.EntityVersionMetadata, resp []model.DataSource) {
	if len(created) != len(resp) {
		t.Fatalf("expect data source count to match: %d vs %d", len(created), len(resp))
	}
	m := map[string]*model.EntityVersionMetadata{}
	for i := range created {
		m[created[i].ID] = &created[i]
	}
	for _, c := range resp {
		evm := m[c.ID]
		if evm == nil || evm.UpdatedAt != c.UpdatedAt {
			t.Fatalf("expect data source to match: %+v vs %+v", *evm, c)
		}
	}
}
func verifyLogCollectors(t *testing.T, created []model.EntityVersionMetadata, resp []model.LogCollector) {
	if len(created) != len(resp) {
		t.Fatalf("expect log collector count to match: %d vs %d", len(created), len(resp))
	}
	m := map[string]*model.EntityVersionMetadata{}
	for i := range created {
		m[created[i].ID] = &created[i]
	}
	for _, c := range resp {
		evm := m[c.ID]
		if evm == nil {
			t.Fatalf("expect log collector to match: NIL vs %+v", c)
		} else if evm.UpdatedAt != c.UpdatedAt {
			t.Fatalf("expect log collector to match: %+v vs %+v", *evm, c)
		}
	}
}

func verifyServiceInstances(t *testing.T, created []model.EntityVersionMetadata, resp []model.ServiceInstance) {
	if len(created) != len(resp) {
		t.Fatalf("expect service instance count to match: %d vs %d", len(created), len(resp))
	}
	m := map[string]*model.EntityVersionMetadata{}
	for i := range created {
		m[created[i].ID] = &created[i]
	}
	for _, c := range resp {
		evm := m[c.ID]
		if evm == nil {
			t.Fatalf("expect service instance to match: NIL vs %+v", c)
		} else if evm.UpdatedAt != c.UpdatedAt {
			t.Fatalf("expect service instance to match: %+v vs %+v", *evm, c)
		}
	}
}

func verifyServiceBindings(t *testing.T, created []model.EntityVersionMetadata, resp []model.ServiceBinding) {
	if len(created) != len(resp) {
		t.Fatalf("expect service binding count to match: %d vs %d", len(created), len(resp))
	}
	m := map[string]*model.EntityVersionMetadata{}
	for i := range created {
		m[created[i].ID] = &created[i]
	}
	for _, c := range resp {
		evm := m[c.ID]
		if evm == nil {
			t.Fatalf("expect service binding to match: NIL vs %+v", c)
		} else if evm.UpdatedAt != c.UpdatedAt {
			t.Fatalf("expect service binding to match: %+v vs %+v", *evm, c)
		}
	}
}

func verifyDataDriverInstances(t *testing.T, created []model.EntityVersionMetadata, resp []model.DataDriverInstanceInventory) {
	if len(created) != len(resp) {
		t.Fatalf("expect data driver inventory to match: %d vs %d", len(created), len(resp))
	}
	m := map[string]*model.EntityVersionMetadata{}
	for i := range created {
		m[created[i].ID] = &created[i]
	}
	for _, c := range resp {
		evm := m[c.ID]
		if evm == nil {
			t.Fatalf("expect data driver instances to match: NIL vs %+v", c)
		} else if evm.UpdatedAt != c.UpdatedAt {
			t.Fatalf("expect data driver instances to match: %+v vs %+v", *evm, c)
		}
	}
}

func verifyCreated(t *testing.T, created *model.EdgeInventoryDeltaPayload, resp *model.EdgeInventoryDeltaResponse) {
	if api.SEND_CATEGORIES_DELTA {
		verifyCategories(t, created.Categories, resp.Created.Categories)
	}
	verifyProjects(t, created.Projects, resp.Created.Projects)
	verifyCloudProfiles(t, created.CloudProfiles, resp.Created.CloudProfiles)
	verifyContainerRegistries(t, created.ContainerRegistries, resp.Created.ContainerRegistries)
	verifyScripts(t, created.Functions, resp.Created.Functions)
	verifyScriptRuntimes(t, created.RuntimeEnvironments, resp.Created.RuntimeEnvironments)
	verifyMLModels(t, created.MLModels, resp.Created.MLModels)
	verifyDataStreams(t, created.DataPipelines, resp.Created.DataPipelines)
	verifyApplications(t, created.Applications, resp.Created.Applications)
	verifyDataSources(t, created.DataSources, resp.Created.DataSources)
	verifyLogCollectors(t, created.LogCollectors, resp.Created.LogCollectors)
	verifyServiceInstances(t, created.SvcInstances, resp.Created.SvcInstances)
	verifyServiceBindings(t, created.SvcBindings, resp.Created.SvcBindings)
	verifyDataDriverInstances(t, created.DataDriverInstances, resp.Created.DataDriverInstances)
}

func verifyUpdated(t *testing.T, updated *model.EdgeInventoryDeltaPayload, resp *model.EdgeInventoryDeltaResponse) {
	if api.SEND_CATEGORIES_DELTA {
		verifyCategories(t, updated.Categories, resp.Updated.Categories)
	}
	verifyProjects(t, updated.Projects, resp.Updated.Projects)
	verifyCloudProfiles(t, updated.CloudProfiles, resp.Updated.CloudProfiles)
	verifyContainerRegistries(t, updated.ContainerRegistries, resp.Updated.ContainerRegistries)
	verifyScripts(t, updated.Functions, resp.Updated.Functions)
	verifyScriptRuntimes(t, updated.RuntimeEnvironments, resp.Updated.RuntimeEnvironments)
	verifyMLModels(t, updated.MLModels, resp.Updated.MLModels)
	verifyDataStreams(t, updated.DataPipelines, resp.Updated.DataPipelines)
	verifyApplications(t, updated.Applications, resp.Updated.Applications)
	verifyDataSources(t, updated.DataSources, resp.Updated.DataSources)
	verifyLogCollectors(t, updated.LogCollectors, resp.Updated.LogCollectors)
	verifyServiceInstances(t, updated.SvcInstances, resp.Updated.SvcInstances)
	verifyServiceBindings(t, updated.SvcBindings, resp.Updated.SvcBindings)
	verifyDataDriverInstances(t, updated.DataDriverInstances, resp.Updated.DataDriverInstances)
}

func verifyDeleted(t *testing.T, deleted *model.EdgeInventoryDeleted, resp *model.EdgeInventoryDeltaResponse) {
	sort.Strings(deleted.Applications)
	sort.Strings(deleted.Categories)
	sort.Strings(deleted.CloudProfiles)
	sort.Strings(deleted.ContainerRegistries)
	sort.Strings(deleted.DataPipelines)
	sort.Strings(deleted.DataSources)
	sort.Strings(deleted.Functions)
	sort.Strings(deleted.MLModels)
	sort.Strings(deleted.Projects)
	sort.Strings(deleted.RuntimeEnvironments)
	sort.Strings(deleted.LogCollectors)
	sort.Strings(deleted.SvcInstances)
	sort.Strings(deleted.SvcBindings)
	sort.Strings(deleted.DataDriverInstances)
	sort.Strings(resp.Deleted.Applications)
	sort.Strings(resp.Deleted.Categories)
	sort.Strings(resp.Deleted.CloudProfiles)
	sort.Strings(resp.Deleted.ContainerRegistries)
	sort.Strings(resp.Deleted.DataPipelines)
	sort.Strings(resp.Deleted.DataSources)
	sort.Strings(resp.Deleted.Functions)
	sort.Strings(resp.Deleted.MLModels)
	sort.Strings(resp.Deleted.Projects)
	sort.Strings(resp.Deleted.RuntimeEnvironments)
	sort.Strings(resp.Deleted.LogCollectors)
	sort.Strings(resp.Deleted.SvcInstances)
	sort.Strings(resp.Deleted.SvcBindings)
	sort.Strings(resp.Deleted.DataDriverInstances)
	if !reflect.DeepEqual(deleted.Applications, resp.Deleted.Applications) {
		t.Fatal("expect deleted Applications to be equal")
	}
	if api.SEND_CATEGORIES_DELTA {
		if !reflect.DeepEqual(deleted.Categories, resp.Deleted.Categories) {
			t.Fatal("expect deleted Categories to be equal")
		}
	}
	if !reflect.DeepEqual(deleted.CloudProfiles, resp.Deleted.CloudProfiles) {
		t.Fatal("expect deleted CloudProfiles to be equal")
	}
	if !reflect.DeepEqual(deleted.ContainerRegistries, resp.Deleted.ContainerRegistries) {
		t.Fatal("expect deleted ContainerRegistries to be equal")
	}
	if !reflect.DeepEqual(deleted.DataPipelines, resp.Deleted.DataPipelines) {
		t.Fatal("expect deleted DataPipelines to be equal")
	}
	if !reflect.DeepEqual(deleted.DataSources, resp.Deleted.DataSources) {
		t.Fatal("expect deleted DataSources to be equal")
	}
	if !reflect.DeepEqual(deleted.Functions, resp.Deleted.Functions) {
		t.Fatal("expect deleted Functions to be equal")
	}
	if !reflect.DeepEqual(deleted.MLModels, resp.Deleted.MLModels) {
		t.Fatal("expect deleted MLModels to be equal")
	}
	if !reflect.DeepEqual(deleted.Projects, resp.Deleted.Projects) {
		t.Fatal("expect deleted Projects to be equal")
	}
	if !reflect.DeepEqual(deleted.RuntimeEnvironments, resp.Deleted.RuntimeEnvironments) {
		t.Fatal("expect deleted RuntimeEnvironments to be equal")
	}
	if !reflect.DeepEqual(deleted.LogCollectors, resp.Deleted.LogCollectors) {
		t.Fatal("expect deleted LogCollectors to be equal")
	}
	if !reflect.DeepEqual(deleted.SvcInstances, resp.Deleted.SvcInstances) {
		t.Fatal("expect deleted ServiceInstances to be equal")
	}
	if !reflect.DeepEqual(deleted.SvcBindings, resp.Deleted.SvcBindings) {
		t.Fatal("expect deleted ServiceBindings to be equal")
	}
	if !reflect.DeepEqual(deleted.DataDriverInstances, resp.Deleted.DataDriverInstances) {
		t.Fatal("expect deleted DataDriverInstances to be equal")
	}
}

func parallel(t *testing.T, wg *sync.WaitGroup, name string, f func()) {
	wg.Add(1)
	go func() {
		t1 := time.Now()
		f()
		wg.Done()
		t2 := time.Now()
		t.Logf("Operation %s took %v", name, t2.Sub(t1))
	}()
}

func TestEdgeInventoryDelta(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup

	// Setup
	dbAPI := newObjectModelAPI(t)

	// Create tenant
	tenant := createTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID

	category := make(map[int]model.Category)
	parallel(t, &wg, "create categories", func() {
		category[0] = createCategory(t, dbAPI, tenantID)
		category[2] = createCategoryCommon(t, dbAPI, tenantID, "test-cat-2", []string{"v21", "v22", "v23"})
		category[3] = createCategoryCommon(t, dbAPI, tenantID, "test-cat-3", []string{"v31", "v32"})
		category[4] = createCategoryCommon(t, dbAPI, tenantID, "test-cat-4", []string{"v41", "v42", "v43"})
	})

	// Wait for object creation
	wg.Wait()

	edge := make(map[int]model.Edge)
	parallel(t, &wg, "create edge 1", func() {
		edge[0] = createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{
			{
				ID:    category[0].ID,
				Value: TestCategoryValue1,
			},
		})
		setEdgeDeviceVersion(t, dbAPI, tenantID, edge[0].ID, "v2.0.0")
	})
	parallel(t, &wg, "create edge 2", func() {
		edge[2] = createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{
			{
				ID:    category[0].ID,
				Value: TestCategoryValue2,
			},
		})
		setEdgeDeviceVersion(t, dbAPI, tenantID, edge[2].ID, "v2.0.0")
	})
	parallel(t, &wg, "create edge 3", func() {
		edge[3] = createEdgeWithLabels(t, dbAPI, tenantID, nil)
		setEdgeDeviceVersion(t, dbAPI, tenantID, edge[3].ID, "v2.0.0")
	})

	var user model.User
	parallel(t, &wg, "create user", func() {
		user = createUser(t, dbAPI, tenantID)
	})

	// create cloud creds
	cloudCreds := make(map[int]model.CloudCreds)
	parallel(t, &wg, "create cloud credentials", func() {
		cloudCreds[0] = createCloudCreds(t, dbAPI, tenantID)
		cloudCreds[2] = createCloudCreds(t, dbAPI, tenantID)
		cloudCreds[3] = createCloudCreds(t, dbAPI, tenantID)
		cloudCreds[4] = createCloudCreds(t, dbAPI, tenantID) // not associated
		cloudCreds[5] = createCloudCreds(t, dbAPI, tenantID) // not associated
	})

	// Wait for object creation
	wg.Wait()

	// create docker profile
	dockerProfile := make(map[int]model.ContainerRegistry)

	dockerProfile[0] = createAWSContainerRegistry(t, dbAPI, tenantID, cloudCreds[0].ID)
	dockerProfile[2] = createAWSContainerRegistry(t, dbAPI, tenantID, cloudCreds[2].ID)
	dockerProfile[3] = createAWSContainerRegistry(t, dbAPI, tenantID, cloudCreds[3].ID)

	// project is cat/v1
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{cloudCreds[0].ID}, []string{dockerProfile[0].ID}, []string{user.ID}, []model.CategoryInfo{
		{
			ID:    category[0].ID,
			Value: TestCategoryValue1,
		},
	})
	projectID := project.ID
	project2 := createExplicitProjectCommon(t, dbAPI, tenantID, []string{cloudCreds[2].ID}, []string{dockerProfile[2].ID}, nil, []string{edge[2].ID})
	project3 := createCategoryProjectCommon(t, dbAPI, tenantID, []string{cloudCreds[3].ID}, []string{dockerProfile[3].ID}, []string{}, []model.CategoryInfo{
		{
			ID:    category[0].ID,
			Value: TestCategoryValue2,
		},
	})
	project4 := createExplicitProjectCommon(t, dbAPI, tenantID, []string{cloudCreds[0].ID, cloudCreds[2].ID}, []string{dockerProfile[0].ID, dockerProfile[2].ID}, nil, []string{edge[0].ID})
	// category based project w/o category assignment should not contain any edges
	project5 := createCategoryProjectCommon(t, dbAPI, tenantID, []string{cloudCreds[3].ID}, []string{dockerProfile[3].ID}, []string{}, nil)
	project6 := createExplicitProjectCommon(t, dbAPI, tenantID, []string{cloudCreds[0].ID}, []string{dockerProfile[0].ID}, nil, []string{edge[0].ID, edge[2].ID})

	ctx, _, _ := makeContext(tenantID, []string{projectID, project2.ID, project3.ID, project4.ID})

	// create script runtime & script & data stream & data source
	dataStream := make(map[int]model.DataStream)
	dataSource := make(map[int]model.DataSource)
	scriptRuntime := make(map[int]model.ScriptRuntime)
	script := make(map[int]model.Script)
	parallel(t, &wg, "create data streams", func() {
		scriptRuntime[0] = createScriptRuntime(t, dbAPI, tenantID, projectID, dockerProfile[0].ID)
		scriptRuntime[2] = createScriptRuntime(t, dbAPI, tenantID, project2.ID, dockerProfile[2].ID)
		scriptRuntime[3] = createScriptRuntime(t, dbAPI, tenantID, project3.ID, dockerProfile[3].ID)
		scriptRuntime[4] = createScriptRuntime(t, dbAPI, tenantID, project4.ID, dockerProfile[0].ID)
		scriptRuntime[5] = createScriptRuntime(t, dbAPI, tenantID, project4.ID, dockerProfile[2].ID)
		scriptRuntime[6] = createScriptRuntime(t, dbAPI, tenantID, project6.ID, dockerProfile[0].ID)

		script[0] = createScript(t, dbAPI, tenantID, projectID, scriptRuntime[0].ID)
		script[2] = createScript(t, dbAPI, tenantID, project2.ID, scriptRuntime[2].ID)
		script[3] = createScript(t, dbAPI, tenantID, project3.ID, scriptRuntime[3].ID)
		script[4] = createScript(t, dbAPI, tenantID, project4.ID, scriptRuntime[4].ID)
		script[5] = createScript(t, dbAPI, tenantID, project4.ID, scriptRuntime[5].ID)
		script[6] = createScript(t, dbAPI, tenantID, project6.ID, scriptRuntime[6].ID)

		dataStream[0] = createDataStream(t, dbAPI, tenantID, projectID, category[0].ID, "v1", cloudCreds[0].ID, script[0].ID)
		dataStream[2] = createDataStream(t, dbAPI, tenantID, project2.ID, category[2].ID, "v21", cloudCreds[2].ID, script[2].ID)
		dataStream[3] = createDataStream(t, dbAPI, tenantID, project3.ID, category[3].ID, "v31", cloudCreds[3].ID, script[3].ID)
		dataStream[4] = createDataStream(t, dbAPI, tenantID, project4.ID, category[4].ID, "v41", cloudCreds[0].ID, script[4].ID)
		dataStream[5] = createDataStreamWithState(t, dbAPI, tenantID, projectID, category[0].ID, "v1", cloudCreds[0].ID, script[0].ID, model.UndeployEntityState.StringPtr())
		dataStream[6] = createDataStreamWithState(t, dbAPI, tenantID, project2.ID, category[2].ID, "v21", cloudCreds[2].ID, script[2].ID, model.UndeployEntityState.StringPtr())
		dataSource[5], dataStream[7] = createDataStreamWithOutIfc(t, dbAPI, tenantID, project6.ID, category[0].ID, "v1", cloudCreds[0].ID, script[6].ID, edge[0].ID)
		dataSource[6], dataStream[8] = createDataStreamWithOutIfc(t, dbAPI, tenantID, project6.ID, category[0].ID, "v1", cloudCreds[0].ID, script[6].ID, edge[3].ID)
	})

	// and data sources
	parallel(t, &wg, "create data sources", func() {
		dataSource[0] = createDataSource(t, dbAPI, tenantID, edge[0].ID, category[0].ID, "v1")
		dataSource[2] = createDataSource(t, dbAPI, tenantID, edge[2].ID, category[2].ID, "v21")
		dataSource[3] = createDataSource(t, dbAPI, tenantID, edge[0].ID, category[0].ID, "v1")
		dataSource[4] = createDataSource(t, dbAPI, tenantID, edge[2].ID, category[4].ID, "v41")
	})

	// create ML model
	mlModel := make(map[int]model.MLModel)
	parallel(t, &wg, "create ML", func() {
		mlModel[0] = createMLModel(t, dbAPI, tenantID, projectID)
		mlModel[2] = createMLModel(t, dbAPI, tenantID, project2.ID)
		mlModel[3] = createMLModel(t, dbAPI, tenantID, project3.ID)
		mlModel[4] = createMLModel(t, dbAPI, tenantID, project4.ID)
	})

	// create application
	application := make(map[int]model.Application)
	parallel(t, &wg, "create applications 1", func() {
		application[0] = createApplication(t, dbAPI, tenantID, "app-name", project4.ID, []string{edge[0].ID}, nil)
	})
	parallel(t, &wg, "create applications 2", func() {
		application[2] = createApplication(t, dbAPI, tenantID, "app-name-2", project4.ID, nil, nil)
	})
	parallel(t, &wg, "create applications 3", func() {
		application[3] = createApplication(t, dbAPI, tenantID, "app-name-3", project.ID, nil, nil)
	})
	parallel(t, &wg, "create applications 4", func() {
		application[4] = createApplication(t, dbAPI, tenantID, "app-name-4", project.ID, nil, []model.CategoryInfo{{ID: category[2].ID, Value: "v21"}})
	})
	parallel(t, &wg, "create applications 5", func() {
		application[5] = createApplication(t, dbAPI, tenantID, "app-name-5", project2.ID, []string{edge[2].ID}, nil)
	})
	parallel(t, &wg, "create applications 6", func() {
		application[6] = createApplication(t, dbAPI, tenantID, "app-name-6", project2.ID, nil, nil)
	})
	parallel(t, &wg, "create applications 7", func() {
		application[7] = createApplication(t, dbAPI, tenantID, "app-name-7", project3.ID, nil, nil)
	})
	parallel(t, &wg, "create applications 8", func() {
		application[8] = createApplication(t, dbAPI, tenantID, "app-name-8", project3.ID, nil, []model.CategoryInfo{{ID: category[2].ID, Value: "v21"}})
	})
	parallel(t, &wg, "create applications 9", func() {
		application[9] = createApplicationWithState(t, dbAPI, tenantID, "app-name-9", project4.ID, []string{edge[0].ID}, nil, model.UndeployEntityState.StringPtr())
	})
	parallel(t, &wg, "create applications 10", func() {
		application[10] = createApplicationWithState(t, dbAPI, tenantID, "app-name-10", project.ID, nil, nil, model.UndeployEntityState.StringPtr())
	})
	parallel(t, &wg, "create applications 11", func() {
		application[11] = createApplicationWithState(t, dbAPI, tenantID, "app-name-11", project2.ID, []string{edge[2].ID}, nil, model.UndeployEntityState.StringPtr())
	})
	parallel(t, &wg, "create applications 12", func() {
		application[12] = createApplicationWithState(t, dbAPI, tenantID, "app-name-12", project3.ID, nil, nil, model.UndeployEntityState.StringPtr())
	})

	// create project services
	projectService := make(map[int]model.ProjectService)
	parallel(t, &wg, "create project services", func() {
		projectService[0] = createProjectService(t, dbAPI, tenantID, project.ID)
		projectService[2] = createProjectService(t, dbAPI, tenantID, project2.ID)
		projectService[3] = createProjectService(t, dbAPI, tenantID, project3.ID)
		projectService[4] = createProjectService(t, dbAPI, tenantID, project4.ID)
		projectService[5] = createProjectService(t, dbAPI, tenantID, project4.ID)
		projectService[6] = createProjectService(t, dbAPI, tenantID, project4.ID)
	})

	// create log collectors
	logCollector := make(map[int]model.LogCollector)
	parallel(t, &wg, "create log collectors", func() {
		logCollector[0] = createLogCollector(t, dbAPI, tenantID, cloudCreds[0].ID)                        // on all edges (infra)
		logCollector[2] = createLogCollector(t, dbAPI, tenantID, cloudCreds[0].ID)                        // on all edges (infra)
		logCollector[3] = createLogCollectorForProject(t, dbAPI, tenantID, project3.ID, cloudCreds[0].ID) // On edge 2 only via category
		logCollector[4] = createLogCollectorForProject(t, dbAPI, tenantID, project4.ID, cloudCreds[0].ID) // On edge, 1 directly
		logCollector[5] = createLogCollectorForProject(t, dbAPI, tenantID, project4.ID, cloudCreds[4].ID) // On edge, 1 directly
		logCollector[6] = createLogCollectorForProject(t, dbAPI, tenantID, project5.ID, cloudCreds[5].ID) // No edges assigned
	})

	// create service classes and bindings
	svcClass := make(map[int]*model.ServiceClass)
	svcInstance := make(map[int]*model.ServiceInstance)
	svcBinding := make(map[int]*model.ServiceBinding)
	parallel(t, &wg, "create services", func() {
		svcClass[0] = createServiceClass(t, dbAPI)
		svcClass[2] = createServiceClass(t, dbAPI)
		svcClass[3] = createServiceClass(t, dbAPI)

		svcInstance[0] = createServiceInstance(ctx, t, dbAPI, tenantID, svcClass[0].ID, project.ID)  // On edge 1
		svcInstance[2] = createServiceInstance(ctx, t, dbAPI, tenantID, svcClass[2].ID, project4.ID) // On edge 1
		svcInstance[3] = createServiceInstance(ctx, t, dbAPI, tenantID, svcClass[3].ID, project3.ID) // On edge 2

		svcBinding[0] = createServiceBinding(ctx, t, dbAPI, tenantID, svcClass[0].ID, "", project4.ID) // On edge 1
		svcBinding[2] = createServiceBinding(ctx, t, dbAPI, tenantID, svcClass[2].ID, "", project.ID)  // On edge 1
		svcBinding[3] = createServiceBinding(ctx, t, dbAPI, tenantID, svcClass[3].ID, "", project3.ID) // On edge 2
	})

	// create service classes and bindings
	dataDriverClass := make(map[int]model.DataDriverClass)
	dataDriverInstance := make(map[int]model.DataDriverInstance)
	dataDriverConfig := make(map[int]model.DataDriverConfig)
	parallel(t, &wg, "create data drivers", func() {
		dataDriverClass[0] = createDataDriverClass(t, dbAPI, tenantID, "dd-1")

		dataDriverInstance[0] = createDataDriverInstance(t, dbAPI, tenantID, dataDriverClass[0].ID, project.ID)  // on edge 1
		dataDriverInstance[2] = createDataDriverInstance(t, dbAPI, tenantID, dataDriverClass[0].ID, project2.ID) // on edge 2
		dataDriverInstance[3] = createDataDriverInstance(t, dbAPI, tenantID, dataDriverClass[0].ID, project5.ID) // No edges assigned
		dataDriverInstance[4] = createDataDriverInstance(t, dbAPI, tenantID, dataDriverClass[0].ID, project6.ID) // on edges 1 & 2
	})

	// Waiting for object creation
	wg.Wait()

	// Teardown
	defer func() {
		for _, ddc := range dataDriverConfig {
			dbAPI.DeleteDataDriverConfig(ctx, ddc.ID, nil)
		}
		for _, ddi := range dataDriverInstance {
			dbAPI.DeleteDataDriverInstance(ctx, ddi.ID, nil)
		}
		for _, ddc := range dataDriverClass {
			dbAPI.DeleteDataDriverClass(ctx, ddc.ID, nil)
		}
		for _, svcBns := range svcBinding {
			dbAPI.DeleteServiceBinding(ctx, svcBns.ID, nil)
		}
		for _, svcIns := range svcInstance {
			dbAPI.DeleteServiceInstance(ctx, svcIns.ID, nil)
		}
		for _, svcCls := range svcClass {
			dbAPI.DeleteServiceClass(ctx, svcCls.ID, nil)
		}
		for _, lc := range logCollector {
			dbAPI.DeleteLogCollector(ctx, lc.ID, nil)
		}
		for _, ds := range projectService {
			dbAPI.DeleteProjectService(ctx, ds.ID, nil)
		}
		for _, ds := range dataSource {
			dbAPI.DeleteDataSource(ctx, ds.ID, nil)
		}
		for _, app := range application {
			dbAPI.DeleteApplication(ctx, app.ID, nil)
		}
		for _, mdl := range mlModel {
			dbAPI.DeleteMLModel(ctx, mdl.ID, nil)
		}
		for _, ds := range dataStream {
			dbAPI.DeleteDataStream(ctx, ds.ID, nil)
		}
		for _, sc := range script {
			dbAPI.DeleteScript(ctx, sc.ID, nil)
		}
		for _, sr := range scriptRuntime {
			dbAPI.DeleteScriptRuntime(ctx, sr.ID, nil)
		}
		for _, proj := range []model.Project{project, project2, project3, project4, project5, project6} {
			dbAPI.DeleteProject(ctx, proj.ID, nil)
		}
		for _, dp := range dockerProfile {
			dbAPI.DeleteDockerProfile(ctx, dp.ID, nil)
		}
		for _, cc := range cloudCreds {
			dbAPI.DeleteCloudCreds(ctx, cc.ID, nil)
		}
		dbAPI.DeleteUser(ctx, user.ID, nil)
		for _, ed := range edge {
			dbAPI.DeleteEdge(ctx, ed.ID, nil)
		}
		for _, cat := range category {
			dbAPI.DeleteCategory(ctx, cat.ID, nil)
		}
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	bogusCatID := base.GetUUID()
	bogusAppID := base.GetUUID()
	bogusProjID := base.GetUUID()
	bogusCloudProfileID := base.GetUUID()
	bogusContainerRegistryID := base.GetUUID()
	bogusScriptID := base.GetUUID()
	bogusScriptRuntimeID := base.GetUUID()
	bogusMLModelID := base.GetUUID()
	bogusDataStreamID := base.GetUUID()
	bogusDataSourceID := base.GetUUID()
	bogusProjectServiceID := base.GetUUID()
	bogusLogCollectorID := base.GetUUID()
	bogusSvcInstanceID := base.GetUUID()
	bogusSvcBindingID := base.GetUUID()
	bogusDataDriverInstanceID := base.GetUUID()

	var testDataList = []struct {
		name    string
		edgeID  string
		payload *model.EdgeInventoryDeltaPayload
		created *model.EdgeInventoryDeltaPayload
		updated *model.EdgeInventoryDeltaPayload
		deleted *model.EdgeInventoryDeleted
	}{
		{
			"edge1",
			edge[0].ID,
			&model.EdgeInventoryDeltaPayload{
				Categories:          []model.EntityVersionMetadata{{ID: bogusCatID}, {ID: category[3].ID, UpdatedAt: category[3].UpdatedAt.Round(time.Minute)}, {ID: category[4].ID, UpdatedAt: category[4].UpdatedAt.In(time.Local)}},
				Applications:        []model.EntityVersionMetadata{{ID: bogusAppID}, {ID: application[3].ID}, {ID: application[5].ID}, {ID: application[6].ID}, {ID: application[7].ID}, {ID: application[8].ID}, {ID: application[9].ID}},
				Projects:            []model.EntityVersionMetadata{{ID: bogusProjID}, {ID: project4.ID}},
				CloudProfiles:       []model.EntityVersionMetadata{{ID: bogusCloudProfileID}, {ID: cloudCreds[2].ID}},
				ContainerRegistries: []model.EntityVersionMetadata{{ID: bogusContainerRegistryID}, {ID: dockerProfile[2].ID}},
				Functions:           []model.EntityVersionMetadata{{ID: bogusScriptID}, {ID: script[3].ID}, {ID: script[4].ID}, {ID: script[5].ID}},
				RuntimeEnvironments: []model.EntityVersionMetadata{{ID: bogusScriptRuntimeID}, {ID: scriptRuntime[4].ID}, {ID: scriptRuntime[5].ID}},
				MLModels:            []model.EntityVersionMetadata{{ID: bogusMLModelID}, {ID: mlModel[2].ID}, {ID: mlModel[3].ID}, {ID: mlModel[4].ID}},
				DataPipelines:       []model.EntityVersionMetadata{{ID: bogusDataStreamID}, {ID: dataStream[2].ID}, {ID: dataStream[3].ID}, {ID: dataStream[4].ID}, {ID: dataStream[5].ID}},
				DataSources:         []model.EntityVersionMetadata{{ID: bogusDataSourceID}, {ID: dataSource[2].ID}, {ID: dataSource[3].ID}, {ID: dataSource[4].ID}},
				ProjectServices:     []model.EntityVersionMetadata{{ID: bogusProjectServiceID}, {ID: projectService[2].ID}, {ID: projectService[3].ID}, {ID: projectService[4].ID}},
				LogCollectors:       []model.EntityVersionMetadata{{ID: bogusLogCollectorID}, {ID: logCollector[2].ID}, {ID: logCollector[3].ID}, {ID: logCollector[4].ID}, {ID: logCollector[5].ID}, {ID: logCollector[6].ID}},
				SvcInstances:        []model.EntityVersionMetadata{{ID: bogusSvcInstanceID}, {ID: svcInstance[0].ID}, {ID: svcInstance[3].ID}},
				SvcBindings:         []model.EntityVersionMetadata{{ID: bogusSvcBindingID}, {ID: svcBinding[0].ID}, {ID: svcBinding[3].ID}},
				DataDriverInstances: []model.EntityVersionMetadata{{ID: bogusDataDriverInstanceID}, {ID: dataDriverInstance[0].ID}, {ID: dataDriverInstance[3].ID}},
			},
			&model.EdgeInventoryDeltaPayload{
				Categories: []model.EntityVersionMetadata{
					{ID: category[0].ID, UpdatedAt: category[0].UpdatedAt},
					{ID: category[2].ID, UpdatedAt: category[2].UpdatedAt},
				},
				Projects: []model.EntityVersionMetadata{
					{ID: project.ID, UpdatedAt: project.UpdatedAt},
					{ID: project6.ID, UpdatedAt: project6.UpdatedAt},
				},
				CloudProfiles: []model.EntityVersionMetadata{
					{ID: cloudCreds[0].ID, UpdatedAt: cloudCreds[0].UpdatedAt},
					{ID: cloudCreds[4].ID, UpdatedAt: cloudCreds[4].UpdatedAt},
				},
				ContainerRegistries: []model.EntityVersionMetadata{
					{ID: dockerProfile[0].ID, UpdatedAt: dockerProfile[0].UpdatedAt},
				},
				Functions: []model.EntityVersionMetadata{
					{ID: script[0].ID, UpdatedAt: script[0].UpdatedAt},
					{ID: script[6].ID, UpdatedAt: script[6].UpdatedAt},
				},
				RuntimeEnvironments: []model.EntityVersionMetadata{
					{ID: scriptRuntime[0].ID, UpdatedAt: scriptRuntime[0].UpdatedAt},
					{ID: scriptRuntime[6].ID, UpdatedAt: scriptRuntime[6].UpdatedAt},
				},
				MLModels: []model.EntityVersionMetadata{
					{ID: mlModel[0].ID, UpdatedAt: mlModel[0].UpdatedAt},
				},
				DataPipelines: []model.EntityVersionMetadata{
					{ID: dataStream[0].ID, UpdatedAt: dataStream[0].UpdatedAt},
					{ID: dataStream[7].ID, UpdatedAt: dataStream[7].UpdatedAt},
				},
				Applications: []model.EntityVersionMetadata{
					{ID: application[0].ID, UpdatedAt: application[0].UpdatedAt},
				},
				DataSources: []model.EntityVersionMetadata{
					{ID: dataSource[0].ID, UpdatedAt: dataSource[0].UpdatedAt},
					{ID: dataSource[5].ID, UpdatedAt: dataSource[5].UpdatedAt},
				},
				ProjectServices: []model.EntityVersionMetadata{
					{ID: projectService[0].ID, UpdatedAt: projectService[0].UpdatedAt},
					{ID: projectService[5].ID, UpdatedAt: projectService[5].UpdatedAt},
				},
				LogCollectors: []model.EntityVersionMetadata{
					{ID: logCollector[0].ID, UpdatedAt: logCollector[0].UpdatedAt},
				},
				SvcInstances: []model.EntityVersionMetadata{
					{ID: svcInstance[2].ID, UpdatedAt: svcInstance[2].UpdatedAt},
				},
				SvcBindings: []model.EntityVersionMetadata{
					{ID: svcBinding[2].ID, UpdatedAt: svcBinding[2].UpdatedAt},
				},
				DataDriverInstances: []model.EntityVersionMetadata{
					{ID: dataDriverInstance[4].ID, UpdatedAt: dataDriverInstance[4].UpdatedAt},
				},
			},
			&model.EdgeInventoryDeltaPayload{
				Categories: []model.EntityVersionMetadata{
					{ID: category[3].ID, UpdatedAt: category[3].UpdatedAt},
				},
				Projects: []model.EntityVersionMetadata{
					{ID: project4.ID, UpdatedAt: project4.UpdatedAt},
				},
				Applications: []model.EntityVersionMetadata{
					{ID: application[3].ID, UpdatedAt: application[3].UpdatedAt},
				},
				CloudProfiles: []model.EntityVersionMetadata{
					{ID: cloudCreds[2].ID, UpdatedAt: cloudCreds[2].UpdatedAt},
				},
				ContainerRegistries: []model.EntityVersionMetadata{
					{ID: dockerProfile[2].ID, UpdatedAt: dockerProfile[2].UpdatedAt},
				},
				RuntimeEnvironments: []model.EntityVersionMetadata{
					{ID: scriptRuntime[4].ID, UpdatedAt: scriptRuntime[4].UpdatedAt},
					{ID: scriptRuntime[5].ID, UpdatedAt: scriptRuntime[5].UpdatedAt},
				},
				Functions: []model.EntityVersionMetadata{
					{ID: script[4].ID, UpdatedAt: script[4].UpdatedAt},
					{ID: script[5].ID, UpdatedAt: script[5].UpdatedAt},
				},
				DataPipelines: []model.EntityVersionMetadata{
					{ID: dataStream[4].ID, UpdatedAt: dataStream[4].UpdatedAt},
				},
				MLModels: []model.EntityVersionMetadata{
					{ID: mlModel[4].ID, UpdatedAt: mlModel[4].UpdatedAt},
				},
				DataSources: []model.EntityVersionMetadata{
					{ID: dataSource[3].ID, UpdatedAt: dataSource[3].UpdatedAt},
				},
				ProjectServices: []model.EntityVersionMetadata{
					{ID: projectService[3].ID, UpdatedAt: projectService[3].UpdatedAt},
				},
				LogCollectors: []model.EntityVersionMetadata{
					{ID: logCollector[2].ID, UpdatedAt: logCollector[2].UpdatedAt},
					{ID: logCollector[4].ID, UpdatedAt: logCollector[4].UpdatedAt},
					{ID: logCollector[5].ID, UpdatedAt: logCollector[5].UpdatedAt},
				},
				SvcInstances: []model.EntityVersionMetadata{
					{ID: svcInstance[0].ID, UpdatedAt: svcInstance[0].UpdatedAt},
				},
				SvcBindings: []model.EntityVersionMetadata{
					{ID: svcBinding[0].ID, UpdatedAt: svcBinding[0].UpdatedAt},
				},
				DataDriverInstances: []model.EntityVersionMetadata{
					{ID: dataDriverInstance[0].ID, UpdatedAt: dataDriverInstance[0].UpdatedAt},
				},
			},
			&model.EdgeInventoryDeleted{
				Categories:          []string{bogusCatID},
				Applications:        []string{bogusAppID, application[5].ID, application[6].ID, application[7].ID, application[8].ID, application[9].ID},
				Projects:            []string{bogusProjID},
				CloudProfiles:       []string{bogusCloudProfileID},
				ContainerRegistries: []string{bogusContainerRegistryID},
				Functions:           []string{bogusScriptID, script[3].ID},
				RuntimeEnvironments: []string{bogusScriptRuntimeID},
				MLModels:            []string{bogusMLModelID, mlModel[2].ID, mlModel[3].ID},
				DataPipelines:       []string{bogusDataStreamID, dataStream[2].ID, dataStream[3].ID, dataStream[5].ID},
				DataSources:         []string{bogusDataSourceID, dataSource[2].ID, dataSource[4].ID},
				ProjectServices:     []string{bogusProjectServiceID, projectService[2].ID, projectService[4].ID},
				LogCollectors:       []string{bogusLogCollectorID, logCollector[3].ID, logCollector[6].ID},
				SvcInstances:        []string{bogusSvcInstanceID, svcInstance[3].ID},
				SvcBindings:         []string{bogusSvcBindingID, svcBinding[3].ID},
				DataDriverInstances: []string{bogusDataDriverInstanceID, dataDriverInstance[3].ID},
			},
		},
		{
			"edge[2]",
			edge[2].ID,
			&model.EdgeInventoryDeltaPayload{
				Categories:          []model.EntityVersionMetadata{{ID: bogusCatID}, {ID: category[4].ID}},
				Applications:        []model.EntityVersionMetadata{{ID: bogusAppID}, {ID: application[11].ID}},
				Projects:            []model.EntityVersionMetadata{{ID: bogusProjID}},
				CloudProfiles:       []model.EntityVersionMetadata{{ID: bogusCloudProfileID}},
				ContainerRegistries: []model.EntityVersionMetadata{{ID: bogusContainerRegistryID}},
				Functions:           []model.EntityVersionMetadata{{ID: bogusScriptID}},
				RuntimeEnvironments: []model.EntityVersionMetadata{{ID: bogusScriptRuntimeID}},
				MLModels:            []model.EntityVersionMetadata{{ID: bogusMLModelID}},
				DataPipelines:       []model.EntityVersionMetadata{{ID: bogusDataStreamID}},
				DataSources:         []model.EntityVersionMetadata{{ID: bogusDataSourceID}},
				ProjectServices:     []model.EntityVersionMetadata{{ID: bogusProjectServiceID}},
				LogCollectors:       []model.EntityVersionMetadata{{ID: bogusLogCollectorID}},
				SvcInstances:        []model.EntityVersionMetadata{{ID: bogusSvcInstanceID}},
				SvcBindings:         []model.EntityVersionMetadata{{ID: bogusSvcBindingID}},
				DataDriverInstances: []model.EntityVersionMetadata{{ID: bogusDataDriverInstanceID}},
			},
			&model.EdgeInventoryDeltaPayload{
				Categories: []model.EntityVersionMetadata{
					{ID: category[0].ID, UpdatedAt: category[0].UpdatedAt},
					{ID: category[2].ID, UpdatedAt: category[2].UpdatedAt},
					{ID: category[3].ID, UpdatedAt: category[3].UpdatedAt},
				},
				Projects: []model.EntityVersionMetadata{
					{ID: project2.ID, UpdatedAt: project2.UpdatedAt},
					{ID: project3.ID, UpdatedAt: project3.UpdatedAt},
					{ID: project6.ID, UpdatedAt: project6.UpdatedAt},
				},
				CloudProfiles: []model.EntityVersionMetadata{
					{ID: cloudCreds[0].ID, UpdatedAt: cloudCreds[0].UpdatedAt},
					{ID: cloudCreds[2].ID, UpdatedAt: cloudCreds[2].UpdatedAt},
					{ID: cloudCreds[3].ID, UpdatedAt: cloudCreds[3].UpdatedAt},
				},
				ContainerRegistries: []model.EntityVersionMetadata{
					{ID: dockerProfile[0].ID, UpdatedAt: dockerProfile[0].UpdatedAt},
					{ID: dockerProfile[2].ID, UpdatedAt: dockerProfile[2].UpdatedAt},
					{ID: dockerProfile[3].ID, UpdatedAt: dockerProfile[3].UpdatedAt},
				},
				Functions: []model.EntityVersionMetadata{
					{ID: script[2].ID, UpdatedAt: script[2].UpdatedAt},
					{ID: script[3].ID, UpdatedAt: script[3].UpdatedAt},
					{ID: script[6].ID, UpdatedAt: script[6].UpdatedAt},
				},
				RuntimeEnvironments: []model.EntityVersionMetadata{
					{ID: scriptRuntime[2].ID, UpdatedAt: scriptRuntime[2].UpdatedAt},
					{ID: scriptRuntime[3].ID, UpdatedAt: scriptRuntime[3].UpdatedAt},
					{ID: scriptRuntime[6].ID, UpdatedAt: scriptRuntime[6].UpdatedAt},
				},
				MLModels: []model.EntityVersionMetadata{
					{ID: mlModel[2].ID, UpdatedAt: mlModel[2].UpdatedAt},
					{ID: mlModel[3].ID, UpdatedAt: mlModel[3].UpdatedAt},
				},
				DataPipelines: []model.EntityVersionMetadata{
					{ID: dataStream[2].ID, UpdatedAt: dataStream[2].UpdatedAt},
					{ID: dataStream[3].ID, UpdatedAt: dataStream[3].UpdatedAt},
				},
				Applications: []model.EntityVersionMetadata{
					{ID: application[5].ID, UpdatedAt: application[5].UpdatedAt},
					{ID: application[7].ID, UpdatedAt: application[7].UpdatedAt},
				},
				DataSources: []model.EntityVersionMetadata{
					{ID: dataSource[2].ID, UpdatedAt: dataSource[2].UpdatedAt},
					{ID: dataSource[4].ID, UpdatedAt: dataSource[4].UpdatedAt},
				},
				ProjectServices: []model.EntityVersionMetadata{
					{ID: projectService[2].ID, UpdatedAt: projectService[2].UpdatedAt},
					{ID: projectService[4].ID, UpdatedAt: projectService[4].UpdatedAt},
				},
				LogCollectors: []model.EntityVersionMetadata{
					{ID: logCollector[0].ID, UpdatedAt: logCollector[0].UpdatedAt},
					{ID: logCollector[2].ID, UpdatedAt: logCollector[2].UpdatedAt},
					{ID: logCollector[3].ID, UpdatedAt: logCollector[3].UpdatedAt},
				},
				SvcInstances: []model.EntityVersionMetadata{
					{ID: svcInstance[3].ID, UpdatedAt: svcInstance[3].UpdatedAt},
				},
				SvcBindings: []model.EntityVersionMetadata{
					{ID: svcBinding[3].ID, UpdatedAt: svcBinding[3].UpdatedAt},
				},
				DataDriverInstances: []model.EntityVersionMetadata{
					{ID: dataDriverInstance[2].ID, UpdatedAt: dataDriverInstance[2].UpdatedAt},
					{ID: dataDriverInstance[4].ID, UpdatedAt: dataDriverInstance[4].UpdatedAt},
				},
			},
			&model.EdgeInventoryDeltaPayload{
				Categories: []model.EntityVersionMetadata{
					{ID: category[4].ID, UpdatedAt: category[4].UpdatedAt},
				},
				Projects:            []model.EntityVersionMetadata{},
				Applications:        []model.EntityVersionMetadata{},
				CloudProfiles:       []model.EntityVersionMetadata{},
				ContainerRegistries: []model.EntityVersionMetadata{},
				RuntimeEnvironments: []model.EntityVersionMetadata{},
				Functions:           []model.EntityVersionMetadata{},
				DataPipelines:       []model.EntityVersionMetadata{},
				MLModels:            []model.EntityVersionMetadata{},
				DataSources:         []model.EntityVersionMetadata{},
				ProjectServices:     []model.EntityVersionMetadata{},
				LogCollectors:       []model.EntityVersionMetadata{},
				SvcInstances:        []model.EntityVersionMetadata{},
				SvcBindings:         []model.EntityVersionMetadata{},
				DataDriverInstances: []model.EntityVersionMetadata{},
			},
			&model.EdgeInventoryDeleted{
				Categories:          []string{bogusCatID},
				Applications:        []string{bogusAppID, application[11].ID},
				Projects:            []string{bogusProjID},
				CloudProfiles:       []string{bogusCloudProfileID},
				ContainerRegistries: []string{bogusContainerRegistryID},
				Functions:           []string{bogusScriptID},
				RuntimeEnvironments: []string{bogusScriptRuntimeID},
				MLModels:            []string{bogusMLModelID},
				DataPipelines:       []string{bogusDataStreamID},
				DataSources:         []string{bogusDataSourceID},
				ProjectServices:     []string{bogusProjectServiceID},
				LogCollectors:       []string{bogusLogCollectorID},
				SvcInstances:        []string{bogusSvcInstanceID},
				SvcBindings:         []string{bogusSvcBindingID},
				DataDriverInstances: []string{bogusDataDriverInstanceID},
			},
		},
	}

	for _, testData := range testDataList {
		t.Run(testData.name, func(t *testing.T) {
			t.Logf("running Test edge inventory delta for edge id %s", testData.edgeID)

			bapl, err := json.Marshal(*testData.payload)
			require.NoError(t, err)
			t.Logf("Get edge inventory delta payload: %s", string(bapl))

			edgeCtx := makeEdgeContext(tenantID, testData.edgeID, nil)
			resp, err := dbAPI.GetEdgeInventoryDelta(edgeCtx, testData.payload)
			require.NoError(t, err)
			ba, err := json.Marshal(resp)
			require.NoError(t, err)
			t.Logf("Got inventory delta for edge: %s", string(ba))

			verifyCreated(t, testData.created, resp)
			verifyUpdated(t, testData.updated, resp)
			verifyDeleted(t, testData.deleted, resp)
		})
	}

	// now test GetEdgeInventoryDeltaW
	r, err := objToReader(*testDataList[0].payload)
	require.NoError(t, err)
	url := "http://example.com/foo?edgeId=" + edge[0].ID
	req := httptest.NewRequest("POST", url, r)
	var w bytes.Buffer
	out := model.EdgeInventoryDeltaResponse{}
	err = dbAPI.GetEdgeInventoryDeltaW(ctx, &w, req)
	require.NoError(t, err)
	err = json.NewDecoder(&w).Decode(&out)
	require.NoError(t, err)
	outba, err := json.Marshal(out)
	require.NoError(t, err)
	t.Logf("GetEdgeInventoryDeltaW returns: %s", string(outba))
}
