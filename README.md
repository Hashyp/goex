# goex

`goex` is a terminal-based dual-pane file browser built with Bubble Tea.

- Left pane: local filesystem.
- Right pane: Azure Blob storage (Azurite by default).

## Repository Layout

- `cmd/goex`: executable entrypoint
- `cmd/seed-azurite`: seed Azurite with demo containers/blobs
- `internal/app`: application logic and tests
- `internal/azureblob`: shared Azure client/bootstrap helpers
- `docs`: architecture and repository documentation
- `scripts`: repeatable local/devcontainer command wrappers
- `configs`: non-secret configuration templates and examples
- `.github/workflows`: CI definitions

## Run

```bash
go run ./cmd/goex
```

## Test

```bash
go test ./...
```

## Azure Right Pane Behavior

- Starts at Azure container list (`azure:/`).
- `Enter` / `l`: enter container or virtual folder.
- `Backspace` / `h`: parent folder; from container root goes back to container list.
- `.` toggles hidden entries for active pane.
- Hidden Azure entries are those where any path segment starts with `.`.
- `r` retries loading active pane (useful after connectivity errors).

If Azure is unavailable, app still runs:
- left pane stays functional,
- right pane shows an error state and can be retried with `r`.

## Azurite Setup and Seed

Start Azurite blob endpoint:

```bash
docker run --rm -p 10000:10000 mcr.microsoft.com/azure-storage/azurite:3.33.0 azurite-blob --blobHost 0.0.0.0 --skipApiVersionCheck
```

Seed demo data:

```bash
go run ./cmd/seed-azurite
```

## Integration Tests (Azurite)

Azurite integration tests are opt-in locally:

```bash
GOEX_RUN_AZURITE_TESTS=1 go test ./...
```
