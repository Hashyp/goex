#!/usr/bin/env sh
set -eu

docker exec default-go-devcontainer sh -lc 'cd /workspaces/defaultdevcontainer && go run ./cmd/defaultdevcontainer'
