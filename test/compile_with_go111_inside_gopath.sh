#!/bin/bash

# This test checks that we can correctly "go get" and then use the go-arg package using
# go 1.11 when the code is within the GOPATH (in which case modules are disabled by default).

docker run \
    --rm \
    -v $(pwd)/some-program:/go/src/some-program \
    -w /go/src/some-program \
    golang:1.11 \
    bash -c "go get github.com/alexflint/go-arg && go build -o /dev/null"
