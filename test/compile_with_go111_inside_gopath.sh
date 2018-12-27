#!/bin/bash

# Under go 1.11, modules are disabled by default when user code is located within the GOPATH.
# In this test, we check that we can correctly "go get" the go-arg package, and then compile
# some code that uses it.

docker run \
    --rm \
    -v $(pwd)/some-program:/go/src/some-program \
    -w /go/src/some-program \
    golang:1.11 \
    bash -c "go get github.com/alexflint/go-arg && go build"
