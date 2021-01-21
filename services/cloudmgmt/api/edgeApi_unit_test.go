package api

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"errors"

	"context"
	"testing"
)

func TestGenerateAndSetShortID(t *testing.T) {
	t.Parallel()
	origNamedExec := namedExec
	defer func() { namedExec = origNamedExec }()

	callCount := 0
	testCases := []struct {
		namedExec                            func(*base.WrappedTx, context.Context, string, interface{}) error
		numAtttempts, expectedNumTxnAttempts int
		errExpected, shortIDExpected         bool
	}{
		// Case 1: Short ID on first attempt
		{
			namedExec: func(tx *base.WrappedTx, ctx context.Context, query string, obj interface{}) error {
				callCount++
				return nil
			},
			numAtttempts: 10, expectedNumTxnAttempts: 1, errExpected: false, shortIDExpected: true,
		},
		// Case 2: Non duplicate error,  hence, exit the loop earlier
		{
			namedExec: func(tx *base.WrappedTx, ctx context.Context, query string, obj interface{}) error {
				callCount++
				return errors.New("random error")
			},
			numAtttempts: 10, expectedNumTxnAttempts: 1, errExpected: true, shortIDExpected: false,
		},
		// Case 3: Dupicate error, try for the given number of attempts
		{
			namedExec: func(tx *base.WrappedTx, ctx context.Context, query string, obj interface{}) error {
				callCount++
				return errcode.NewDatabaseDuplicateError("foo")
			},
			numAtttempts: 2, expectedNumTxnAttempts: 2, errExpected: true, shortIDExpected: false,
		},
	}

	for _, testCase := range testCases {
		namedExec = testCase.namedExec
		testObj := &EdgeClusterDBO{}
		callCount = 0
		err := generateAndSetShortIDForEdgeCluster(context.Background(), &base.WrappedTx{}, testObj, testCase.numAtttempts)
		if callCount != testCase.expectedNumTxnAttempts {
			t.Fatalf("expected num of transaction attempts to be %d, but got %d", testCase.expectedNumTxnAttempts, callCount)
		}
		if err != nil != testCase.errExpected {
			t.Fatalf("expected %v but got %v", testCase.errExpected, err != nil)
		}
		if testObj.ShortID != nil != testCase.shortIDExpected {
			t.Fatalf("expected %v but got %v", testCase.shortIDExpected, testObj.ShortID != nil)
		}
	}
}
