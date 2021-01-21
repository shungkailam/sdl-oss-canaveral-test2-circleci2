#!/bin/bash

# Test runner for Canaveral

# golang specific variables
export GOPATH=~/.go_workspace
export GOROOT=/tmp/go
export PATH=$GOROOT/bin:$PATH
export SRC=$GOPATH/src
export SHERLOCK_SRC=$SRC

pushd $SHERLOCK_SRC/cloudservices

if [[ ! -f "./cloudmgmt/build/cloudmgmt" ]]; then
  echo "Not building cloudmgmt, skip tests"
  popd
  exit 0
fi

env

GO111MODULE=on go test -timeout 900s -parallel 16 -cpu 16 -cover ./cloudmgmt/...

if [ $? -eq 0 ]
then
  echo "All cloudmgmt unit tests done successfully"
  #stop docker container started by AI service
  docker stop $(docker ps -q --filter ancestor=ai:build)
  popd
  exit 0
else
  echo "Some cloudmgmt unit tests failed"
  #stop docker container started by AI service
  #Comment this ,if you want to debug failure scenarios.
  docker stop $(docker ps -q --filter ancestor=ai:build)
  popd
  exit 1
fi
