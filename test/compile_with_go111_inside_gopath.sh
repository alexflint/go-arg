#!/bin/bash

# This test checks that we can correctly "go get" and then use the go-arg package using
# go 1.11 when the code is within the GOPATH (in which case modules are disabled by default).

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

docker run \
    --rm \
    -v $DIR/some-program:/go/src/some-program \
    -w /go/src/some-program \
    golang:1.11 \
    bash -c "go get github.com/alexflint/go-arg && go build -o /dev/null"
