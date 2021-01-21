#!/bin/bash

## Build cloudmgmt (dev build) for running in GCP
## Once built, use push_gcp.sh to push the docker images to GCP
## then use undeploy_sherlock_cloud_gcp.sh and deploy_sherlock_cloud_gcp.sh
## from sherlock_cloud_deployer-x.y.z to undeploy / deploy to GCP

PROJECT_ID="$(gcloud config get-value project)"

cp -f ../../../package/docker/Dockerfile ../../Dockerfile
docker build -t gcr.io/${PROJECT_ID}/cloudmgmt-dev ../..
rm -f ../../Dockerfile

