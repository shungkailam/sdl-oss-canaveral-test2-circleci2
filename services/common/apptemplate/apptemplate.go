package apptemplate

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"text/template"

	"regexp"
	"strings"
)

// Some complex yaml might contain nested {{}}
// (see, e.g., prometheusOperator.yaml)
// Note: regex being a DFA can't deal with arbitrary nesting!
// Here we will take a simple approach and
// (1) only deal with one level nesting,
// (2) escape such with a print function
const leftDouble = `{{`

// NoDouble - no '{{' or '}}',
// (actually a bit more restrictive)
// can be: zero or more of
// '{' followed by one or more non-'{}'
// '}' followed by one or more non-'{}'
// a non-'{}' char
const NoDouble = `(?:{[^{}]+|}[^{}]+|[^{}])*?`

// we don't support .Variable substitution inside nested double
const NestedDouble = `(?:{{[^{}]+}}[^{}]*)+`
const rightDouble = `}}`

const NestTemplate = leftDouble + NoDouble + NestedDouble + NoDouble + rightDouble

var reNest = regexp.MustCompile(NestTemplate)

// we will escape template which is not simple value substitution form,
// i.e., templates not of the form: {{ .Some.Name }}
// template = '{{' followed by any number of non-newline whitespaces,
//            followed by one char that is NOT in '{}.' or whitespace,
//            followed by zero or more chars that is NOT in '{}' followed by
//            '}}'
const DoubleTemplate = `({{[\t\f ]*[^{}.\s][^{}]*}})`

var reTemplate = regexp.MustCompile(DoubleTemplate)

type EdgeService struct {
	Endpoint string
}

// EdgeParameters are edge-specific settings originating at
// edge.
type EdgeParameters struct {
	Services map[string]EdgeService
}

// All parameters which are specific to an application
// on a specific edge.
type AppParameters struct {
	EdgeParameters
	EdgeName    string
	EdgeID      string
	ProjectID   string
	ProjectName string
	AppID       string
	AppName     string
	AppVersion  string
	Categories  map[string]string
	Env         *string
	Namespace   string
	Parameters  map[string]string
}

// Define functions bound to app template
var funcMap = template.FuncMap{
	"sha1": func(args ...string) (string, error) {
		hash := sha1.New()
		for _, arg := range args {
			_, err := hash.Write([]byte(arg))
			if err != nil {
				return "", err
			}
		}
		return fmt.Sprintf("%x", hash.Sum(nil)), nil
	},
}

func escapeTemplate(s string) string {
	return reTemplate.ReplaceAllString(s, "{{`$1`}}")
}

func escapePrint(s string) string {
	return fmt.Sprintf("{{print `%s`}}", strings.Join(strings.Split(s, "`"), "` \"`\" `"))
}

func escapeTemplates(s string) string {
	lines := []string{}
	results := []string{}
	ss := strings.Split(s, "\n")
	for _, line := range ss {
		if asm := reNest.FindAllSubmatch([]byte(line), -1); len(asm) != 0 {
			sm := string(asm[0][0])
			psfx := strings.Split(line, sm)
			ep := escapePrint(sm)
			rlines := escapeTemplate(strings.Join(lines, "\n"))
			pfx := escapeTemplate(psfx[0])
			sfx := escapeTemplate(psfx[1])
			results = append(results, rlines, fmt.Sprintf("%s%s%s", pfx, ep, sfx))
			lines = []string{}
		} else {
			lines = append(lines, line)
		}
	}
	if len(lines) != 0 {
		rlines := escapeTemplate(strings.Join(lines, "\n"))
		results = append(results, rlines)
	}
	return strings.Join(results, "\n")
}

// RenderWithParams() renders application YAML using golang template engine.
func RenderWithParams(params *AppParameters, in string) (out string, services []string, err error) {
	services = []string{}
	// Execute YAML template
	buf := bytes.Buffer{}
	in = escapeTemplates(in)
	tmpl, err := template.New("Application").Funcs(funcMap).Parse(in)
	if err != nil {
		return "", nil, err
	}
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", nil, err
	}
	out = buf.String()
	// Figure out which service has been referenced. This can be used by
	// an edge to lazily instantiate services.
	for name, svc := range params.Services {
		buf := bytes.Buffer{}
		params.Services[name] = EdgeService{}
		err = tmpl.Execute(&buf, params)
		// We either failed rendering template or content differs.
		// Eithe way service must have been referenced.
		if err != nil || out != buf.String() {
			services = append(services, name)
		}
		params.Services[name] = svc
	}
	if params.Env != nil {
		out, err = AddEnvsToYamlMaybe(params.AppID, out, params.Env)
		if err != nil {
			return "", nil, err
		}
	}
	return out, services, nil
}
