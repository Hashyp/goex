# Architecture

`defaultdevcontainer` is a Bubble Tea TUI app with a thin entrypoint and internal package boundary.

- `cmd/goex/main.go` wires and starts the Bubble Tea program.
- `internal/app` contains all state and behavior (`Model`, panes, rendering, input handling).
- Tests live alongside source in `internal/app`.

This structure keeps app internals private while preserving a clean executable boundary.
