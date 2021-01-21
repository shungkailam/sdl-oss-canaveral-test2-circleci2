#!/bin/bash
set -e
FILE_DIR=`dirname "$0"`
pushd "${FILE_DIR}"
SCRIPT_PATH=`pwd`
popd
pushd "${SCRIPT_PATH}"
go run "${SCRIPT_PATH}/generator/generate.go" -o errors.go resources/*.json
popd