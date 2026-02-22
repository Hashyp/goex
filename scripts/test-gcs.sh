#!/usr/bin/env sh
set -eu

docker exec goex sh -lc 'cd /workspaces/defaultdevcontainer && STORAGE_EMULATOR_HOST=${STORAGE_EMULATOR_HOST:-http://127.0.0.1:4443} GOEX_RUN_GCS_TESTS=1 go test ./...'
