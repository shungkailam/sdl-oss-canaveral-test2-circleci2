#!/bin/bash

# Test runner for Canaveral

# golang specific variables
export GOPATH=~/.go_workspace
export GOROOT=/tmp/go
export PATH=$GOROOT/bin:$PATH
export SRC=$GOPATH/src
export SHERLOCK_SRC=$SRC

pushd $SHERLOCK_SRC/cloudservices

if [[ ! -d "./operator/build" ]]; then
  echo "Not building operator, skip tests"
  popd
  exit 0
fi

GO111MODULE=on go test -cover ./operator/...

if [ $? -eq 0 ]
then
  echo "All operator unit tests done successfully"
  popd
  exit 0
else
  echo "Some operator unit tests failed"
  popd
  exit 1
fi

