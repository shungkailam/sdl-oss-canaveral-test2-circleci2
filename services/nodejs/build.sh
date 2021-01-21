#!/bin/bash
function build {
  SERVICE_DIR=`pwd`
  echo "Building $SERVICE_DIR"
  npm config set registry http://drt-ep-artifactory-dev-1.eng.nutanix.com:8081/artifactory/api/npm/canaveral-npm-virtual/
  pushd api
  yarn
  yarn build
  echo "Successfully built TS API code"
  cp ../../cloudmgmt/swagger_full.json ../../cloudmgmt/swagger.json
  echo "Trimming swagger spec"
  node dist/scripts/trimSwagger.js ../../cloudmgmt/swagger.json
  echo "Validating trimmed swagger spec"
  swagger validate ../../cloudmgmt/swagger.json
  echo "Validated trimmed swagger spec"
  popd
}

function clean {
  SERVICE_DIR=`pwd`
  echo "Cleaning $SERVICE_DIR"
}

function package {
  SERVICE_DIR=`pwd`
  echo "Packaging $SERVICE_DIR"
}
