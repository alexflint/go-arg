#!/bin/bash

# This test checks that we can compile some code that depends on go-arg when using go 1.11 
# with the new go module system active.

docker run \
    --rm \
    -v $(pwd)/some-program:/src \
    -w /src \
    golang:1.11 \
    go build -o /dev/null
