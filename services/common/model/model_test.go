package model_test

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const (
	NOW string = "2018-01-01T01:01:01Z"
)

// A bunch of helpers for model tests
func timeNow(t *testing.T) time.Time {
	n, err := time.Parse(time.RFC3339, NOW)
	require.NoError(t, err)
	return n
}
