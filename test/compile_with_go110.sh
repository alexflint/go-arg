#!/bin/bash

# This test checks that we can correctly "go get" and then use the go-arg package
# under go 1.10, which was the last release before introduction of the new go
# module system.

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

docker run \
    --rm \
    -v $DIR/some-program:/src \
    -w /src \
    golang:1.10 \
    bash -c "go get github.com/alexflint/go-arg && go build -o /dev/null"
