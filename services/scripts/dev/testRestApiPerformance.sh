#!/bin/bash

#
# Simple script to test cloudmgmt REST API performance
# Currently simply test performance for get all objects
# for each object type.
# Note: mock data contains about 60 datasource objects
# with a total size of around 3MB, which is causing huge
# performance issue for DynamoDB (ntnxsherlock.com)
# sherlockntnx.com is running GCP + MySQL pod,
# {prod, dev}.ntnxsherlock.com each runs AWS RDS Aurora MySQL
#

USAGE="Usage: $0 <iterations>"

if [ "$#" -ne 1 ]; then
  echo $USAGE
  exit 1
fi

doGetCall() {
  ENDPOINT=$1
  TENANT_ID=$2
  API_PATH=$3
  ITER=$4
  ts=$(gdate +%s%N)
  counter=1
  while [ $counter -le $ITER ]
  do
    curl -k -X GET --header 'Accept: application/json' --cookie "X-NTNX-SHERLOCK-TENANT-ID=$TENANT_ID" "$ENDPOINT/v1/$API_PATH/" >/dev/null 2>&1
    ((counter++))
  done
  tt=$((($(gdate +%s%N) - $ts)/1000000))
  echo "$ENDPOINT: get: $API_PATH, iter: $ITER, time: ${tt}ms"
}

entities=( edges sensors datastreams scripts categories users projects cloudcreds datasources )
endpoints=( https://dev.ntnxsherlock.com https://sherlockntnx.com https://prod.ntnxsherlock.com https://ntnxsherlock.com )
iter=$1

for endpoint in "${endpoints[@]}"
do
  for entity in "${entities[@]}"
  do
     doGetCall $endpoint tenant-id-waldot $entity $iter
  done
done
