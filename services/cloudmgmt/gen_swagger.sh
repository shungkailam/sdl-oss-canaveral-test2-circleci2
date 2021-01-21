#!/bin/bash

# simple script to build cloudmgmt-swagger-gen docker image,
# then run the image as a container to generate swagger.json
# in the current directory

docker build -f SwaggerDockerfile -t cloudmgmt-swagger-gen ..
docker run --rm -it -v "$(pwd)":/output cloudmgmt-swagger-gen
