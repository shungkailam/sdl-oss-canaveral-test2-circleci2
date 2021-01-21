/*
 * Copyright (c) 2016 Nutanix Inc. All rights reserved.
 */

package errcode

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"reflect"
	"sort"

	"github.com/golang/glog"
)

var (
	errorCodeToConcreteType = make(map[uint64]reflect.Type)
)

// Facility identifies component which thru error
type Facility int

// ErrorCode implements error interface and stores error code along with facility
type ErrorCode interface {
	error
	GetTypeName() string
	GetCode() uint64
	GetHTTPStatus() int
	GetLocation() string
	GetFacility() Facility
	IsRetryable() bool
	GetUIErrorMessage(userLocale string) (string, error)
}

// ErrorCodeImpl is helper type for implementing various error codes
type ErrorCodeImpl struct {
	TypeName   string
	Code       uint64
	HTTPStatus int
	Location   string
	Facility   Facility
}

// Helper functions to unmarshal ErrorCode json(s)
func Register(code uint64, typ reflect.Type) {
	v := reflect.New(typ)
	gob.Register(v.Interface())
	errorCodeToConcreteType[code] = typ
}

func UnmarshalJSONs(b []byte) ([]ErrorCode, error) {
	errcodes := []ErrorCode{}
	msgs := []json.RawMessage{}
	err := json.Unmarshal(b, &msgs)
	if err != nil {
		glog.Errorf("Error unmarshalling : %s", err.Error())
		return errcodes, err
	}
	for _, msg := range msgs {
		errcode, err := UnmarshalJSON(msg)
		if err != nil {
			glog.Errorf("Error unmarshalling : %s", err.Error())
			return errcodes, err
		}
		errcodes = append(errcodes, errcode)
	}
	return errcodes, nil
}

func UnmarshalJSON(b []byte) (ErrorCode, error) {
	jsonStr := string(b)
	errcode := ErrorCodeImpl{}
	err := json.Unmarshal(b, &errcode)
	if err != nil {
		glog.Errorf("Error unmarshalling string %s : %s", jsonStr, err.Error())
		return nil, err
	}

	typ, ok := errorCodeToConcreteType[errcode.Code]
	if !ok {
		return nil, fmt.Errorf("Unknown error code unmarshalling string %s : %d", jsonStr, errcode.Code)
	}
	glog.V(5).Infof("[errcode.Code:%d] ErrorCodeToConcreteType got type : %s", errcode.Code, typ.Name())
	val := reflect.New(typ)
	err = json.Unmarshal(b, val.Interface())
	if err != nil {
		glog.Errorf("Error unmarshalling string %s : %s", jsonStr, err.Error())
		return nil, err
	}
	value, ok := val.Interface().(ErrorCode)
	if !ok {
		err = fmt.Errorf("Unmarshalled object (type:%s) does not satisfy interface ErrorCode", typ.Name())
		glog.Errorf("Error unmarshalling string %s : %s", jsonStr, err.Error())
		return nil, err
	}
	return value, nil
}

// GetTypeName implements ErrorCode interface
func (e ErrorCodeImpl) GetTypeName() string { return e.TypeName }

// GetCode implements ErrorCode interface
func (e ErrorCodeImpl) GetCode() uint64 { return e.Code }

// GetHTTPStatus implements ErrorCode interface
func (e ErrorCodeImpl) GetHTTPStatus() int { return e.HTTPStatus }

// GetLocation implements ErrorCode interface
func (e ErrorCodeImpl) GetLocation() string { return e.Location }

// GetFacility implements ErrorCode interface
func (e ErrorCodeImpl) GetFacility() Facility { return e.Facility }

// internal method to format error message for the UI
func formatUIErrorMessage(msg string, kv map[string]string) (string, error) {
	// Perform the template substitutions and return.

	var buf = bytes.NewBufferString("")
	tmpl, err := template.New("errMsg").Parse(msg)
	if err != nil {
		return "", err
	}
	err = tmpl.Execute(buf, kv)
	if err != nil {
		return "", err
	}
	return html.UnescapeString(buf.String()), nil
}

// FormatErrorMessage formats error message from msg template and key-val pairs
func FormatErrorMessage(msg string, code uint64, kv map[string]string) string {

	var kvs []string
	var buf = bytes.NewBufferString("[")
	var basicMsg = fmt.Sprintf("%s (error=0x%x)", msg, code)

	if len(kv) == 0 {
		return basicMsg
	}
	for k, v := range kv {
		kvs = append(kvs, fmt.Sprintf("%s=\"%s\", ", k, v))
	}
	// sort key-val pairs alphabetically
	sort.Sort(sort.StringSlice(kvs))
	for _, str := range kvs {
		buf.WriteString(str)
	}
	// truncate last comma away
	if len(kv) > 0 {
		buf.Truncate(buf.Len() - 2)
	}
	buf.WriteString("] ")
	buf.WriteString(basicMsg)
	return html.UnescapeString(buf.String())
}

type NotImplementedError struct {
	Op string
}

func (e *NotImplementedError) Error() string { return e.Op + ": Not implemented." }

type NotSupportedInVer1Error struct {
	Op string
}

func (e *NotSupportedInVer1Error) Error() string {
	return e.Op + ": Not supported in version 1 of the product."
}
