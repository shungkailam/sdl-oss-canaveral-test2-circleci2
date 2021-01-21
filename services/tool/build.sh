#!/bin/bash -x

function prepare {
  SERVICE_DIR=`pwd`
  echo "Preparing $SERVICE_DIR"
  GRPC_OUT=${SERVICE_DIR}/generated/grpc
  mkdir ./build
  mkdir -p "$GRPC_OUT"
}

function build {
  SERVICE_DIR=`pwd`
  echo "Building $SERVICE_DIR"
  go build  -ldflags '-w -s' -a -installsuffix cgo -o ./build/supportlogcleaner ./supportlogcleaner/main.go
  go build  -ldflags '-w -s' -a -installsuffix cgo -o ./build/softwareupdatecleaner ./softwareupdatecleaner/main.go
  go build  -ldflags '-w -s' -a -installsuffix cgo -o ./build/serviceclassupserter ./serviceclassupserter/main.go
  go build  -ldflags '-w -s' -a -installsuffix cgo -o ./build/athenadatauploader ./athenadatauploader/main.go
  echo "Built go source files"
}

function clean {
  SERVICE_DIR=`pwd`
  echo "Cleaning $SERVICE_DIR"
  rm -rf ./build
}

function package {
  SERVICE_DIR=`pwd`
  echo "Packaging $SERVICE_DIR"
}
