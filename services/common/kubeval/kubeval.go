package kubeval

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/golang/glog"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v2"
)

var (
	dns1123Pattern       = `[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*`
	dns1123Error         = "Name must be valid DNS-1123 subdomain"
	dns1123Regex         = regexp.MustCompilePOSIX(dns1123Pattern)
	labelNamePattern     = `([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]`
	labelPrefixPartError = "Prefix part must be valid DNS subdomain"
	labelNamePartError   = "Name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character"
	labelNameRegex       = regexp.MustCompilePOSIX(labelNamePattern)
	labelValuePattern    = `(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?`
	labelValueError      = "A valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character"
	labelValueRegex      = regexp.MustCompilePOSIX(labelValuePattern)
)

// ValidFormat is a type for quickly forcing
// new formats on the gojsonschema loader
type ValidFormat struct{}

// InvalidValueError is used to give error whem the entered hostport is reserved
type InvalidValueError struct {
	gojsonschema.ResultErrorFields
}

func newInvalidValueError(context *gojsonschema.JsonContext, value interface{}, details gojsonschema.ErrorDetails) *InvalidValueError {
	err := InvalidValueError{}
	err.SetContext(context)
	err.SetType("invalid_value_error")
	// it is important to use SetDescriptionFormat() as this is used to call SetDescription() after it has been parsed
	// using the description of err will be overridden by this.
	err.SetDescriptionFormat("{{.err}}")
	err.SetValue(value)
	err.SetDetails(details)

	return &err
}

func init() {
	// Without forcing these types the schema fails to load
	// Need to Work out proper handling for these types
	gojsonschema.FormatCheckers.Add("int64", ValidFormat{})
	gojsonschema.FormatCheckers.Add("byte", ValidFormat{})
	gojsonschema.FormatCheckers.Add("int32", ValidFormat{})
	gojsonschema.FormatCheckers.Add("int-or-string", ValidFormat{})
}

// IsFormat always returns true and meets the
// gojsonschema.FormatChecker interface
func (f ValidFormat) IsFormat(input interface{}) bool {
	return true
}

// ValidationResult contains the details from
// validating a given Kubernetes resource
type ValidationResult struct {
	FileName   string
	Kind       string
	APIGroup   string
	APIVersion string
	Errors     []gojsonschema.ResultError
}

// detectLineBreak returns the relevant platform specific line ending
func detectLineBreak(haystack []byte) string {
	if bytes.Contains(haystack, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}

func determineSchema(kind, apiGroup, apiVersion, schemaVersion string) string {
	// The schema is located at ../../common/kubeval/kubernetes-json-schema/ under the k8s version
	fileloc, _ := filepath.Abs("../../common/kubeval/kubernetes-json-schema/")
	baseURL := "file://" + fileloc
	if apiGroup == "" {
		return fmt.Sprintf("%s/%s/%s-%s.json",
			baseURL,
			schemaVersion,
			strings.ToLower(kind),
			strings.ToLower(apiVersion),
		)
	}
	apiGroup = strings.TrimSuffix(apiGroup, ".authorization.k8s.io")
	apiGroup = strings.TrimSuffix(apiGroup, ".k8s.io")
	return fmt.Sprintf("%s/%s/%s-%s-%s.json",
		baseURL,
		schemaVersion,
		strings.ToLower(kind),
		strings.ToLower(apiGroup),
		strings.ToLower(apiVersion),
	)
}

func determineSchemaForIstioCRD(kind, apiGroup, apiVersion string) string {
	// The schema for istio is located at ../../common/kubeval/kubernetes-json-schema/istio under the k8s version
	fileloc, _ := filepath.Abs("../../common/kubeval/kubernetes-json-schema/istio")
	baseURL := "file://" + fileloc
	apiGroup = strings.TrimSuffix(apiGroup, ".istio.io")
	return fmt.Sprintf("%s/%s/%s/%s.json",
		baseURL,
		strings.ToLower(apiGroup),
		strings.ToLower(apiVersion),
		strings.ToLower(kind),
	)
}

func determineResourceType(body interface{}) (kind string, apiGroup string, apiVersion string, err error) {
	ok := false
	cast, _ := body.(map[string]interface{})
	if _, ok = cast["kind"]; !ok {
		err = errors.New("Missing kind")
		return
	}
	if _, ok = cast["apiVersion"]; !ok {
		err = errors.New("Missing apiVersion")
		return
	}
	if cast["kind"] == nil {
		err = errors.New("Missing kind value")
		return
	}
	if cast["apiVersion"] == nil {
		err = errors.New("Missing apiVersion value")
		return
	}
	kind = cast["kind"].(string)
	apiVersionGroup := cast["apiVersion"].(string)
	tokens := strings.Split(apiVersionGroup, "/")
	if len(tokens) == 1 {
		apiVersion = tokens[0]
	} else if len(tokens) == 2 {
		apiGroup = tokens[0]
		apiVersion = tokens[1]
	} else {
		err = fmt.Errorf("Invalid apiGroup %q", apiVersionGroup)
	}
	return
}

func validateHostport(kind string, body interface{}) *InvalidValueError {
	// This lists the keys that need to be traversed.
	hostPortPath := []string{"spec", "template", "spec", "containers", "ports", "hostPort"}
	if kind == "cronjob" {
		hostPortPath = []string{"spec", "jobTemplate", "spec", "containers", "ports", "hostPort"}
	}
	disallowedPorts := []int{22, 25, 111, 2379, 2380, 6443, 6781, 6782, 6783, 6784, 8080,
		10248, 10249, 10250, 10251, 10252, 10255, 10256, 45665}
	validHostportFn := func(value interface{}) *InvalidValueError {
		usedPort, ok := value.(int)
		if ok != true {
			// It does not contain a hostport
			return nil
		}

		for _, port := range disallowedPorts {
			if usedPort == port {
				jsonContext := gojsonschema.NewJsonContext("hostport", nil)
				errDetail := gojsonschema.ErrorDetails{
					"err": fmt.Sprintf("Port %d is reserved. ", usedPort),
				}
				err := newInvalidValueError(jsonContext,
					usedPort,
					errDetail)
				return err
			}
		}
		return nil
	}
	res := validateValue(body, hostPortPath, validHostportFn)
	return res
}

// Names of deployments, statefulsets, daemonsets, config maps and
// secrets must be conform to DNS-1123
func validateName(body interface{}) *InvalidValueError {
	// This lists the keys that need to be traversed.
	namePath := []string{"metadata", "name"}
	validNameFn := func(value interface{}) *InvalidValueError {
		name, ok := value.(string)
		if ok && name != "" && dns1123Regex.FindString(name) == name {
			return nil
		}
		jsonContext := gojsonschema.NewJsonContext("metadata.name",
			nil)
		return newInvalidValueError(jsonContext,
			name,
			gojsonschema.ErrorDetails{
				"err": fmt.Sprintf("Invalid name '%s'. %s",
					name, dns1123Error),
			})
	}
	return validateValue(body, namePath, validNameFn)
}

func validateLabels(kind string, body interface{}) (errors []*InvalidValueError) {
	// This lists the keys that need to be traversed.
	path := []string{"metadata", "labels"}
	fn := func(value interface{}) *InvalidValueError {
		labels, ok := value.(map[string]interface{})
		if !ok {
			glog.V(5).Infof("%s has no labels", kind)
			return nil
		}
		for key, valIf := range labels {
			var prefix, name string

			val, ok := valIf.(string)
			if !ok {
				glog.V(5).Infof("%s not string", name)
				continue
			}
			jsonContext := gojsonschema.NewJsonContext(
				fmt.Sprintf("metadata.labels.%s", name),
				nil)

			tokens := strings.SplitN(key, "/", 2)
			if len(tokens) == 1 {
				prefix = ""
				name = tokens[0]
			} else if len(tokens) == 2 {
				prefix = tokens[0]
				name = tokens[1]
			}

			if prefix != "" && dns1123Regex.FindString(prefix) != prefix {
				errors = append(errors, newInvalidValueError(jsonContext,
					name,
					gojsonschema.ErrorDetails{
						"err": fmt.Sprintf("Invalid label name prefix '%s'. %s",
							prefix, labelPrefixPartError),
					}))
			}
			if labelNameRegex.FindString(name) != name {
				errors = append(errors, newInvalidValueError(jsonContext,
					name,
					gojsonschema.ErrorDetails{
						"err": fmt.Sprintf("Invalid label name '%s'. %s",
							name, labelNamePartError),
					}))
			}
			if labelValueRegex.FindString(val) != val {
				errors = append(errors, newInvalidValueError(jsonContext,
					name,
					gojsonschema.ErrorDetails{
						"err": fmt.Sprintf("Invalid label value '%s'. %s",
							val, labelValueError),
					}))
			}
		}
		return nil
	}
	// controller resource specify pod templates with labels
	if strings.ToLower(kind) == "daemonset" ||
		strings.ToLower(kind) == "deployment" || strings.ToLower(kind) == "statefulset" {
		path := []string{"spec", "template", "metadata", "labels"}
		validateValue(body, path, fn)
	}
	if strings.ToLower(kind) == "cronjob" {
		path := []string{"spec", "jobTemplate", "metadata", "labels"}
		validateValue(body, path, fn)
	}
	validateValue(body, path, fn)
	return
}

func validateStorageClass(body interface{}) *InvalidValueError {
	// This lists the keys that need to be traversed.
	storageClassPath := []string{"spec", "volumeClaimTemplates", "spec", "storageClassName"}
	allowedStorageClasses := map[string]bool{"silver": true, "local": true, "shared": true}
	validScFn := func(value interface{}) *InvalidValueError {
		storageClassUsed, ok := value.(string)
		if ok != true {
			// It does not contain a storageclass
			return nil
		}
		if ok := allowedStorageClasses[storageClassUsed]; !ok {
			jsonContext := gojsonschema.NewJsonContext("storageClassName", nil)
			errDetail := gojsonschema.ErrorDetails{
				"err": fmt.Sprintf("storageClassName %s is not allowed. Only local storage class is allowed", storageClassUsed),
			}
			err := newInvalidValueError(jsonContext,
				storageClassUsed,
				errDetail)
			return err
		}
		return nil
	}
	res := validateValue(body, storageClassPath, validScFn)
	return res
}

func validatePVCStorageClass(body interface{}) *InvalidValueError {
	// This lists the keys that need to be traversed.
	storageClassPath := []string{"spec", "storageClassName"}
	allowedStorageClasses := map[string]bool{"local": true}
	validScFn := func(value interface{}) *InvalidValueError {
		storageClassUsed, ok := value.(string)
		if ok != true {
			// It does not contain a storageclass
			return nil
		}
		if ok := allowedStorageClasses[storageClassUsed]; !ok {
			jsonContext := gojsonschema.NewJsonContext("storageClassName", nil)
			errDetail := gojsonschema.ErrorDetails{
				"err": fmt.Sprintf("storageClassName %s is not allowed. Only local storage class is allowed", storageClassUsed),
			}
			err := newInvalidValueError(jsonContext,
				storageClassUsed,
				errDetail)
			return err
		}
		return nil
	}
	res := validateValue(body, storageClassPath, validScFn)
	return res
}

// validate the value is correct, specPath is the path to the value to be evaluated
func validateValue(body interface{}, specPath []string, validationFn func(interface{}) *InvalidValueError) *InvalidValueError {

	// This is only for validationg the value, and assumes the keys will be validated by jsonswagger
	// If the key is not there then I do not error out
	for specIndex, key := range specPath {
		switch x := body.(type) {
		case map[string]interface{}:
			var ok bool
			if body, ok = x[key]; !ok {
				return nil
			}
		case []interface{}:
			for _, v := range x {
				err := validateValue(v, specPath[specIndex:], validationFn)
				if err != nil {
					return err
				}
			}
		}
	}

	return validationFn(body)
}

func addCRD(crdMap map[string]*gojsonschema.JSONLoader, body map[string]interface{}, loader *gojsonschema.JSONLoader) {
	if spec, ok := body["spec"]; ok {
		if spec, ok := spec.(map[string]interface{}); ok {
			if group, ok := spec["group"]; ok {
				if names, ok := spec["names"]; ok {
					if names, ok := names.(map[string]interface{}); ok {
						if kind, ok := names["kind"]; ok {
							kg := fmt.Sprintf("%s.%s", kind, group)
							crdMap[kg] = loader
						}
					}
				}
			}
		}
	}
}

func addCRDs(crdMap map[string]*gojsonschema.JSONLoader, crds string) {
	if crds == "" {
		return
	}
	baCRDs := []byte(crds)
	nl := detectLineBreak(baCRDs)
	bits := bytes.Split(baCRDs, []byte(nl+"---"+nl))
	for _, element := range bits {
		if len(element) > 0 {
			var spec interface{}
			err := yaml.Unmarshal(element, &spec)
			if err != nil {
				continue
			}
			body := convertToStringKeys(spec)
			cast, _ := body.(map[string]interface{})
			if len(cast) == 0 {
				continue
			}
			documentLoader := gojsonschema.NewGoLoader(body)
			addCRD(crdMap, cast, &documentLoader)
		}
	}
}

func addBuiltinCRDs(crdMap map[string]*gojsonschema.JSONLoader) error {
	crdDir, err := filepath.Abs("../../common/kubeval/crds")
	if err != nil {
		return err
	}
	files, err := ioutil.ReadDir(crdDir)
	if err != nil {
		return err
	}
	for _, file := range files {
		filePath := filepath.Join(crdDir, file.Name())
		buf, err := ioutil.ReadFile(filePath)
		if err != nil {
			return err
		}
		addCRDs(crdMap, string(buf))
	}
	return nil
}

// validateResource validates a single Kubernetes resource against
// the relevant schema, detecting the type of resource automatically
func validateResource(data []byte, fileName string, version string, crdMap map[string]*gojsonschema.JSONLoader) (*ValidationResult, error) {
	var spec interface{}
	result := ValidationResult{}
	result.FileName = fileName
	err := yaml.Unmarshal(data, &spec)
	if err != nil {
		return nil, errors.New("Failed to decode YAML please check format.")
	}

	body := convertToStringKeys(spec)

	if body == nil {
		return nil, nil
	}

	cast, _ := body.(map[string]interface{})
	if len(cast) == 0 {
		return nil, nil
	}

	documentLoader := gojsonschema.NewGoLoader(body)

	kind, apiGroup, apiVersion, err := determineResourceType(body)
	if err != nil {
		return nil, err
	}

	result.Kind = kind
	result.APIGroup = apiGroup
	result.APIVersion = apiVersion
	mainSchemaName := determineSchema(kind, apiGroup, apiVersion, version)

	glog.V(8).Infoln("mainSchemaName", mainSchemaName)

	schemalocation := strings.TrimPrefix(mainSchemaName, "file://")
	haveCRD := false
	isIstioCR := false
	if _, err := os.Stat(schemalocation); err != nil {
		if os.IsNotExist(err) {
			if kind == "CustomResourceDefinition" {
				addCRD(crdMap, cast, &documentLoader)
				// skip validation of CRD
				return &result, nil
			}

			// Check if it is CR of known kind
			var apiGroupVersion string
			if strings.HasSuffix(apiGroup, ".istio.io") {
				isIstioCR = true
				mainSchemaName = determineSchemaForIstioCRD(kind, apiGroup, apiVersion)
				glog.V(5).Infof("istio SchemaName %s", mainSchemaName)
				schemalocation = strings.TrimPrefix(mainSchemaName, "file://")
				if _, err := os.Stat(schemalocation); err != nil {
					glog.V(5).Infof("Error in istio schemalocation stat: %v", err)
					if os.IsNotExist(err) {
						apiGroupVersion = fmt.Sprintf("%s/%s", apiGroup, apiVersion)
						return &result, fmt.Errorf("%s.%s is not supported",
							kind, apiGroupVersion)
					} else {
						return &result, fmt.Errorf("Problem loading schema: %s", err)
					}
				} else {
					isIstioCR = true
				}
			} else {
				kg := fmt.Sprintf("%s.%s", kind, apiGroup)
				if _, ok := crdMap[kg]; ok {
					haveCRD = true
				}

				apiGroupVersion = fmt.Sprintf("%s/%s", apiGroup, apiVersion)
				if apiGroup == "" {
					apiGroupVersion = apiGroup
				}
			}
			if !haveCRD && !isIstioCR {
				return &result, fmt.Errorf("%s.%s is not supported",
					kind, apiGroupVersion)
			}
		} else {
			return &result, fmt.Errorf("Problem loading schema: %s", err)
		}
	}

	var results *gojsonschema.Result
	if isIstioCR {
		results, err = validateIstioCR(body, version, mainSchemaName)
		if err != nil {
			glog.V(5).Infof("Error in istio validation: %v", err)
			return &result, err
		}
	} else {
		schemaLoader := gojsonschema.NewSchemaLoader()
		kg := fmt.Sprintf("%s.%s", kind, apiGroup)
		pMainSchema := crdMap[kg]
		if pMainSchema == nil {
			mainSchema := gojsonschema.NewReferenceLoader(mainSchemaName)
			pMainSchema = &mainSchema
		}
		schema, err := schemaLoader.Compile(*pMainSchema)
		if err != nil {
			return &result, fmt.Errorf("Problem compiling schema %s from: %s", mainSchemaName, err)
		}
		results, err = schema.Validate(documentLoader)
		if err != nil {
			return &result, fmt.Errorf("Problem validating: %s", err)
		}
	}
	// Validate name
	if err := validateName(body); err != nil {
		results.AddError(err, err.Details())
	}
	// Validate labels
	for _, err := range validateLabels(kind, body) {
		results.AddError(err, err.Details())
	}
	// TODO: if kind is service validate ports
	if strings.ToLower(kind) == "cronjob" || strings.ToLower(kind) == "daemonset" ||
		strings.ToLower(kind) == "deployment" || strings.ToLower(kind) == "statefulset" {
		err := validateHostport(kind, body)
		if err != nil {
			results.AddError(err, err.Details())
		}
	}
	if strings.ToLower(kind) == "statefulset" {
		err := validateStorageClass(body)
		if err != nil {
			results.AddError(err, err.Details())
		}
	}
	if strings.ToLower(kind) == "persistentvolumeclaim" {
		err := validatePVCStorageClass(body)
		if err != nil {
			results.AddError(err, err.Details())
		}
	}
	if results.Valid() {
		return &result, nil
	}

	result.Errors = results.Errors()
	return &result, nil
}

func validateIstioCR(body interface{}, version, schemalocation string) (*gojsonschema.Result, error) {
	glog.V(5).Info("Istio CR validation")
	cast, _ := body.(map[string]interface{})
	if len(cast) != 4 { // We allow apiVerison, kind, metadata, spec
		var extraFields []string
		kind := cast["kind"].(string)
		for key := range cast {
			if key != "apiVersion" && key != "kind" && key != "metadata" && key != "spec" {
				extraFields = append(extraFields, key)
			}
		}

		return nil, fmt.Errorf("Yaml has following errors: [%s](root):  Additional properties %v not allowed", kind, extraFields)
	}
	metadata, ok := cast["metadata"]
	if !ok {
		err := errors.New("Missing metadata")
		return nil, err
	}
	// only v1.15.4-standalone-strict-restricted-full has metadata without namespace field
	metadataSchema := determineSchema("ObjectMeta", "", "meta-v1", "v1.15.4-standalone-strict-restricted-full")
	glog.V(5).Infof("Metadata: %v", metadataSchema)
	metadataLocation := strings.TrimPrefix(metadataSchema, "file://")
	if _, err := os.Stat(metadataLocation); err != nil {
		return nil, err
	}
	results, err := validateSubresourceWithSchema(metadata, metadataSchema)
	if err != nil {
		return nil, err
	}

	if !results.Valid() {
		return results, nil
	}

	cast, _ = body.(map[string]interface{})
	spec, ok := cast["spec"]
	if !ok {
		err = errors.New("Missing spec")
		return nil, err
	}
	return validateSubresourceWithSchema(spec, schemalocation)
}

func validateSubresourceWithSchema(data interface{}, schemaPath string) (*gojsonschema.Result, error) {
	documentLoader := gojsonschema.NewGoLoader(data)
	mainSchema := gojsonschema.NewReferenceLoader(schemaPath)
	pMainSchema := &mainSchema
	schemaLoader := gojsonschema.NewSchemaLoader()
	schema, err := schemaLoader.Compile(*pMainSchema)
	if err != nil {
		glog.V(5).Infof("Error in schemacompile: %v\n", err)
		return nil, err
	}
	return schema.Validate(documentLoader)
}

// Validate a Kubernetes YAML file, parsing out individual resources
// and validating them all according to the  relevant schemas
// TODO This function requires a judicious amount of refactoring.
func Validate(config []byte, fileName string, version string, crds string) ([]ValidationResult, error) {
	results := make([]ValidationResult, 0)

	if len(config) == 0 {
		result := ValidationResult{}
		result.FileName = fileName
		results = append(results, result)
		return results, nil
	}

	bits := bytes.Split(config, []byte(detectLineBreak(config)+"---"+detectLineBreak(config)))
	crdMap := make(map[string]*gojsonschema.JSONLoader)
	if err := addBuiltinCRDs(crdMap); err != nil {
		return nil, err
	}
	addCRDs(crdMap, crds)

	var err error
	for _, element := range bits {
		if len(element) > 0 {
			var result *ValidationResult
			result, err = validateResource(element, fileName, version, crdMap)
			if err != nil {
				return results, err
			}
			// skip nil result since it can come from empty yaml section
			// we don't want to fail if yaml contain empty section
			// we only want to fail if the entire yaml is effectively empty
			if result != nil {
				results = append(results, *result)
			}
		}
	}
	// fail if entire yaml results is empty
	if err == nil && len(results) == 0 {
		result := ValidationResult{}
		result.FileName = fileName
		results = append(results, result)
		return results, nil
	}
	return results, err
}

func toLowerCaseMap(sa []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, s := range sa {
		m[strings.ToLower(s)] = struct{}{}
	}
	return m
}
