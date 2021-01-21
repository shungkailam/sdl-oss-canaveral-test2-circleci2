#!/bin/bash

function prepare {
  SERVICE_DIR=`pwd`
  PROTO_SRC=${SERVICE_DIR}/proto
  GRPC_OUT=${SERVICE_DIR}/generated/grpc
  echo "Preparing $SERVICE_DIR"
  mkdir ./build
  mkdir -p generated/cfssl
  mkdir -p "$GRPC_OUT"
  download-ui
  fetch_artifact cfssl-swagger.json 185
  swagger generate client -t generated/cfssl -f cfssl-swagger.json -A cfssl

  # Addition for auditlog service

  GRPC_PROTO=${SERVICE_DIR}/build/auditlog
  mkdir -p "$GRPC_PROTO"
  pushd ${GRPC_PROTO}
  fetch_artifact auditlog_api.proto 214

  PROTOC=protoc
  PROTO_SRC=${SERVICE_DIR}/build/auditlog
  GRPC_OUT=${SERVICE_DIR}/generated/auditlog
  mkdir -p "$GRPC_OUT"  
  ${PROTOC} -I=$PROTO_SRC --go_out=plugins=grpc:$GRPC_OUT "$PROTO_SRC/auditlog_api.proto"

  popd

  # PROTO_SRC and GRPC_OUT were changed above to build the auditlog proto file.
  # Changing them back

  PROTO_SRC=${SERVICE_DIR}/proto
  GRPC_OUT=${SERVICE_DIR}/generated/grpc

  protoc -I$PROTO_SRC --go_out=plugins=grpc:$GRPC_OUT "$PROTO_SRC/cloudmgmt.proto"
  echo "Generated protobuf stubs"
}

function build {
  go build  -ldflags '-w -s' -a -installsuffix cgo -o ./build/cloudmgmt ./cmd/main.go
  echo "Built go source files"
  swagger generate spec -w ./cmd -i ./swagger_in.json -o ./swagger_full.json
  echo "Generated swagger spec"
  swagger validate ./swagger_full.json
  echo "Validated swagger spec"
  if [[ $COVERAGE_BUILD -eq 1 ]]; then
    cd cmd && go test -ldflags '-w -s' -a -installsuffix cgo -c -o ../build/cloudmgmt_cov --coverpkg ../... && cd -
    echo "Built cloudmgmt coverage binary"
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
  # Copy generated swagger.json file
  cp ./swagger_full.json "$1/cloudmgmtapi_swagger.json"
  cp ./swagger.json "$1/xi_iot_api.json"
  cp ./swagger.json "$1/kps_api.json"
}

function download-ui {
  . ./ui_version.sh
  cp ./ui_version.sh ./build/ui-version.txt
  cp ./robots.txt ./build/robots.txt
  UI_SERVICE_NAME=sherlock-cloudmgmt-ui.tgz
  pushd ./build
  fetch_artifact ${UI_SERVICE_NAME} ${UI_VERSION}
  tar -zxvf ${UI_SERVICE_NAME}
  popd
  echo "Fetched ${UI_SERVICE_NAME}"
}
