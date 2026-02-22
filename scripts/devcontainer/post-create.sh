#!/usr/bin/env bash
set -euo pipefail

alias_line='alias goex="go run /workspaces/defaultdevcontainer/cmd/goex/main.go"'

mkdir -p "$HOME"
touch "$HOME/.bash_aliases"
touch "$HOME/.bashrc"

grep -Fqx "$alias_line" "$HOME/.bash_aliases" || echo "$alias_line" >> "$HOME/.bash_aliases"
grep -Fqx 'source ~/.bash_aliases' "$HOME/.bashrc" || echo 'source ~/.bash_aliases' >> "$HOME/.bashrc"
