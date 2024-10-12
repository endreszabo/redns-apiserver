#!/bin/bash

set -o errexit

docker run --rm -it -v $PWD/proto:/proto -v $PWD/proto:/genproto y7hu/golang-protoc-alpine:1.22

docker buildx build . --network=host -t y7hu/redns_strayserver:0.1
