#!/bin/bash

SWAGGER=file:///Users/heiko.koehler/sherlock-cloudmgmt/services/common/kubeval/kubernetes-json-schema/swagger-1.15.4-restricted.json

./openapi2jsonschema -o v1.15.4-standalone-strict-restricted-full --kubernetes --strict --stand-alone $SWAGGER

./openapi2jsonschema -o v1.15.4-standalone-strict-restricted-full --kubernetes --expanded --strict --stand-alone $SWAGGER

