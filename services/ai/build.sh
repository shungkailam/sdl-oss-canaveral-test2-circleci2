#!/bin/bash -x

function prepare {
  SERVICE_DIR=`pwd`
  echo "Preparing $SERVICE_DIR"
  PROTO_SRC=${SERVICE_DIR}/proto
  GRPC_OUT=${SERVICE_DIR}/generated/golang/grpc
  mkdir -p "$GRPC_OUT"
  protoc -I$PROTO_SRC --go_out=plugins=grpc:$GRPC_OUT "$PROTO_SRC/mlmodel.proto"
  echo "Generated protobuf stubs"
}

function build {
  echo "Building AI service dockerfile"
  pushd ..
  docker build -t ai:build -f $CLOUD_SERVICES_PARENT/ai/Dockerfile $CLOUD_SERVICES_PARENT
  popd
}

function clean {
  SERVICE_DIR=`pwd`
  echo "Cleaning $SERVICE_DIR"
  rm -rf ./generated
}

function package {
  SERVICE_DIR=`pwd`
  echo "Packaging $SERVICE_DIR"
}
