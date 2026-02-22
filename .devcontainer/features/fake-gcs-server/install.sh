#!/usr/bin/env bash
set -euo pipefail

if command -v fake-gcs-server >/dev/null 2>&1; then
  exit 0
fi

tmp_dir=$(mktemp -d)
GOBIN="$tmp_dir" go install github.com/fsouza/fake-gcs-server@latest
install -m 0755 "$tmp_dir/fake-gcs-server" /usr/local/bin/fake-gcs-server
rm -rf "$tmp_dir"
