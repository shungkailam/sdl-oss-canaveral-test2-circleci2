package errcode

import (
	"strings"

	"github.com/olivere/elastic"
)

type SQLErrorType uint

const (
	UNKNOWN                 = SQLErrorType(0)
	DUPLICATE_RECORD        = SQLErrorType(1)
	UNSATISIFIED_DEPENDENCY = SQLErrorType(2)
)

func GetSQLErrorType(err error) SQLErrorType {
	msg := err.Error()
	if strings.Contains(msg, "violates foreign key constraint") {
		return UNSATISIFIED_DEPENDENCY
	}
	if strings.Contains(msg, "duplicate key value violates unique constraint") {
		return DUPLICATE_RECORD
	}
	return UNKNOWN
}

func TranslateDatabaseError(ID string, err error) error {
	if _, ok := err.(ErrorCode); ok {
		return err
	}
	sqlErrorType := GetSQLErrorType(err)
	if sqlErrorType == DUPLICATE_RECORD {
		return NewDatabaseDuplicateError(ID)
	}
	if sqlErrorType == UNSATISIFIED_DEPENDENCY {
		return NewDatabaseDependencyError(ID)
	}
	return NewInternalDatabaseError(err.Error())
}

func TranslateSearchError(ID string, err error) error {
	if e, ok := err.(*elastic.Error); ok {
		if e.Details != nil && strings.Contains(e.Details.Type, "not_found_exception") {
			return NewBadRequestError(ID)
		}
	}
	return NewInternalDatabaseError(err.Error())
}

// IsDependencyConstraintError returns true if the error is database dependency constraint error
func IsDependencyConstraintError(err error) bool {
	_, ok := err.(*DatabaseDependencyError)
	if ok {
		return true
	}
	_, ok = err.(*RecordInUseError)
	if ok {
		return true
	}
	return false
}

// IsDuplicateRecordError checks if the given error is a duplicate error
func IsDuplicateRecordError(err error) bool {
	if _, ok := err.(*DatabaseDuplicateError); ok {
		return true
	}
	return false
}

// IsRecordNotFound checks if the given error is a record not found error
func IsRecordNotFound(err error) bool {
	if _, ok := err.(*RecordNotFoundError); ok {
		return true
	}
	return false
}

// IsBadRequestExError checks if the given error is of type `BadRequestExError`
func IsBadRequestExError(err error) bool {
	if _, ok := err.(*BadRequestExError); ok {
		return true
	}
	return false
}
