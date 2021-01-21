#!/bin/bash

# A simple script to generate swagger.json
# in the current directory & validate it.

/usr/bin/swagger generate spec -w ./cmd -i ./swagger_in.json -o /output/swagger.json \
&& /usr/bin/swagger validate /output/swagger.json
