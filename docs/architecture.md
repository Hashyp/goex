# Architecture

`defaultdevcontainer` is a Bubble Tea TUI app with a thin entrypoint and internal package boundary.

- `cmd/goex/main.go` wires and starts the Bubble Tea program.
- `internal/app` contains UI state, key handling, pane orchestration, and backend abstractions.
- `internal/azureblob` contains shared Azurite/Azure client bootstrap helpers reused by app and tooling.
- `cmd/seed-azurite` seeds Azurite with deterministic demo data.
- Tests live alongside source in `internal/app`.

## Pane Backends

Each pane is backed by a `PaneBackend` implementation:

- `LocalBackend`: local filesystem listing/navigation.
- `AzureBlobBackend`: container listing and virtual-folder navigation over blob prefixes.

The right pane starts with Azure containers and transitions into object-prefix navigation.

## Async Load Flow

Pane listing operations run via `tea.Cmd` and return pane-scoped load messages.

- non-blocking load on startup and navigation refresh
- per-pane loading/error state
- retry support (`r`) for active pane

This keeps the UI responsive for slower Azure operations.
