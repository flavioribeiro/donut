#!/bin/bash
source ./scripts/mac_check_deps.sh

# deps
source ./scripts/setup_deps_flags.sh

go run -race main.go