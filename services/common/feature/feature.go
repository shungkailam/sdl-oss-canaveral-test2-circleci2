package feature

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"fmt"
	"reflect"
	"strings"

	version "github.com/hashicorp/go-version"
)

// VersionRange keeps the min and max versions
type VersionRange struct {
	MinVersion *version.Version
	MaxVersion *version.Version
}

// Features keeps the features with effective versions
type Features struct {
	versions map[string]*VersionRange
}

// Add adds a feature with the effective version range
func (fv *Features) Add(name string, minVersion, maxVersion string) error {
	vRange := &VersionRange{}
	if strings.HasPrefix(minVersion, "v") {
		minVer, err := version.NewVersion(minVersion)
		if err != nil {
			return errcode.NewBadRequestExError("minVersion", err.Error())
		}
		vRange.MinVersion = minVer
	}
	if strings.HasPrefix(maxVersion, "v") {
		maxVer, err := version.NewVersion(maxVersion)
		if err != nil {
			return errcode.NewBadRequestExError("maxVersion", err.Error())
		}
		vRange.MaxVersion = maxVer
	}
	if vRange.MinVersion == nil && vRange.MaxVersion == nil {
		return errcode.NewBadRequestError("version")
	}
	if fv.versions == nil {
		fv.versions = map[string]*VersionRange{}
	}
	fv.versions[name] = vRange
	return nil
}

// Get sets the supported feature in the interface param.
// Name added in Add method must match the json field tags in the features
func (fv *Features) Get(ver string, features interface{}) error {
	iValue := reflect.ValueOf(features)
	if iValue.Type().Kind() != reflect.Ptr || reflect.Indirect(iValue).Type().Kind() != reflect.Struct {
		return errcode.NewBadRequestError("features")
	}
	if fv.versions == nil {
		return nil
	}
	inVer, err := version.NewVersion(ver)
	if err != nil {
		return errcode.NewBadRequestExError("version", err.Error())
	}
	fields := []string{}
	for name, verRange := range fv.versions {
		minVer := verRange.MinVersion
		maxVer := verRange.MaxVersion
		if minVer != nil && inVer.LessThan(minVer) {
			continue
		}
		if maxVer != nil && inVer.GreaterThan(maxVer) {
			continue
		}
		fields = append(fields, fmt.Sprintf("\"%s\": true", name))
	}
	jsonObj := fmt.Sprintf("{%s}", strings.Join(fields, ","))
	return base.ConvertFromJSON([]byte(jsonObj), features)
}
