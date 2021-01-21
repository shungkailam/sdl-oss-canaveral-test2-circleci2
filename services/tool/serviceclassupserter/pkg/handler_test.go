package pkg_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"cloudservices/tool/serviceclassupserter/pkg"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	ingressNginxSvcClassData = []byte(`{
		"type": "ingress-nginx",
		"svcVersion": "v100.0",
		"scope": "SERVICEDOMAIN",
		"minSvcDomainVersion": "1.15.0",
		"name": "ingress-nginx",
		"description": "Kubernetes Ingress",
		"state": "FINAL",
		"bindable": true,
		"schemas": {
			"svcInstance": {
				"create": {
					"parameters": {
						"properties": {},
						"title": "IngressCreateOptions",
						"type": "object"
					}
				},
				"update": {
					"parameters": {
						"properties": {
							"type": {}
						},
						"title": "IngressUpdateOptions",
						"type": "object"
					}
				}
			},
			"svcBinding": {
				"create": {
					"parameters": {}
				}
			}
		},
		"tags": [
			{
				"name": "essential",
				"value": "yes"
			},
			{
				"name": "category",
				"value": "ingress"
			}
		]
	}`)

	kafkaSvcClassData = []byte(`{
		"type": "kafka",
		"svcVersion": "v100.0",
		"scope": "PROJECT",
		"minSvcDomainVersion": "1.15.0",
		"name": "Kafka",
		"description": "Message Streaming Service",
		"state": "FINAL",
		"bindable": true,
		"schemas": {
			"svcInstance": {
				"create": {
					"parameters": {
						"properties": {
							"profile": {
								"enum": [
									"Durability",
									"Performance"
								],
								"type": "string"
							}
						},
						"title": "KafkaCreateOptions",
						"type": "object"
					}
				},
				"update": {
					"parameters": {
						"properties": {},
						"title": "KafkaUpdateOptions",
						"type": "object"
					}
				}
			},
			"svcBinding": {
				"create": {
					"parameters": {}
				}
			}
		},
		"tags": [
			{
				"name": "essential",
				"value": "yes"
			},
			{
				"name": "category",
				"value": "pub-sub"
			}
		]
	}`)

	istioSvcClassData = []byte(`{
		"type": "istio",
		"svcVersion": "v100.0",
		"scope": "SERVICEDOMAIN",
		"minSvcDomainVersion": "1.18",
		"name": "istio class",
		"description": "service mesh",
		"state": "FINAL",
		"bindable": true,
		"schemas": {
			"svcInstance": {
				"create": {
					"parameters": {}
				},
				"update": {
					"parameters": {}
				}
			},
			"svcBinding": {
				"create": {
					"parameters": {}
				}
			}
		},
		"tags": [
			{
				"name": "essential",
				"value": "yes"
			},
			{
				"name": "category",
				"value": "monitoring"
			}
		]
	}`)

	ingressSvcClassData = []byte(`{
		"type": "ingress",
		"svcVersion": "v1.0",
		"scope": "SERVICEDOMAIN",
		"minSvcDomainVersion": "1.15.0",
		"name": "Ingress",
		"description": "Kubernetes Ingress",
		"state": "FINAL",
		"bindable": false,
		"schemas": {
			"svcInstance": {
				"create": {
					"parameters": {
						"properties": {
							"type": {
								"enum": [
									"Traefik",
									"NGINX"
								],
								"type": "string"
							}
						},
						"title": "IngressCreateOptions",
						"type": "object"
					}
				},
				"update": {
					"parameters": {
						"properties": {
							"type": {
								"enum": [
									"Traefik",
									"NGINX"
								],
								"type": "string"
							}
						},
						"title": "IngressUpdateOptions",
						"type": "object"
					}
				}
			},
			"svcBinding": {
				"create": {
					"parameters": {}
				}
			},
			"tags": [
				{
					"name": "essential",
					"value": "yes"
				},
				{
					"name": "category",
					"value": "ingress"
				}
			]
		}
	}`)
)

func unmarshalServiceClass(t *testing.T, data []byte) *model.ServiceClass {
	svcClass := &model.ServiceClass{}
	err := json.Unmarshal(data, svcClass)
	require.NoError(t, err)
	return svcClass
}

func generateMetadataFile(t *testing.T, svcClasses map[string]*model.ServiceClass) {
	tmpDir := pkg.Cfg.DataDir
	metadata := pkg.ServiceClassMetadata{}
	for relPath, svcClass := range svcClasses {
		svcClass.ID = ""
		svcClassFilepath := filepath.Join(tmpDir, relPath)
		svcClassDirPath := filepath.Dir(svcClassFilepath)
		os.MkdirAll(svcClassDirPath, 0777)
		data, err := json.Marshal(svcClass)
		require.NoError(t, err)
		t.Logf("Writing Service Class %s to %s\n", string(data), svcClassFilepath)
		err = ioutil.WriteFile(svcClassFilepath, data, 0777)
		require.NoError(t, err)
		metadata.Includes = append(metadata.Includes, relPath)
	}
	metaBytes, err := json.Marshal(metadata)
	require.NoError(t, err)
	t.Logf("Writing metadata %s\n", string(metaBytes))
	metadataFile := filepath.Join(tmpDir, "metadata.json")
	err = ioutil.WriteFile(metadataFile, metaBytes, 0777)
	require.NoError(t, err)
}

func TestServiceClassUpserter(t *testing.T) {
	ctx := base.GetOperatorContext(context.Background())
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	defer dbAPI.Close()
	tmpDir, err := ioutil.TempDir("/tmp", "svc-class-test")
	require.NoError(t, err)
	defer os.Remove(tmpDir)
	pkg.Cfg.DataDir = tmpDir
	pkg.Cfg.DeleteOnMissing = true

	svcClasses := map[string]*model.ServiceClass{}
	svcClassesToAdd := map[string]*model.ServiceClass{}
	svcClassesToUpdate := map[string]*model.ServiceClass{}
	svcClassIDsToDelete := map[string]bool{}
	// Prepare data
	// Make sure only other classes do not interfere
	tagValue := base.GetUUID()

	// ingressNginxSvcClassData is marked for update
	svcClass := unmarshalServiceClass(t, ingressNginxSvcClassData)
	svcClass.ID = base.GetUUID()
	svcClass.Name = "name-" + svcClass.ID
	svcClass.Type = svcClass.Type + svcClass.ID
	svcClass.Tags = append(svcClass.Tags, model.ServiceClassTag{
		Name:  "test",
		Value: tagValue,
	})
	t.Logf("%+v\n", svcClass)
	_, err = dbAPI.CreateServiceClass(ctx, svcClass, nil)
	require.NoError(t, err)
	defer dbAPI.DeleteServiceClass(ctx, svcClass.ID, nil)
	relPath := fmt.Sprintf("%s/%s-1.json", svcClass.Type, svcClass.Type)
	svcClass.Description = "Updated description"
	svcClasses[relPath] = svcClass
	svcClassesToUpdate[svcClass.Type] = svcClass

	// kafkaSvcClassData is marked for deletion
	svcClass = unmarshalServiceClass(t, kafkaSvcClassData)
	svcClass.ID = base.GetUUID()
	svcClass.Name = "name-" + svcClass.ID
	svcClass.Type = svcClass.Type + svcClass.ID
	svcClass.Tags = append(svcClass.Tags, model.ServiceClassTag{
		Name:  "test",
		Value: tagValue,
	})
	t.Logf("%+v\n", svcClass)
	_, err = dbAPI.CreateServiceClass(ctx, svcClass, nil)
	require.NoError(t, err)
	defer dbAPI.DeleteServiceClass(ctx, svcClass.ID, nil)
	svcClassIDsToDelete[svcClass.Type] = true

	// istioSvcClassData is marked for no change
	svcClass = unmarshalServiceClass(t, istioSvcClassData)
	svcClass.ID = base.GetUUID()
	svcClass.Name = "name-" + svcClass.ID
	svcClass.Type = svcClass.Type + svcClass.ID
	svcClass.Tags = append(svcClass.Tags, model.ServiceClassTag{
		Name:  "test",
		Value: tagValue,
	})
	t.Logf("%+v\n", svcClass)
	_, err = dbAPI.CreateServiceClass(ctx, svcClass, nil)
	require.NoError(t, err)
	defer dbAPI.DeleteServiceClass(ctx, svcClass.ID, nil)
	relPath = fmt.Sprintf("%s/%s-1.json", svcClass.Type, svcClass.Type)
	svcClasses[relPath] = svcClass

	// ingressSvcClassData is marked for addition
	svcClass = unmarshalServiceClass(t, ingressSvcClassData)
	svcClass.ID = base.GetUUID()
	svcClass.Name = "name-" + svcClass.ID
	svcClass.Type = svcClass.Type + svcClass.ID
	svcClass.Tags = append(svcClass.Tags, model.ServiceClassTag{
		Name:  "test",
		Value: tagValue,
	})
	t.Logf("%+v\n", svcClass)
	relPath = fmt.Sprintf("%s/%s-1.json", svcClass.Type, svcClass.Type)
	svcClasses[relPath] = svcClass
	svcClassesToAdd[svcClass.Type] = svcClass

	generateMetadataFile(t, svcClasses)

	createHandler := func(ctx context.Context, dbAPI api.ObjectModelAPI, svcClasses map[string]*model.ServiceClass) error {
		t.Logf("To add %+v\n", svcClasses)
		if !reflect.DeepEqual(svcClassesToAdd, svcClasses) {
			t.Fatalf("expected %+v\n, found %+v\n", svcClassesToAdd, svcClasses)
		}
		return nil
	}
	updateHandler := func(ctx context.Context, dbAPI api.ObjectModelAPI, svcClasses map[string]*model.ServiceClass) error {
		t.Logf("To update %+v\n", svcClasses)
		if len(svcClasses) != len(svcClassesToUpdate) {
			t.Fatalf("expected %+v\n, found %+v\n", svcClassesToAdd, svcClasses)
		}
		for _, svcClass := range svcClassesToUpdate {
			if _, ok := svcClasses[svcClass.Type]; !ok {
				t.Fatalf("missing expected Service Class %+v", svcClass)
			}
		}
		return nil
	}
	deleteHandler := func(ctx context.Context, dbAPI api.ObjectModelAPI, svcClassIDs map[string]bool) error {
		t.Logf("To delete %+v\n", svcClassIDs)
		if !reflect.DeepEqual(svcClassIDsToDelete, svcClassIDs) {
			t.Fatalf("expected %+v\n, found %+v\n", svcClassIDsToDelete, svcClassIDs)
		}
		return nil
	}
	tags := []string{"test=" + tagValue}
	err = pkg.UpsertServiceClasses(ctx, &model.ServiceClassQueryParam{Tags: tags}, createHandler, updateHandler, deleteHandler)
	require.NoError(t, err)
}
