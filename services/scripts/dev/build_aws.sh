#!/bin/bash

## Build cloudmgmt (dev build) for running in AWS
## Once built, use push_aws.sh to push the docker images to AWS
## Need to make sure that AWS credentials are set up and you have access to push to the ECR repo

## This is needed to store the AWS login credentials which need to be retrived every 12 hrs
$(aws ecr get-login --no-include-email --region us-west-2)
cp -f ../../../package/docker/Dockerfile ../../Dockerfile
docker build -t 770301640873.dkr.ecr.us-west-2.amazonaws.com/cloudmgmt-dev ../..
rm -f ../../Dockerfile

