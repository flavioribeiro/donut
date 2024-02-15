#!/bin/bash
source ./scripts/mac_check_deps.sh

export CGO_LDFLAGS="-L$(brew --prefix srt)/lib -lsrt" 
export CGO_CFLAGS="-I$(brew --prefix srt)/include/"

go run -race main.go