#!/usr/bin/env bash
set -euo pipefail

mkdir -p /tmp/azurite /tmp/minio /tmp/minio-data /tmp/fake-gcs /tmp/fake-gcs/data

if ! ss -ltn | grep -q ':10000'; then
  nohup azurite \
    --location /tmp/azurite \
    --blobHost 0.0.0.0 \
    --queueHost 0.0.0.0 \
    --tableHost 0.0.0.0 \
    --skipApiVersionCheck \
    >/tmp/azurite/azurite.log 2>&1 < /dev/null &
fi

if ! ss -ltn | grep -q ':9000'; then
  nohup minio server /tmp/minio-data --address :9000 --console-address :9001 \
    >/tmp/minio/minio.log 2>&1 < /dev/null &
fi

if ! ss -ltn | grep -q ':4443'; then
  nohup fake-gcs-server \
    -scheme http \
    -host 0.0.0.0 \
    -port 4443 \
    -backend filesystem \
    -filesystem-root /tmp/fake-gcs/data \
    -external-url http://127.0.0.1:4443 \
    >/tmp/fake-gcs/fake-gcs.log 2>&1 < /dev/null &
fi
