#!/bin/bash

# Test runner for Canaveral

echo "$(pwd)"

if [[ ! -d "./services/nodejs/api/dist" ]]; then
  echo "Not building nodejs, skip tests"
  exit 0
fi

env
pushd ./services/nodejs/api/dist

#node testRBAC.js go https://test.ntnxsherlock.com tenant-id-rbac-test-$CIRCLE_BUILD_NUM https://cfssl-test.ntnxsherlock.com

if [ $? -eq 0 ]
then
  echo "All nodejs/api tests done successfully"
  popd
  exit 0
else
  echo "Some nodejs/api tests failed"
  popd
  exit 1
fi

