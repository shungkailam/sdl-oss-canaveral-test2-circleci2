#!/bin/bash -x

function prepare {
  SERVICE_DIR=`pwd`
  echo "Preparing $SERVICE_DIR"
  PROTO_SRC=${SERVICE_DIR}/proto
  GRPC_OUT=${SERVICE_DIR}/generated/grpc
  mkdir ./build
  mkdir -p "$GRPC_OUT"
  protoc -I$PROTO_SRC --go_out=plugins=grpc:$GRPC_OUT "$PROTO_SRC/devtools.proto"
  echo "Generated protobuf stubs"

  SWAGGER_SRC=${SERVICE_DIR}/devtools_swagger.json
  SWAGGER_OUT=${SERVICE_DIR}/generated/swagger
  mkdir -p "$SWAGGER_OUT"
  /tmp/swagger generate server -t $SWAGGER_OUT \
        -f $SWAGGER_SRC
  echo "Generated devtools swagger code"
}

function build {
  SERVICE_DIR=`pwd`
  echo "Building $SERVICE_DIR"
  go build  -ldflags '-w -s' -a -installsuffix cgo -o ./build/devtools ./cmd/main.go
  echo "Built go source files"
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
  cp ./devtools_swagger.json "$1/devtools_swagger.json"
}
