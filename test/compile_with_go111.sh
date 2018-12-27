#!/bin/bash

docker run \
    --rm \
    -v $(pwd)/some-program:/src \
    -w /src \
    golang:1.11 \
    go build
