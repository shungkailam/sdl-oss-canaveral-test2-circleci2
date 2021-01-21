/*
 * Copyright (c) 2017 Nutanix Inc. All rights reserved.
 * Generate error types from definitions.
 */

//go:generate bash -c "go run generate.go -o ../errors.go ../resources/*.json"

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"text/template"
)

var (
	outputGoFile     = flag.String("o", "output.go", "Path to generated Go file")
	PackagesToImport = []string{
		"fmt",
		"reflect",
		"runtime",
		"strings",
		"github.com/golang/glog",
	}
)

const (
	// translate errorSpec to Go code
	errorBlock = `
		type {{.TypeName}} struct {
			ErrorCodeImpl
			{{range .Keys}} {{.}} string
			{{end}}
		}

		func New{{.TypeName}}({{range .Keys}}{{.}} string, {{end}}) *{{.TypeName}} {
			_, file, line, _ := runtime.Caller(1)
			impl := ErrorCodeImpl{
				TypeName: "{{.TypeName}}",
				Code: {{.ErrorCodeConst}},
				HTTPStatus: {{.HTTPStatus}},
				Location: fmt.Sprintf("%s:%d", file, line),
				Facility: {{.Facility}},
			}

			return &{{.TypeName}} {
				ErrorCodeImpl: impl,
				{{range .Keys}}{{.}}: strings.Replace({{.}}, "\"", "\\\"", -1),
				{{end}}
			}
		}

		func (e {{.TypeName}}) Error() string {
			kv := make(map[string]string)
			{{range .Keys}}kv["{{.}}"] = e.{{.}}
			{{end}}
			kv["Location"] = e.Location
			return FormatErrorMessage("{{.Msg}}", {{.ErrorCodeConst}}, kv)
		}

		func (e {{.TypeName}}) IsRetryable() bool {
			return {{.IsRetryable}}
		}`

	errorUIFuncBlock = `
		func (e {{.TypeName}}) GetUIErrorMessage(userLocale string) (string,error) {
			uiMessage:= "{{.Msg}}"
			{{if .UIMsg}}
				switch userLocale{
				{{range $key, $val := .UIMsg}}
					case "{{$key}}":
						uiMessage = "{{$val}}"
				{{end}}
				default:
					glog.Errorf("Unknown locale %s, will use default message %s",userLocale,uiMessage)
				}

			{{end}}

			kv := make(map[string]string)
			{{if .Keys}}
				{{range .Keys}}kv["{{.}}"] = e.{{.}}
				{{end}}
			{{end}}
			return formatUIErrorMessage(uiMessage, kv)
		}`
	// translate []errorConf to init()
	initFuncBlock = `
		func init() {
			{{range .}} {{range .Errors}}
			Register({{.Name}}, reflect.TypeOf({{.TypeName}}{}))
			{{end}}{{end}}
		}`
)

type errorSpec struct {
	// facility error code belongs to (not encoded in json)
	Facility string
	// name w/o spaces in upper-camel case
	Name string
	// log message
	Msg string
	// http status code
	HTTPStatus uint
	// can operation be retried?
	IsPermanent bool
	//UI message
	UIMsg map[string]string
	// error code specific keys
	// Keys will be translated to struct fields
	Keys []string
}

func (spec errorSpec) ErrorCodeConst() string {
	return spec.Name
}

func (spec errorSpec) TypeName() string {
	return spec.Name + "Error"
}

func (spec errorSpec) IsRetryable() bool {
	return !spec.IsPermanent
}

type errorConf struct {
	// facility error code belongs to
	Facility string
	// hexadecimal error code offset
	ErrorCodeOffset string
	// list of errors in this facility
	Errors []errorSpec
}

func (conf *errorConf) FacilityName() string {
	return conf.Facility + "Facility"
}

func generateFacilityEnum(w *os.File, confs []errorConf) {
	var off = 0
	var facilities []string

	w.WriteString("const (\n")
	for _, conf := range confs {
		facilities = append(facilities, conf.FacilityName())
	}
	sort.Sort(sort.StringSlice(facilities))
	for _, facility := range facilities {
		off++
		fmt.Fprintf(w, "%s = Facility(%d)\n", facility, off)
	}
	w.WriteString(")\n\n")
}

func generateErrorEnum(w *os.File, confs []errorConf) {
	w.WriteString("const (\n")
	for _, conf := range confs {
		var off uint64

		_, err := fmt.Sscanf(conf.ErrorCodeOffset, "0x%x", &off)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(w, "\n// %s errors\n", strings.ToUpper(conf.Facility))
		for _, spec := range conf.Errors {
			fmt.Fprintf(w, "%s = 0x%x\n", spec.ErrorCodeConst(), off)
			off++
		}
	}
	w.WriteString(")\n\n")
}

func process(out *os.File, confs []errorConf) {
	var errorTempl, errorUIFuncTempl, initFuncTempl *template.Template

	errorTempl, err := template.New("errorTempl").Parse(errorBlock)
	if err != nil {
		log.Fatal(err)
	}

	errorUIFuncTempl, err = template.New("errorUIFuncTempl").Parse(errorUIFuncBlock)
	if err != nil {
		log.Fatal(err)
	}

	initFuncTempl, err = template.New("initFuncTempl").Parse(initFuncBlock)
	if err != nil {
		log.Fatal(err)
	}

	// write header
	out.WriteString("// GENERATED DO NOT EDIT!\n\n")
	out.WriteString("package errcode\n\n")

	out.WriteString("import (\n")
	for _, pkg := range PackagesToImport {
		out.WriteString(`"` + pkg + `"
		`)
	}
	out.WriteString(")\n")

	// write facilties enum
	generateFacilityEnum(out, confs)

	// write error enum
	generateErrorEnum(out, confs)

	for _, conf := range confs {
		for _, spec := range conf.Errors {
			spec.Facility = conf.FacilityName()
			errorTempl.Execute(out, &spec)
			out.WriteString("\n")
			errorUIFuncTempl.Execute(out, &spec)
			out.WriteString("\n")
		}
	}

	// write init
	initFuncTempl.Execute(out, &confs)
}

func main() {
	var confs []errorConf

	flag.Parse()

	out, err := os.OpenFile(*outputGoFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	for _, input := range flag.Args() {
		var tmpConfs []errorConf

		f, err := os.Open(input)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Reading", input)
		dec := json.NewDecoder(f)
		err = dec.Decode(&tmpConfs)
		if err != nil {
			log.Fatal(err)
		}
		confs = append(confs, tmpConfs...)
		f.Close()
	}

	// process all error facilities
	process(out, confs)

	// go fmt output
	cmd := exec.Command("go", "fmt", *outputGoFile)
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
