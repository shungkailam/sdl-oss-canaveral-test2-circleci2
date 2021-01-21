package apptemplate

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	yaml2 "github.com/ghodss/yaml"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	yaml "k8s.io/apimachinery/pkg/util/yaml"
)

const (
	secretTemplate = `apiVersion: v1
data:
%s
kind: Secret
metadata:
  name: %s
  labels:
    app: %s
type: Opaque`
)

// Adapted from: data-stream code {

const (
	cronJobKind               = "cronjob"
	replicasetKind            = "replicaset"
	deploymentKind            = "deployment"
	statefulsetKind           = "statefulset"
	daemonsetKind             = "daemonset"
	jobKind                   = "job"
	podKind                   = "pod"
	replicationControllerKind = "replicationcontroller"
)

func isContainerContainer(kind string) bool {
	return kind == replicasetKind ||
		kind == deploymentKind ||
		kind == statefulsetKind ||
		kind == daemonsetKind ||
		kind == cronJobKind ||
		kind == jobKind ||
		kind == podKind ||
		kind == replicationControllerKind
}

func getContainersPath(kind string) []string {
	if isContainerContainer(kind) == false {
		panic("Expected kind to be a container container")
	}
	if kind == cronJobKind {
		return []string{"spec", "jobTemplate", "spec", "template", "spec", "containers"}
	} else if kind == podKind {
		return []string{"spec", "containers"}
	} else {
		return []string{"spec", "template", "spec", "containers"}
	}
}

type containerIteratorOp func(container map[string]interface{}) error

// iterateContainers - iterate through the containers in unstructYaml
// and apply op on each container
func iterateContainers(
	unstructYaml *unstructured.Unstructured,
	op containerIteratorOp,
) error {
	yamlObjName := unstructYaml.GetName()
	kind := strings.ToLower(unstructYaml.GetKind())

	if !isContainerContainer(kind) {
		return nil
	}

	path := getContainersPath(kind)
	containerSlice, ok, err := unstructured.NestedSlice(unstructYaml.Object,
		path...)
	if !ok {
		glog.V(5).Infof("Containers not found in %s", yamlObjName)
		return nil
	} else if err != nil {
		glog.Errorf("Container slice is not []interface{}")
		return err
	}

	for i, elem := range containerSlice {
		glog.V(6).Infof("Process container %+v\n", elem)
		container, ok := elem.(map[string]interface{})
		if !ok {
			glog.Errorf("container index=%d is not in map[str]interface{} format. APIObject name: %s", i, yamlObjName)
			continue
		}
		if err := op(container); err != nil {
			return err
		}
		containerSlice[i] = container
	}
	return unstructured.SetNestedSlice(unstructYaml.Object,
		containerSlice, path...)
}

// Input: Yaml string with multiple k8s objects seperated by "---"
func getObjsFromYaml(yamlStr string) (rtAr []*unstructured.Unstructured, err error) {
	glog.V(6).Info("#Enter getYamlObjs")

	ior := strings.NewReader(yamlStr)
	decoder := yaml.NewYAMLToJSONDecoder(ior)
	for {
		unstructuredYaml := &unstructured.Unstructured{}
		err = decoder.Decode(unstructuredYaml)
		if err == io.EOF {
			break
		}
		if err != nil {
			glog.Errorf("Error while decoding YAML object: %s",
				err.Error())
			return nil, err
		}
		glog.V(7).Infof("Unstruct data: %s", unstructuredYaml)
		if len(unstructuredYaml.Object) == 0 {
			continue
		}
		rtAr = append(rtAr, unstructuredYaml)
	}
	glog.V(6).Info("#Exit getYamlObjs")
	return rtAr, nil
}

// Taken from: data-stream code }

// getYamlFromObjs - marshal objs back to yaml, joined with "---"
func getYamlFromObjs(objs []*unstructured.Unstructured) (yamlStr string, err error) {
	yamlParts := []string{}
	for _, obj := range objs {
		if yamlBytes, err2 := yaml2.Marshal(obj); err2 != nil {
			err = err2
			return
		} else {
			yamlParts = append(yamlParts, string(yamlBytes))
		}
	}
	yamlStr = strings.Join(yamlParts, "---\n")
	return
}
func secretName(appID string) string {
	return "secret-" + appID
}
func appLabel(appID string) string {
	return "app-" + appID
}

// updateEnv - update env to add names from nvMap
// whose value is from secret ref to app secret with the same key
func updateEnv(appID string, env *[]interface{}, nvMap map[string]string) (err error) {
	newEnv := []interface{}{}
	for _, e := range *env {
		// only keep entry not overwritten
		overwritten := false
		if m, ok := e.(map[string]interface{}); ok {
			if nm, ok := m["name"]; ok {
				if n, ok := nm.(string); ok {
					if _, ok := nvMap[n]; ok {
						overwritten = true
					}
				}
			}
		}
		if !overwritten {
			newEnv = append(newEnv, e)
		}
	}
	secretName := secretName(appID)
	for name, _ := range nvMap {
		envEntry := make(map[string]interface{})
		err = unstructured.SetNestedField(envEntry, name, "name")
		if err != nil {
			return
		}
		secretKeyRef := make(map[string]interface{})
		err = unstructured.SetNestedField(secretKeyRef, secretName, "name")
		if err != nil {
			return
		}
		err = unstructured.SetNestedField(secretKeyRef, name, "key")
		if err != nil {
			return
		}
		valueFrom := make(map[string]interface{})
		err = unstructured.SetNestedField(valueFrom, secretKeyRef, "secretKeyRef")
		if err != nil {
			return
		}
		err = unstructured.SetNestedField(envEntry, valueFrom, "valueFrom")
		if err != nil {
			return
		}
		newEnv = append(newEnv, envEntry)
	}
	*env = newEnv
	return
}

func trimMap(m map[string]string) map[string]string {
	m2 := map[string]string{}
	for k, v := range m {
		kt := strings.TrimSpace(k)
		vt := strings.TrimSpace(v)
		if len(kt) != 0 && len(vt) != 0 {
			m2[kt] = vt
		}
	}
	return m2
}

// parseEnvs - parse the envsStr json string to return a name, value map
func parseEnvs(envsStr *string) (map[string]string, error) {
	if envsStr == nil {
		return nil, nil
	}
	nvMap := make(map[string]string)
	err := json.Unmarshal([]byte(*envsStr), &nvMap)
	return trimMap(nvMap), err
}

func RedactEnvs(envsStr *string) *string {
	if envsStr == nil {
		return nil
	}
	nvMap := make(map[string]string)
	err := json.Unmarshal([]byte(*envsStr), &nvMap)
	if err != nil {
		return nil
	}
	if len(nvMap) == 0 {
		return nil
	}
	for k := range nvMap {
		nvMap[k] = ""
	}
	ba, err := json.Marshal(nvMap)
	if err != nil {
		return nil
	}
	s := string(ba)
	return &s
}

// addEnvsToYaml - save the nvMap into a k8s secret,
// then add the keys are env var keys with value reference to the secret
func addEnvsToYaml(appID, yamlStr string, nvMap map[string]string) (string, error) {
	secret := mapToSecrets(appID, nvMap)
	objs, err := getObjsFromYaml(yamlStr)
	if err != nil {
		return "", err
	}
	op := func(container map[string]interface{}) error {
		env, _, err := unstructured.NestedSlice(container, "env")
		if err != nil {
			return err
		}
		if err := updateEnv(appID, &env, nvMap); err != nil {
			return err
		}
		return unstructured.SetNestedSlice(container,
			env, "env")
	}
	for i, obj := range objs {
		kind := strings.ToLower(obj.GetKind())
		if isContainerContainer(kind) {
			err := iterateContainers(obj, op)
			if err != nil {
				return "", err
			}
			objs[i] = obj
		}
	}
	newYaml, err := getYamlFromObjs(objs)
	if err != nil {
		return "", err
	}
	newYaml = fmt.Sprintf("%s\n---\n%s", newYaml, secret)
	return newYaml, nil
}

// AddEnvsToYamlMaybe - if env parse to non empty map,
// then add the env vars into the yaml string
func AddEnvsToYamlMaybe(appID, yamlStr string, env *string) (string, error) {
	nvMap, err := parseEnvs(env)
	if err != nil {
		return "", err
	}
	if len(nvMap) == 0 {
		return yamlStr, nil
	}
	return addEnvsToYaml(appID, yamlStr, nvMap)
}

func mapToSecrets(appID string, nvMap map[string]string) string {
	name := secretName(appID)
	appLabel := appLabel(appID)
	sa := []string{}
	for n, v := range nvMap {
		sa = append(sa, fmt.Sprintf("  %s: %s", n, base64.StdEncoding.EncodeToString([]byte(v))))
	}
	data := strings.Join(sa, "\n")
	return fmt.Sprintf(secretTemplate, data, name, appLabel)
}
