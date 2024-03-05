#!/bin/bash
source ./scripts/mac_check_deps.sh

# deps
source ./scripts/setup_deps_flags.sh

# For debugging:
# go test -v -p 1 ./...
# ref https://github.com/golang/go/issues/46959#issuecomment-1407594935

go test ./...