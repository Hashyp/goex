#!/usr/bin/env bash
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y --no-install-recommends ripgrep
apt-get clean
rm -rf /var/lib/apt/lists/*
