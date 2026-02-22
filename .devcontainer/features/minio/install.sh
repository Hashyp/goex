#!/usr/bin/env bash
set -euo pipefail

if command -v minio >/dev/null 2>&1; then
  exit 0
fi

tmp_bin=$(mktemp)
curl -fsSL https://dl.min.io/server/minio/release/linux-amd64/minio -o "$tmp_bin"
install -m 0755 "$tmp_bin" /usr/local/bin/minio
rm -f "$tmp_bin"
