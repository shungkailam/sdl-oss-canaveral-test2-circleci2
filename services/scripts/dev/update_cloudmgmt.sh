#!/bin/bash

#
# Simple script to update an existing cloudmgmt deployment to the latest.
# Warning: run this script will delete all data in your DB and reload mock data
#
# Pre-condition: your kubectl should be set to the context (cluster + namespace)
# you want to operate in
#

pod=`kubectl get pods | grep cloudmgmt-deployment | awk '{print $1}'`
kubectl delete pod $pod

# wait till cloudmgmt pod is running
echo -n "Wait till cloudmgmt pod is running ..."
while true; do
    state=`kubectl get pods | grep cloudmgmt-deployment | awk '{printf "%s", $3}'`
    if [ "$state" == "Running" ]; then
    break
    fi
    echo -n "."
    sleep 1
done

# init DB
pod=`kubectl get pods | grep cloudmgmt-deployment | awk '{print $1}'`
kubectl exec -ti $pod -- node dist/rest/db-scripts/deleteDBCommon.js

sleep 1

kubectl exec -ti $pod -- node dist/rest/db-scripts/initDBCommon.js

echo "All Done!"

