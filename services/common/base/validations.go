package base

import (
	"cloudservices/common/errcode"
	"regexp"
	"strings"

	version "github.com/hashicorp/go-version"
)

const dns1123LabelFmt string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"

// DNS1123LabelMaxLength is a label's max length in DNS (RFC 1123)
const DNS1123LabelMaxLength int = 63
const labelValueFmt string = "(" + qualifiedNameFmt + ")?"
const qnameCharFmt string = "[A-Za-z0-9]"
const qnameExtCharFmt string = "[-A-Za-z0-9_.]"
const qualifiedNameFmt string = "(" + qnameCharFmt + qnameExtCharFmt + "*)?" + qnameCharFmt

var qualifiedNameRegexp = regexp.MustCompile("^" + qualifiedNameFmt + "$")

// LabelValueMaxLength is a label's max length
const LabelValueMaxLength int = 63

var labelValueRegexp = regexp.MustCompile("^" + labelValueFmt + "$")

var dns1123LabelRegexp = regexp.MustCompile("^" + dns1123LabelFmt + "$")

// IsDNS1123Label asserts if the given value passes rfc1123(https://tools.ietf.org/html/rfc1123)
func IsDNS1123Label(value string) bool {
	if len(value) > DNS1123LabelMaxLength {
		return false
	}
	if !dns1123LabelRegexp.MatchString(value) {
		return false
	}
	return true
}

// IsValidLabelValue checks is the given value is a valid k8s label
// RFC: Should we even add it here as this is k8s specific?
func IsValidLabelValue(value string) bool {
	if len(value) > LabelValueMaxLength {
		return false
	}
	if !labelValueRegexp.MatchString(value) {
		return false
	}
	return true
}

// IsValidIP4 returns true if ipAddress is a valid IPv4 address
func IsValidIP4(ipAddress string) bool {
	return reIP4.MatchString(strings.TrimSpace(ipAddress))
}

// ValidateVersion validates a version string. Valid version looks like v1.2.3
func ValidateVersion(ver string) error {
	_, err := ValidateVersionEx(ver)
	if err != nil {
		return err
	}
	return nil
}

// ValidateVersionEx validates a version string returning the version object if it succeeds.
// Valid version looks like v1.2.3
func ValidateVersionEx(ver string) (*version.Version, error) {
	versionObj, err := version.NewVersion(ver)
	if err != nil {
		return nil, errcode.NewBadRequestExError("version", err.Error())
	}
	return versionObj, nil
}

// CompareVersions compares two versions. -1, 0 or 1 is returned depending on
// if the first is lesser, equal or greater than the second version respectively
func CompareVersions(first string, second string) (int, error) {
	firstVer, err := version.NewVersion(first)
	if err != nil {
		return 0, errcode.NewBadRequestExError("version", err.Error())
	}
	secondVer, err := version.NewVersion(second)
	if err != nil {
		return 0, errcode.NewBadRequestExError("version", err.Error())
	}
	result := firstVer.Compare(secondVer)
	return result, nil
}
