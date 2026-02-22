# goex

`goex` is a terminal-based dual-pane file browser built with Bubble Tea.

- Left pane: local filesystem.
- Right pane: S3-compatible storage (MinIO by default).

## Repository Layout

- `cmd/goex`: executable entrypoint
- `cmd/seed-azurite`: seed Azurite with demo containers/blobs
- `cmd/seed-gcs`: seed fake-gcs-server with demo buckets/objects
- `cmd/seed-minio`: seed MinIO with demo buckets/objects
- `internal/app`: application logic and tests
- `internal/azureblob`: shared Azure client/bootstrap helpers
- `internal/s3blob`: shared S3 client/bootstrap helpers
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

## Devcontainer Storage Emulators

The devcontainer starts these local storage endpoints automatically:

- Azurite: `http://127.0.0.1:10000`
- MinIO (S3 API): `http://127.0.0.1:9000`
- fake-gcs-server (GCS emulator): `http://127.0.0.1:4443`

GCS emulator env defaults inside the devcontainer:

- `STORAGE_EMULATOR_HOST=http://127.0.0.1:4443`
- `GOEX_GCS_EMULATOR_HOST=http://127.0.0.1:4443`

See `docs/gcs-emulator.md` for usage details.

## S3 Right Pane Behavior

- Starts at S3 bucket list (`s3:///`).
- `Enter` / `l`: enter bucket or virtual folder.
- `Backspace` / `h`: parent folder; from bucket root goes back to bucket list.
- `.` toggles hidden entries for active pane.
- Hidden S3 entries are those where any key segment starts with `.`.
- `r` retries loading active pane (useful after connectivity errors).
- `p` opens backend picker modal for the active pane (file system / azure / s3 / gcs).
  - `Enter` applies selected backend to that pane.
  - `Esc` closes modal without changes.
  - Both panes can use the same backend at the same time.

If S3 is unavailable, app still runs:
- left pane stays functional,
- right pane shows an error state and can be retried with `r`.

## MinIO Setup and Seed

Start MinIO:

```bash
docker run --rm -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /data --console-address :9001
```

Seed demo data:

```bash
go run ./cmd/seed-minio
```

S3 client runtime defaults (override with env vars):

- `GOEX_S3_ENDPOINT` (default `http://127.0.0.1:9000`)
- `GOEX_S3_REGION` (default `us-east-1`)
- `GOEX_S3_ACCESS_KEY` (default `minioadmin`)
- `GOEX_S3_SECRET_KEY` (default `minioadmin`)
- `GOEX_S3_PATH_STYLE` (default `true`)
- `GOEX_S3_REQUEST_TIMEOUT` (default `30s`)

## Integration Tests (MinIO)

MinIO integration tests are opt-in locally:

```bash
GOEX_RUN_MINIO_TESTS=1 go test ./...
```

## Integration Tests (GCS Emulator)

Run GCS integration tests against fake-gcs-server with:

```bash
./scripts/test-gcs.sh
```

## Seed GCS Emulator

Seed fake-gcs-server with demo buckets/objects:

```bash
go run ./cmd/seed-gcs
```
