# goex

`goex` is a terminal-based dual-pane file browser built with Bubble Tea.

## Repository Layout

- `cmd/defaultdevcontainer`: executable entrypoint
- `internal/app`: application logic and tests
- `docs`: architecture and repository documentation
- `scripts`: repeatable local/devcontainer command wrappers
- `configs`: non-secret configuration templates and examples
- `.github/workflows`: CI definitions

## Run

```bash
go run ./cmd/defaultdevcontainer
```

## Test

```bash
go test ./...
```
