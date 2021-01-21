#!/bin/bash -x

function prepare {
  SERVICE_DIR=`pwd`
  echo "Preparing $SERVICE_DIR"
  PROTO_SRC=${SERVICE_DIR}/proto
  GRPC_OUT=${SERVICE_DIR}/generated/grpc
  mkdir ./build
  mkdir -p "$GRPC_OUT"
  protoc -I$PROTO_SRC --go_out=plugins=grpc:$GRPC_OUT "$PROTO_SRC/event.proto"
  echo "Generated protobuf stubs"
}

function build {
  SERVICE_DIR=`pwd`
  echo "Building $SERVICE_DIR"
  go build  -ldflags '-w -s' -a -installsuffix cgo -o ./build/eventserver ./cmd/main.go
  echo "Built go source files"
  if [[ $COVERAGE_BUILD -eq 1 ]]; then
    cd cmd && go test -ldflags '-w -s' -a -installsuffix cgo -c -o ../build/eventserver_cov --coverpkg ../... && cd -
    echo "Built eventserver coverage binary"
  fi
}

function clean {
  SERVICE_DIR=`pwd`
  echo "Cleaning $SERVICE_DIR"
  rm -rf ./build
  rm -rf ./generated
}

function package {
  SERVICE_DIR=`pwd`
  echo "Packaging $SERVICE_DIR"
}
