#!/usr/bin/env sh
set -eu

docker exec goex sh -lc 'go run /workspaces/defaultdevcontainer/cmd/goex/main.go'
