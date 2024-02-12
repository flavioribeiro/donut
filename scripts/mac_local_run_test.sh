#!/bin/bash
source ./scripts/mac_check_deps.sh

export CGO_LDFLAGS="-L$(brew --prefix srt)/lib -lsrt" 
export CGO_CFLAGS="-I$(brew --prefix srt)/include/"

# For debugging:
# go test -v -p 1 ./...
# ref https://github.com/golang/go/issues/46959#issuecomment-1407594935
go test ./...