#!/bin/bash

PROJECT_ID="$(gcloud config get-value project)"

gcloud docker -- push gcr.io/${PROJECT_ID}/cloudmgmt-dev

