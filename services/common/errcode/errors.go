// GENERATED DO NOT EDIT!

package errcode

import (
	"fmt"
	"github.com/golang/glog"
	"reflect"
	"runtime"
	"strings"
)

const (
	CommonFacility = Facility(1)
)

const (

	// COMMON errors
	BadRequest            = 0x5000
	DataConversion        = 0x5001
	RecordNotFound        = 0x5002
	InvalidCredentials    = 0x5003
	DatabaseDuplicate     = 0x5004
	DatabaseDependency    = 0x5005
	RecordInUse           = 0x5006
	PermissionDenied      = 0x5007
	InternalDatabase      = 0x5008
	Internal              = 0x5009
	MinValBadRequest      = 0x500a
	MaxValBadRequest      = 0x500b
	MinLenBadRequest      = 0x500c
	MaxLenBadRequest      = 0x500d
	MalformedBadRequest   = 0x500e
	WrongOptionBadRequest = 0x500f
	BadRequestEx          = 0x5010
	PreConditionFailed    = 0x5011
	RecordInUseEx         = 0x5012
	MalformedBadRequestEx = 0x5013
)

type BadRequestError struct {
	ErrorCodeImpl
	FieldName string
}

func NewBadRequestError(FieldName string) *BadRequestError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "BadRequestError",
		Code:       BadRequest,
		HTTPStatus: 400,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &BadRequestError{
		ErrorCodeImpl: impl,
		FieldName:     strings.Replace(FieldName, "\"", "\\\"", -1),
	}
}

func (e BadRequestError) Error() string {
	kv := make(map[string]string)
	kv["FieldName"] = e.FieldName

	kv["Location"] = e.Location
	return FormatErrorMessage("Invalid input data", BadRequest, kv)
}

func (e BadRequestError) IsRetryable() bool {
	return true
}

func (e BadRequestError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Invalid input data"

	switch userLocale {

	case "en_US":
		uiMessage = "Invalid input data for '{{.FieldName}}'."

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	kv["FieldName"] = e.FieldName

	return formatUIErrorMessage(uiMessage, kv)
}

type DataConversionError struct {
	ErrorCodeImpl
	Msg string
}

func NewDataConversionError(Msg string) *DataConversionError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "DataConversionError",
		Code:       DataConversion,
		HTTPStatus: 400,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &DataConversionError{
		ErrorCodeImpl: impl,
		Msg:           strings.Replace(Msg, "\"", "\\\"", -1),
	}
}

func (e DataConversionError) Error() string {
	kv := make(map[string]string)
	kv["Msg"] = e.Msg

	kv["Location"] = e.Location
	return FormatErrorMessage("Data conversion error", DataConversion, kv)
}

func (e DataConversionError) IsRetryable() bool {
	return true
}

func (e DataConversionError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Data conversion error"

	switch userLocale {

	case "en_US":
		uiMessage = "Error occurred in data conversion: '{{.Msg}}'."

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	kv["Msg"] = e.Msg

	return formatUIErrorMessage(uiMessage, kv)
}

type RecordNotFoundError struct {
	ErrorCodeImpl
	ID string
}

func NewRecordNotFoundError(ID string) *RecordNotFoundError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "RecordNotFoundError",
		Code:       RecordNotFound,
		HTTPStatus: 404,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &RecordNotFoundError{
		ErrorCodeImpl: impl,
		ID:            strings.Replace(ID, "\"", "\\\"", -1),
	}
}

func (e RecordNotFoundError) Error() string {
	kv := make(map[string]string)
	kv["ID"] = e.ID

	kv["Location"] = e.Location
	return FormatErrorMessage("Record not found error", RecordNotFound, kv)
}

func (e RecordNotFoundError) IsRetryable() bool {
	return true
}

func (e RecordNotFoundError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Record not found error"

	switch userLocale {

	case "en_US":
		uiMessage = "Cannot find a record with ID {{.ID}}."

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	kv["ID"] = e.ID

	return formatUIErrorMessage(uiMessage, kv)
}

type InvalidCredentialsError struct {
	ErrorCodeImpl
}

func NewInvalidCredentialsError() *InvalidCredentialsError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "InvalidCredentialsError",
		Code:       InvalidCredentials,
		HTTPStatus: 401,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &InvalidCredentialsError{
		ErrorCodeImpl: impl,
	}
}

func (e InvalidCredentialsError) Error() string {
	kv := make(map[string]string)

	kv["Location"] = e.Location
	return FormatErrorMessage("Cannot login due to an incorrect user name or password", InvalidCredentials, kv)
}

func (e InvalidCredentialsError) IsRetryable() bool {
	return true
}

func (e InvalidCredentialsError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Cannot login due to an incorrect user name or password"

	switch userLocale {

	case "en_US":
		uiMessage = "Cannot log in. User name or password is incorrect."

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	return formatUIErrorMessage(uiMessage, kv)
}

type DatabaseDuplicateError struct {
	ErrorCodeImpl
	ID string
}

func NewDatabaseDuplicateError(ID string) *DatabaseDuplicateError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "DatabaseDuplicateError",
		Code:       DatabaseDuplicate,
		HTTPStatus: 400,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &DatabaseDuplicateError{
		ErrorCodeImpl: impl,
		ID:            strings.Replace(ID, "\"", "\\\"", -1),
	}
}

func (e DatabaseDuplicateError) Error() string {
	kv := make(map[string]string)
	kv["ID"] = e.ID

	kv["Location"] = e.Location
	return FormatErrorMessage("Record duplicate error", DatabaseDuplicate, kv)
}

func (e DatabaseDuplicateError) IsRetryable() bool {
	return true
}

func (e DatabaseDuplicateError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Record duplicate error"

	switch userLocale {

	case "en_US":
		uiMessage = "We found a duplicate record with ID {{.ID}}."

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	kv["ID"] = e.ID

	return formatUIErrorMessage(uiMessage, kv)
}

type DatabaseDependencyError struct {
	ErrorCodeImpl
	ID string
}

func NewDatabaseDependencyError(ID string) *DatabaseDependencyError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "DatabaseDependencyError",
		Code:       DatabaseDependency,
		HTTPStatus: 400,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &DatabaseDependencyError{
		ErrorCodeImpl: impl,
		ID:            strings.Replace(ID, "\"", "\\\"", -1),
	}
}

func (e DatabaseDependencyError) Error() string {
	kv := make(map[string]string)
	kv["ID"] = e.ID

	kv["Location"] = e.Location
	return FormatErrorMessage("Record dependency error", DatabaseDependency, kv)
}

func (e DatabaseDependencyError) IsRetryable() bool {
	return true
}

func (e DatabaseDependencyError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Record dependency error"

	switch userLocale {

	case "en_US":
		uiMessage = "Record with ID {{.ID}} has an unsatisfied dependency."

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	kv["ID"] = e.ID

	return formatUIErrorMessage(uiMessage, kv)
}

type RecordInUseError struct {
	ErrorCodeImpl
}

func NewRecordInUseError() *RecordInUseError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "RecordInUseError",
		Code:       RecordInUse,
		HTTPStatus: 400,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &RecordInUseError{
		ErrorCodeImpl: impl,
	}
}

func (e RecordInUseError) Error() string {
	kv := make(map[string]string)

	kv["Location"] = e.Location
	return FormatErrorMessage("Record dependency error", RecordInUse, kv)
}

func (e RecordInUseError) IsRetryable() bool {
	return true
}

func (e RecordInUseError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Record dependency error"

	switch userLocale {

	case "en_US":
		uiMessage = "This record is being referenced in other records."

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	return formatUIErrorMessage(uiMessage, kv)
}

type PermissionDeniedError struct {
	ErrorCodeImpl
	Reason string
}

func NewPermissionDeniedError(Reason string) *PermissionDeniedError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "PermissionDeniedError",
		Code:       PermissionDenied,
		HTTPStatus: 403,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &PermissionDeniedError{
		ErrorCodeImpl: impl,
		Reason:        strings.Replace(Reason, "\"", "\\\"", -1),
	}
}

func (e PermissionDeniedError) Error() string {
	kv := make(map[string]string)
	kv["Reason"] = e.Reason

	kv["Location"] = e.Location
	return FormatErrorMessage("Permission Denied", PermissionDenied, kv)
}

func (e PermissionDeniedError) IsRetryable() bool {
	return true
}

func (e PermissionDeniedError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Permission Denied"

	switch userLocale {

	case "en_US":
		uiMessage = "You do not have the required permission: '{{.Reason}}'."

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	kv["Reason"] = e.Reason

	return formatUIErrorMessage(uiMessage, kv)
}

type InternalDatabaseError struct {
	ErrorCodeImpl
	Msg string
}

func NewInternalDatabaseError(Msg string) *InternalDatabaseError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "InternalDatabaseError",
		Code:       InternalDatabase,
		HTTPStatus: 500,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &InternalDatabaseError{
		ErrorCodeImpl: impl,
		Msg:           strings.Replace(Msg, "\"", "\\\"", -1),
	}
}

func (e InternalDatabaseError) Error() string {
	kv := make(map[string]string)
	kv["Msg"] = e.Msg

	kv["Location"] = e.Location
	return FormatErrorMessage("Internal database error", InternalDatabase, kv)
}

func (e InternalDatabaseError) IsRetryable() bool {
	return true
}

func (e InternalDatabaseError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Internal database error"

	switch userLocale {

	case "en_US":
		uiMessage = "An internal database error occurred: '{{.Msg}}'."

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	kv["Msg"] = e.Msg

	return formatUIErrorMessage(uiMessage, kv)
}

type InternalError struct {
	ErrorCodeImpl
	Msg string
}

func NewInternalError(Msg string) *InternalError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "InternalError",
		Code:       Internal,
		HTTPStatus: 500,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &InternalError{
		ErrorCodeImpl: impl,
		Msg:           strings.Replace(Msg, "\"", "\\\"", -1),
	}
}

func (e InternalError) Error() string {
	kv := make(map[string]string)
	kv["Msg"] = e.Msg

	kv["Location"] = e.Location
	return FormatErrorMessage("Internal error", Internal, kv)
}

func (e InternalError) IsRetryable() bool {
	return true
}

func (e InternalError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Internal error"

	switch userLocale {

	case "en_US":
		uiMessage = "An internal error occurred: '{{.Msg}}'."

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	kv["Msg"] = e.Msg

	return formatUIErrorMessage(uiMessage, kv)
}

type MinValBadRequestError struct {
	ErrorCodeImpl
	FieldName  string
	FieldValue string
}

func NewMinValBadRequestError(FieldName string, FieldValue string) *MinValBadRequestError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "MinValBadRequestError",
		Code:       MinValBadRequest,
		HTTPStatus: 400,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &MinValBadRequestError{
		ErrorCodeImpl: impl,
		FieldName:     strings.Replace(FieldName, "\"", "\\\"", -1),
		FieldValue:    strings.Replace(FieldValue, "\"", "\\\"", -1),
	}
}

func (e MinValBadRequestError) Error() string {
	kv := make(map[string]string)
	kv["FieldName"] = e.FieldName
	kv["FieldValue"] = e.FieldValue

	kv["Location"] = e.Location
	return FormatErrorMessage("Invalid value", MinValBadRequest, kv)
}

func (e MinValBadRequestError) IsRetryable() bool {
	return true
}

func (e MinValBadRequestError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Invalid value"

	switch userLocale {

	case "en_US":
		uiMessage = "Input value for '{{.FieldName}}' must be at least {{.FieldValue}}."

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	kv["FieldName"] = e.FieldName
	kv["FieldValue"] = e.FieldValue

	return formatUIErrorMessage(uiMessage, kv)
}

type MaxValBadRequestError struct {
	ErrorCodeImpl
	FieldName  string
	FieldValue string
}

func NewMaxValBadRequestError(FieldName string, FieldValue string) *MaxValBadRequestError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "MaxValBadRequestError",
		Code:       MaxValBadRequest,
		HTTPStatus: 400,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &MaxValBadRequestError{
		ErrorCodeImpl: impl,
		FieldName:     strings.Replace(FieldName, "\"", "\\\"", -1),
		FieldValue:    strings.Replace(FieldValue, "\"", "\\\"", -1),
	}
}

func (e MaxValBadRequestError) Error() string {
	kv := make(map[string]string)
	kv["FieldName"] = e.FieldName
	kv["FieldValue"] = e.FieldValue

	kv["Location"] = e.Location
	return FormatErrorMessage("Invalid value", MaxValBadRequest, kv)
}

func (e MaxValBadRequestError) IsRetryable() bool {
	return true
}

func (e MaxValBadRequestError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Invalid value"

	switch userLocale {

	case "en_US":
		uiMessage = "'{{.FieldName}}' must be less than or equal to {{.FieldValue}}."

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	kv["FieldName"] = e.FieldName
	kv["FieldValue"] = e.FieldValue

	return formatUIErrorMessage(uiMessage, kv)
}

type MinLenBadRequestError struct {
	ErrorCodeImpl
	FieldName  string
	FieldValue string
}

func NewMinLenBadRequestError(FieldName string, FieldValue string) *MinLenBadRequestError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "MinLenBadRequestError",
		Code:       MinLenBadRequest,
		HTTPStatus: 400,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &MinLenBadRequestError{
		ErrorCodeImpl: impl,
		FieldName:     strings.Replace(FieldName, "\"", "\\\"", -1),
		FieldValue:    strings.Replace(FieldValue, "\"", "\\\"", -1),
	}
}

func (e MinLenBadRequestError) Error() string {
	kv := make(map[string]string)
	kv["FieldName"] = e.FieldName
	kv["FieldValue"] = e.FieldValue

	kv["Location"] = e.Location
	return FormatErrorMessage("Invalid value", MinLenBadRequest, kv)
}

func (e MinLenBadRequestError) IsRetryable() bool {
	return true
}

func (e MinLenBadRequestError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Invalid value"

	switch userLocale {

	case "en_US":
		uiMessage = "'{{.FieldName}}' must be at least {{.FieldValue}} in length."

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	kv["FieldName"] = e.FieldName
	kv["FieldValue"] = e.FieldValue

	return formatUIErrorMessage(uiMessage, kv)
}

type MaxLenBadRequestError struct {
	ErrorCodeImpl
	FieldName  string
	FieldValue string
}

func NewMaxLenBadRequestError(FieldName string, FieldValue string) *MaxLenBadRequestError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "MaxLenBadRequestError",
		Code:       MaxLenBadRequest,
		HTTPStatus: 400,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &MaxLenBadRequestError{
		ErrorCodeImpl: impl,
		FieldName:     strings.Replace(FieldName, "\"", "\\\"", -1),
		FieldValue:    strings.Replace(FieldValue, "\"", "\\\"", -1),
	}
}

func (e MaxLenBadRequestError) Error() string {
	kv := make(map[string]string)
	kv["FieldName"] = e.FieldName
	kv["FieldValue"] = e.FieldValue

	kv["Location"] = e.Location
	return FormatErrorMessage("Invalid value", MaxLenBadRequest, kv)
}

func (e MaxLenBadRequestError) IsRetryable() bool {
	return true
}

func (e MaxLenBadRequestError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Invalid value"

	switch userLocale {

	case "en_US":
		uiMessage = "'{{.FieldName}}' must be less than or equal to {{.FieldValue}} in length."

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	kv["FieldName"] = e.FieldName
	kv["FieldValue"] = e.FieldValue

	return formatUIErrorMessage(uiMessage, kv)
}

type MalformedBadRequestError struct {
	ErrorCodeImpl
	FieldName string
}

func NewMalformedBadRequestError(FieldName string) *MalformedBadRequestError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "MalformedBadRequestError",
		Code:       MalformedBadRequest,
		HTTPStatus: 400,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &MalformedBadRequestError{
		ErrorCodeImpl: impl,
		FieldName:     strings.Replace(FieldName, "\"", "\\\"", -1),
	}
}

func (e MalformedBadRequestError) Error() string {
	kv := make(map[string]string)
	kv["FieldName"] = e.FieldName

	kv["Location"] = e.Location
	return FormatErrorMessage("Malformed value", MalformedBadRequest, kv)
}

func (e MalformedBadRequestError) IsRetryable() bool {
	return true
}

func (e MalformedBadRequestError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Malformed value"

	switch userLocale {

	case "en_US":
		uiMessage = "Malformed value for '{{.FieldName}}'."

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	kv["FieldName"] = e.FieldName

	return formatUIErrorMessage(uiMessage, kv)
}

type WrongOptionBadRequestError struct {
	ErrorCodeImpl
	FieldName  string
	FieldValue string
}

func NewWrongOptionBadRequestError(FieldName string, FieldValue string) *WrongOptionBadRequestError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "WrongOptionBadRequestError",
		Code:       WrongOptionBadRequest,
		HTTPStatus: 400,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &WrongOptionBadRequestError{
		ErrorCodeImpl: impl,
		FieldName:     strings.Replace(FieldName, "\"", "\\\"", -1),
		FieldValue:    strings.Replace(FieldValue, "\"", "\\\"", -1),
	}
}

func (e WrongOptionBadRequestError) Error() string {
	kv := make(map[string]string)
	kv["FieldName"] = e.FieldName
	kv["FieldValue"] = e.FieldValue

	kv["Location"] = e.Location
	return FormatErrorMessage("Wrong option", WrongOptionBadRequest, kv)
}

func (e WrongOptionBadRequestError) IsRetryable() bool {
	return true
}

func (e WrongOptionBadRequestError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Wrong option"

	switch userLocale {

	case "en_US":
		uiMessage = "Value for '{{.FieldName}}' must be one of {{.FieldValue}}."

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	kv["FieldName"] = e.FieldName
	kv["FieldValue"] = e.FieldValue

	return formatUIErrorMessage(uiMessage, kv)
}

type BadRequestExError struct {
	ErrorCodeImpl
	FieldName string
	Msg       string
}

func NewBadRequestExError(FieldName string, Msg string) *BadRequestExError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "BadRequestExError",
		Code:       BadRequestEx,
		HTTPStatus: 400,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &BadRequestExError{
		ErrorCodeImpl: impl,
		FieldName:     strings.Replace(FieldName, "\"", "\\\"", -1),
		Msg:           strings.Replace(Msg, "\"", "\\\"", -1),
	}
}

func (e BadRequestExError) Error() string {
	kv := make(map[string]string)
	kv["FieldName"] = e.FieldName
	kv["Msg"] = e.Msg

	kv["Location"] = e.Location
	return FormatErrorMessage("Invalid input data", BadRequestEx, kv)
}

func (e BadRequestExError) IsRetryable() bool {
	return true
}

func (e BadRequestExError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Invalid input data"

	switch userLocale {

	case "en_US":
		uiMessage = "{{.Msg}}"

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	kv["FieldName"] = e.FieldName
	kv["Msg"] = e.Msg

	return formatUIErrorMessage(uiMessage, kv)
}

type PreConditionFailedError struct {
	ErrorCodeImpl
	Msg string
}

func NewPreConditionFailedError(Msg string) *PreConditionFailedError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "PreConditionFailedError",
		Code:       PreConditionFailed,
		HTTPStatus: 412,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &PreConditionFailedError{
		ErrorCodeImpl: impl,
		Msg:           strings.Replace(Msg, "\"", "\\\"", -1),
	}
}

func (e PreConditionFailedError) Error() string {
	kv := make(map[string]string)
	kv["Msg"] = e.Msg

	kv["Location"] = e.Location
	return FormatErrorMessage("Precondition failed", PreConditionFailed, kv)
}

func (e PreConditionFailedError) IsRetryable() bool {
	return true
}

func (e PreConditionFailedError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Precondition failed"

	switch userLocale {

	case "en_US":
		uiMessage = "{{.Msg}}"

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	kv["Msg"] = e.Msg

	return formatUIErrorMessage(uiMessage, kv)
}

type RecordInUseExError struct {
	ErrorCodeImpl
	Record         string
	RefRecords     string
	RefRecordNames string
}

func NewRecordInUseExError(Record string, RefRecords string, RefRecordNames string) *RecordInUseExError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "RecordInUseExError",
		Code:       RecordInUseEx,
		HTTPStatus: 400,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &RecordInUseExError{
		ErrorCodeImpl:  impl,
		Record:         strings.Replace(Record, "\"", "\\\"", -1),
		RefRecords:     strings.Replace(RefRecords, "\"", "\\\"", -1),
		RefRecordNames: strings.Replace(RefRecordNames, "\"", "\\\"", -1),
	}
}

func (e RecordInUseExError) Error() string {
	kv := make(map[string]string)
	kv["Record"] = e.Record
	kv["RefRecords"] = e.RefRecords
	kv["RefRecordNames"] = e.RefRecordNames

	kv["Location"] = e.Location
	return FormatErrorMessage("Record dependency error", RecordInUseEx, kv)
}

func (e RecordInUseExError) IsRetryable() bool {
	return true
}

func (e RecordInUseExError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Record dependency error"

	switch userLocale {

	case "en_US":
		uiMessage = "{{.Record}} is being referenced in other {{.RefRecords}}: {{.RefRecordNames}}"

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	kv["Record"] = e.Record
	kv["RefRecords"] = e.RefRecords
	kv["RefRecordNames"] = e.RefRecordNames

	return formatUIErrorMessage(uiMessage, kv)
}

type MalformedBadRequestExError struct {
	ErrorCodeImpl
	FieldName string
	Msg       string
}

func NewMalformedBadRequestExError(FieldName string, Msg string) *MalformedBadRequestExError {
	_, file, line, _ := runtime.Caller(1)
	impl := ErrorCodeImpl{
		TypeName:   "MalformedBadRequestExError",
		Code:       MalformedBadRequestEx,
		HTTPStatus: 400,
		Location:   fmt.Sprintf("%s:%d", file, line),
		Facility:   CommonFacility,
	}

	return &MalformedBadRequestExError{
		ErrorCodeImpl: impl,
		FieldName:     strings.Replace(FieldName, "\"", "\\\"", -1),
		Msg:           strings.Replace(Msg, "\"", "\\\"", -1),
	}
}

func (e MalformedBadRequestExError) Error() string {
	kv := make(map[string]string)
	kv["FieldName"] = e.FieldName
	kv["Msg"] = e.Msg

	kv["Location"] = e.Location
	return FormatErrorMessage("Malformed value", MalformedBadRequestEx, kv)
}

func (e MalformedBadRequestExError) IsRetryable() bool {
	return true
}

func (e MalformedBadRequestExError) GetUIErrorMessage(userLocale string) (string, error) {
	uiMessage := "Malformed value"

	switch userLocale {

	case "en_US":
		uiMessage = "Malformed value for '{{.FieldName}}': {{.Msg}}"

	default:
		glog.Errorf("Unknown locale %s, will use default message %s", userLocale, uiMessage)
	}

	kv := make(map[string]string)

	kv["FieldName"] = e.FieldName
	kv["Msg"] = e.Msg

	return formatUIErrorMessage(uiMessage, kv)
}

func init() {

	Register(BadRequest, reflect.TypeOf(BadRequestError{}))

	Register(DataConversion, reflect.TypeOf(DataConversionError{}))

	Register(RecordNotFound, reflect.TypeOf(RecordNotFoundError{}))

	Register(InvalidCredentials, reflect.TypeOf(InvalidCredentialsError{}))

	Register(DatabaseDuplicate, reflect.TypeOf(DatabaseDuplicateError{}))

	Register(DatabaseDependency, reflect.TypeOf(DatabaseDependencyError{}))

	Register(RecordInUse, reflect.TypeOf(RecordInUseError{}))

	Register(PermissionDenied, reflect.TypeOf(PermissionDeniedError{}))

	Register(InternalDatabase, reflect.TypeOf(InternalDatabaseError{}))

	Register(Internal, reflect.TypeOf(InternalError{}))

	Register(MinValBadRequest, reflect.TypeOf(MinValBadRequestError{}))

	Register(MaxValBadRequest, reflect.TypeOf(MaxValBadRequestError{}))

	Register(MinLenBadRequest, reflect.TypeOf(MinLenBadRequestError{}))

	Register(MaxLenBadRequest, reflect.TypeOf(MaxLenBadRequestError{}))

	Register(MalformedBadRequest, reflect.TypeOf(MalformedBadRequestError{}))

	Register(WrongOptionBadRequest, reflect.TypeOf(WrongOptionBadRequestError{}))

	Register(BadRequestEx, reflect.TypeOf(BadRequestExError{}))

	Register(PreConditionFailed, reflect.TypeOf(PreConditionFailedError{}))

	Register(RecordInUseEx, reflect.TypeOf(RecordInUseExError{}))

	Register(MalformedBadRequestEx, reflect.TypeOf(MalformedBadRequestExError{}))

}
