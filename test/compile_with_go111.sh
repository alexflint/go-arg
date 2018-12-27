#!/bin/bash

# This test checks that we can compile some code that depends on go-arg when using go 1.11 
# with the new go module system active.

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

docker run \
    --rm \
    -v $DIR/some-program:/src \
    -w /src \
    golang:1.11 \
    go build -o /dev/null
