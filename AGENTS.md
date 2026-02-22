# Agent Development Guide - goex

You are a Go expert designing and implementing high-quality, clean code solutions that favor SOLID principles.

## Project Overview

goex is a terminal-based dual-pane file browser built with Bubble Tea that allows navigation and file operations between local file systems and Azure Blob Storage.

## Repository Layout

```text
.
|- cmd/goex/main.go   # application entrypoint
|- internal/app/                     # core TUI application logic
|  |- model.go
|  |- pane.go
|  |- view.go
|  |- ...
|- docs/                             # architecture and contributor docs
|- scripts/                          # repeatable local/devcontainer helpers
|- configs/                          # non-secret config templates/examples
|- .github/workflows/                # CI pipelines
|- .devcontainer/devcontainer.json
|- go.mod
|- go.sum
|- README.md
|- AGENTS.md
```

Keep the root directory reserved for project metadata and top-level config only.

Placement rules for new files:

- Go executable entrypoints: `cmd/<binary-name>/main.go`
- Application business/UI logic: `internal/app`
- Tests for app logic: next to source in `internal/app`
- Design/architecture/contributor docs: `docs`
- Repeatable helper scripts: `scripts`
- Config templates and examples (non-secrets): `configs`
- GitHub automation: `.github/workflows`

## Technology Stack

- Go 1.25
- `github.com/charmbracelet/bubbles/table`
- `github.com/charmbracelet/bubbletea`
- `github.com/charmbracelet/lipgloss`

## CRITICAL: Container Requirements

Most commands must run inside the devcontainer. The working directory inside the container is `/workspaces/defaultdevcontainer`.

Use `devpod-cli` for all DevPod workspace lifecycle commands (`up`, `ssh`, `stop`, `delete`, etc.). Do not use the `devpod` command.

Use:

```bash
docker exec goex sh -lc 'cd /workspaces/defaultdevcontainer && <your-command>'
```

Examples:

```bash
docker exec goex sh -lc 'cd /workspaces/defaultdevcontainer && go run ./cmd/goex'
docker exec goex sh -lc 'cd /workspaces/defaultdevcontainer && go test ./...'
./scripts/run.sh
./scripts/test.sh
```

`git` and `gh` commands can run outside the container.

## CRITICAL: Mandatory Recreate After Devcontainer Config Changes

If `.devcontainer/devcontainer.json` is changed in any way, you must recreate the DevPod workspace before running any further development/testing commands.

Use these exact identifiers:

- Workspace name: `defaultdevcontainer`
- Container name: `goex`
- In-container project path: `/workspaces/defaultdevcontainer`

Run this exact sequence from the host:

```bash
devpod-cli up defaultdevcontainer --recreate --ide none --open-ide=false
docker exec goex sh -lc 'command -v minio >/dev/null && echo minio:installed; command -v fake-gcs-server >/dev/null && echo fake-gcs-server:installed; command -v azurite >/dev/null && echo azurite:installed'
docker exec goex sh -lc 'ss -ltn | grep -E ":(10000|9000|4443)"'
docker exec goex sh -lc 'curl -fsS http://127.0.0.1:9000/minio/health/live >/dev/null && echo minio:running; curl -fsS http://127.0.0.1:4443/storage/v1/b >/dev/null && echo fake-gcs-server:running'
docker exec goex sh -lc 'cd /workspaces/defaultdevcontainer && go test ./...'
```

Do not skip this recreate step after devcontainer config changes.

## CRITICAL: Branching and Clean Working Tree

Always start a new feature branch before implementation work.

Before starting:

- Ensure the working tree is clean.
- Switch to the default branch (`master` in this repository).
- Create a new feature branch from `master`.

## Definition of Done

A feature is done only after tests pass.

Always run the full test suite after implementation or fixes:

```bash
docker exec goex sh -lc 'cd /workspaces/defaultdevcontainer && go test ./...'
```

Provide a concise commit message summary when asked.
