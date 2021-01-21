package apptemplate_test

import (
	"regexp"
	"testing"

	"cloudservices/common/apptemplate"

	"github.com/stretchr/testify/require"
)

func TestAppRender(t *testing.T) {
	// note: we no longer support functions in our yaml rendering
	const yaml = `
	apiVersion: v1
	kind: ConfigMap
	metadata:
	  name: web-content
	  annotations:
		sha1: "{{(print .AppVersion .AppID | sha1)}}"
		bytes: "{{ (	print .AppVersion .AppID)}}"
	data:
	  index.html: |
		EdgeName={{ .EdgeName}}
		EdgeID={{	.EdgeID}}
		AppID={{  .AppID}}
		AppName={{ 	.AppName}}
		AppVersion={{.AppVersion}}
		ProjectName={{.ProjectName}}
		ProjectID={{.ProjectID}}
		Namespace={{.Namespace}}
		TestEdgeCategory={{.Categories.TestEdgeCategory}}
		TestEscapingCategory={{.Categories.TestEscapingCategory}}
		KafkaSvc={{.Services.Kafka.Endpoint}}
		Parameter={{.Parameters.Param1}}
	`
	const expectedYaml = `
	apiVersion: v1
	kind: ConfigMap
	metadata:
	  name: web-content
	  annotations:
		sha1: "{{(print .AppVersion .AppID | sha1)}}"
		bytes: "{{ (	print .AppVersion .AppID)}}"
	data:
	  index.html: |
		EdgeName=edge-name
		EdgeID=edge-id
		AppID=app-id
		AppName=app-name
		AppVersion=42
		ProjectName=project-name
		ProjectID=XYZ
		Namespace=project-XYZ
		TestEdgeCategory=test-edge-category
		TestEscapingCategory=["test","escaping","category","<test>"]
		KafkaSvc=kafka-svc
		Parameter=value1
	`
	params := apptemplate.AppParameters{
		EdgeParameters: apptemplate.EdgeParameters{
			Services: map[string]apptemplate.EdgeService{
				"Kafka": {Endpoint: "kafka-svc"},
				"NATS":  {Endpoint: "nats-svc"},
			},
		},
		EdgeName:    "edge-name",
		EdgeID:      "edge-id",
		ProjectID:   "XYZ",
		Namespace:   "project-XYZ",
		ProjectName: "project-name",
		AppID:       "app-id",
		AppName:     "app-name",
		AppVersion:  "42",
		Categories: map[string]string{
			"TestEdgeCategory":     "test-edge-category",
			"TestEscapingCategory": "[\"test\",\"escaping\",\"category\",\"<test>\"]",
		},
		Parameters: map[string]string{
			"Param1": "value1",
		},
	}
	out, edgeServices, err := apptemplate.RenderWithParams(&params, yaml)
	require.NoError(t, err)
	if out != expectedYaml {
		t.Fatalf("App not rendered correctly. Expected '%s' got '%s'",
			expectedYaml, out)
	}
	if len(edgeServices) != 1 {
		t.Fatal("Expected one edge services to be referenced")
	}
	if edgeServices[0] != "Kafka" {
		t.Fatalf("Expected kafka to be referenced not %q", edgeServices[0])
	}
}

func TestNoDouble(t *testing.T) {
	var re = regexp.MustCompile("^" + apptemplate.NoDouble + "$")
	tss := []struct {
		s     string
		match bool
	}{
		{"", true},
		{"{", false},
		{"}", false},
		{"{ ", true},
		{"} ", true},
		{"{ {", false},
		{"} }", false},
		{"{ { ", true},
		{"} } ", true},
		{"{}", false},
		{"}{", false},
		{"{ }", false},
		{"{ } ", true},
		{" { } ", true},
		{"  {  } ", true},
		{"  {   } ", true},
		{"  {  }  { } ", true},
		{"{{", false},
		{"}}", false},
		{" {{", false},
		{" {{ ", false},
		{" {{ ", false},
		{"  {{ ", false},
		{"{{ }", false},
		{"{{ } ", false},
		{"{{ }}", false},
	}
	for _, ts := range tss {
		require.Equal(t, ts.match, re.MatchString(ts.s))
	}
}

func TestNestedDouble(t *testing.T) {
	var re = regexp.MustCompile("^" + apptemplate.NestedDouble + "$")
	tss := []struct {
		s     string
		match bool
	}{
		{"{}", false},
		{"{{}}", false},
		{"{{ }}", true},
		{" {{ }}", false},
		{"{{ }} ", true},
		{"{{ }}{{ }}", true},
		{"{{ }}{{ }} ", true},
		{"{{ }} {{ }} ", true},
	}
	for _, ts := range tss {
		require.Equal(t, ts.match, re.MatchString(ts.s))
	}
}
func TestNestTemplate(t *testing.T) {
	var re = regexp.MustCompile("^" + apptemplate.NestTemplate + "$")
	tss := []struct {
		s     string
		match bool
	}{
		{"{{ }}", false},
		{"{{ {{ }} }}", true},
		{"{{  { {{ }}  } }}", true},
		{"{{  {  } {{ }}  }}", true},
		{"{{  {  {{  } }}  }}", false},
		{"{{ {{ }}  {{ }} }}", true},
	}
	for _, ts := range tss {
		require.Equal(t, ts.match, re.MatchString(ts.s))
	}
}
func TestDoubleTemplate(t *testing.T) {
	var re = regexp.MustCompile("^" + apptemplate.DoubleTemplate + "$")
	tss := []struct {
		s     string
		match bool
	}{
		{"{{}}", false},
		{"{{ }}", false},
		{"{{.}}", false},
		{"{{a}}", true},
		{"{{ a}}", true},
		{"{{ a }}", true},
		{"{{ a. }}", true},
		{"{{ .a }}", false},
		{"{{ { }}", false},
	}
	for _, ts := range tss {
		require.Equal(t, ts.match, re.MatchString(ts.s))
	}
}
