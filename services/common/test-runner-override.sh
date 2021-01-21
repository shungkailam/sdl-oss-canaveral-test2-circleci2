#!/bin/bash

# Test runner for Canaveral

# golang specific variables
export GOPATH=~/.go_workspace
export GOROOT=/tmp/go
export PATH=$GOROOT/bin:$PATH
export SRC=$GOPATH/src
export SHERLOCK_SRC=$SRC

pushd $SHERLOCK_SRC/cloudservices

if [[ ! -d "./cloudmgmt/build" ]]; then
  echo "Not building cloudmgmt, skip tests"
  popd
  exit 0
fi

GO111MODULE=on go test -cover ./common/...

if [ $? -eq 0 ]
then
  echo "All common unit tests done successfully"
  popd
  exit 0
else
  echo "Some common unit tests failed"
  popd
  exit 1
fi

