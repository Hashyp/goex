#!/usr/bin/env sh
set -eu

docker exec goex sh -lc 'cd /workspaces/defaultdevcontainer && go test ./...'
