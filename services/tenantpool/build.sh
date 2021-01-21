#!/bin/bash -x

function prepare {
  SERVICE_DIR=`pwd`
  echo "Preparing $SERVICE_DIR"
  PROTO_SRC=${SERVICE_DIR}/proto
  SWAGGER_SRC=${SERVICE_DIR}/swagger
  GRPC_OUT=${SERVICE_DIR}/generated/grpc
  SWAGGER_OUT=${SERVICE_DIR}/generated/swagger
  mkdir ./build
  mkdir -p "$GRPC_OUT"
  mkdir -p "$SWAGGER_OUT"
  protoc -I$PROTO_SRC --go_out=plugins=grpc:$GRPC_OUT "$PROTO_SRC/tenantpool.proto"
  echo "Generated protobuf stubs"
  swagger generate client -t $SWAGGER_OUT -f ${SWAGGER_SRC}/bott_swagger.yml -A bottService
  echo "Generated bottservice swagger client"
}

function build {
  SERVICE_DIR=`pwd`
  echo "Building $SERVICE_DIR"
  go build  -ldflags '-w -s' -a -installsuffix cgo -o ./build/tenantpoolserver ./cmd/main.go
  echo "Built go source files"
  go build  -ldflags '-w -s' -a -installsuffix cgo -o ./build/tenantpoolcli ./cli/main.go
  echo "Built go CLI source files"
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
