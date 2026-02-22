# GCS Backend Feature Plan

## Scope and Constraints

- Both panes can be switched to any supported backend via picker (`p`); this feature adds GCS as an additional selectable backend.
- Existing default right pane backend remains S3; GCS is added as a selectable pane backend via picker (`p`).
- Runtime target is local fake-gcs-server emulator in devcontainer.
- Emulator endpoint resolution order is fixed:
1. `GOEX_GCS_EMULATOR_HOST`
2. `STORAGE_EMULATOR_HOST`
3. default `http://127.0.0.1:4443`
- GCS flat namespace must be projected as virtual folders for navigation parity.
- Root GCS view must account for multiple buckets.
- Pane columns remain: `Name`, `Size`, `Date`, `Time`.
- `/` search behavior must remain backend-agnostic and consistent across local, Azure, S3, and GCS panes.
- No avoidable code duplication; prioritize maintainable abstractions aligned with existing local/Azure/S3 backends.
- App must not crash if GCS backend is unavailable.

## Startup and Failure Behavior

- If GCS backend initialization or list fails, app still starts.
- Pane using GCS enters explicit error state (no rows shown for that pane).
- Footer/status shows concise error in the form `GCS unavailable: <reason>`.
- User can retry with `r` (active pane retry action).
- On retry success, pane exits error state and loads rows normally.
- Other panes remain fully functional regardless of GCS state.

## UX and Navigation Model

### GCS Root

- GCS pane starts in root mode showing available buckets.
- Buckets are rendered as directory-like entries.
- Enter bucket with `l` or `Enter`.

### Inside Bucket

- View represents virtual path prefix within that bucket.
- Use delimiter `/` to derive virtual folders from object names.
- Render virtual folders as directories (`<DIR>`).
- Render objects directly under current prefix as files.
- `l`/`Enter` enters virtual folder.
- `h`/`Backspace` goes to parent virtual folder.
- At bucket root, `h` returns to bucket list.

### Hidden Handling for GCS

- Hidden means any path segment starts with `.`.
- If hidden toggle is off, hide entries where any segment is hidden.

### Search Behavior (`/`)

- Search remains a pane-local UI concern applied to rendered row names, not backend-specific list logic.
- Regex query behavior, match highlighting, and `n`/`N` navigation must behave identically for GCS rows and existing providers.
- Switching a pane backend via picker (`p`) must preserve existing search semantics for that pane.

## Architecture Design

### 1. Add GCS Location Type

Extend location model with GCS state:

- `type GCSMode string`
- `GCSModeBuckets`
- `GCSModeObjects`
- `type GCSLocation struct { Mode GCSMode; Bucket string; Prefix string }`

### 2. Add GCS Backend Implementation

Implement `GCSBackend` under `internal/app` following `PaneBackend`:

- `InitialLocation() Location` returns bucket-root mode.
- `List(ctx, state, showHidden)` lists buckets or objects+virtual folders.
- `Enter(ctx, state, highlighted)` transitions bucket->objects or prefix->child prefix.
- `Parent(state)` transitions child prefix->parent or bucket root->bucket list.
- `DisplayPath(state)` follows existing S3-style convention: `gcs:///`, `gcs:///bucket`, `gcs:///bucket/<prefix>` (with `<prefix>` already slash-terminated when non-empty).
- `ParentHighlightName(state)` supports stable parent selection behavior.

### 2.5. Extend Entry Kind Semantics for GCS Buckets

- Add a dedicated `EntryKind` named `KindGCSBucket`.
- Update `Entry.IsDirLike()` to treat GCS buckets as directory-like.
- Update `Entry.TypeOrSize()` so `KindGCSBucket` renders as `<BKT>`.
- Ensure sort/grouping behavior remains consistent with existing bucket/container/directory-first ordering.

### 3. GCS Client Construction

Add shared bootstrap package for GCS client setup:

- New package: `internal/gcsblob`.
- Mirror existing storage package structure:
1. `internal/gcsblob/client.go` (config parsing + client creation)
2. `internal/gcsblob/bootstrap.go` (idempotent helpers like ensure-bucket)
- Define explicit config contract in `internal/gcsblob/client.go`:
1. `Endpoint`: resolve with `GOEX_GCS_EMULATOR_HOST` -> `STORAGE_EMULATOR_HOST` -> `http://127.0.0.1:4443`
2. `RequestTimeout`: `GOEX_GCS_REQUEST_TIMEOUT`, default `30s`
- Export `DefaultConfig()` and `NewClient(ctx, cfg)` (same pattern as `internal/s3blob`).
- Export `EnsureBucket(ctx, client, name)` in `internal/gcsblob/bootstrap.go`.
- Build emulator-aware client (`cloud.google.com/go/storage`) without requiring cloud credentials for emulator paths.
- Keep client/config error messages user-readable for status/footer rendering.

### 4. Backend Factory + Picker Integration

- Add `paneBackendGCS` choice in backend factory.
- Add picker label `GCS`.
- Wire factory creation path to instantiate `GCSBackend` or static error backend when bootstrap fails.
- Keep same-backend-on-both-panes behavior intact.
- Update `paneBackendChoiceFromPane` to include:
1. direct `GCSBackend` type mapping
2. `StaticErrorBackend` + `GCSLocation` mapping (same pattern currently used for Azure/S3)
- Set static-error display path for failed GCS bootstrap to `gcs:///`.

### 5. Pagination and Safety Strategy

- Fetch all pages for current location before replacing rows (same current behavior).
- Add GCS entry cap guardrail `maxGCSEntries` to prevent runaway memory usage.
- Keep list logic encapsulated so incremental/streaming rendering can be added later.

### 6. Error UX Strategy

- Show errors in status/footer with pane context.
- Preserve last successful rows (if any) on transient failures after initial load.
- On initial load failure: empty row set + explicit error state.
- Reuse existing retry key `r`.

### 7. Hidden and Sorting Ownership

- Hidden filtering happens in backend entry generation (not view layer).
- Shared entry sorter remains:
1. buckets/directories first
2. objects/files second
3. alphabetical by name within group

### 8. Explicitly Out of Scope (This Feature)

- Cross-pane copy/move/sync.
- Object upload/download/edit.
- Bucket create/delete/rename.
- Real GCP authentication and project discovery.
- Background auto-refresh/watch mode.

## Implementation Plan with Review/Refactor Gates

## Step 1: Add Shared GCS Bootstrap Package

Tasks:
1. Create `internal/gcsblob` for emulator config parsing and client creation.
2. Define default endpoint behavior aligned with devcontainer (`http://127.0.0.1:4443`).
3. Mirror existing `client.go` + `bootstrap.go` split used by other storage packages.
4. Add focused unit tests for endpoint precedence and timeout parsing.
5. Export `Config`, `DefaultConfig()`, `NewClient(ctx, cfg)`, and `EnsureBucket(ctx, client, name)`.

Acceptance criteria:
1. Bootstrap returns a working client when emulator host is reachable.
2. Misconfiguration produces actionable errors.
3. `LoadTimeout` for GCS backend is sourced from GCS config (not hardcoded in backend).

Review + refactor gate:
1. Review consistency with `internal/azureblob` and `internal/s3blob` patterns.
2. Refactor option parsing for shared conventions if duplication is found.

## Step 2: Implement `GCSBackend` in `internal/app`

Tasks:
1. Add `GCSMode`/`GCSLocation` types in location model.
2. Add `backend_gcs.go` implementing `PaneBackend`.
3. Extend `EntryKind` and entry rendering semantics for GCS bucket rows.
4. Implement bucket listing and object listing with virtual folder derivation.
5. Implement hidden-segment filtering and max-entry guardrail.
6. Implement display path and parent highlight semantics using existing S3-style convention:
1. `gcs:///` for bucket root
2. `gcs:///bucket` for selected bucket root
3. `gcs:///bucket/<prefix>` where `<prefix>` is stored slash-terminated for non-empty prefixes
7. Set backend entry cap constant to `maxGCSEntries = 20000` for parity with Azure/S3 guardrails.
8. Keep backend implementation search-neutral: no GCS-specific filtering that conflicts with existing `/` regex behavior.

Acceptance criteria:
1. Can browse buckets, virtual folders, and objects with `l`/`h` parity.
2. Sorting and hidden behavior match Azure/S3 conventions.
3. GCS path formatting matches the same style rules already used by S3 path display.
4. `GCSBackend.LoadTimeout()` returns configured timeout from `gcsblob.DefaultConfig()`.
5. Running `/` search in a GCS pane produces the same highlight and next/prev-match behavior as in local/Azure/S3 panes.

Review + refactor gate:
1. Review list/transform helper readability and testability.
2. Refactor shared cloud helper code if GCS introduces repeated logic.

## Step 3: Wire Factory and Backend Picker

Tasks:
1. Add `paneBackendGCS` choice to `backend_factory`.
2. Add picker label and index handling updates.
3. Ensure `paneBackendChoiceFromPane` supports `GCSBackend` and matching static error location type (`GCSLocation` case in `StaticErrorBackend` branch).
4. Keep model initialization unchanged (S3 still default right pane unless product decision changes).
5. Ensure backend-factory GCS failure path uses `NewStaticErrorBackendWithLocation(err, GCSLocation{Mode: GCSModeBuckets}, \"gcs:///\")`.

Acceptance criteria:
1. Picker shows `GCS` and switching active pane works.
2. Failed GCS bootstrap yields static error backend with retry path intact.

Review + refactor gate:
1. Review factory branching complexity.
2. Refactor backend registration if switch statement growth becomes error-prone.

## Step 4: Add `cmd/seed-gcs` Data Seeder

Tasks:
1. Create `cmd/seed-gcs/main.go`.
2. Connect via same GCS emulator bootstrap path.
3. Create multiple buckets if missing.
4. Seed realistic object keys including nested prefixes and hidden segments.
5. Make command idempotent and safe to rerun.
6. Match existing large-volume seeding pattern used by `seed-minio` and `seed-azurite` (bulk folders/files + progress logs), not only minimal sample data.
7. Use the same bulk constants for parity:
1. `bulkFoldersPerBucket = 30`
2. `bulkFilesPerFolder = 22`
3. `bulkRootFiles = 12`
4. `progressLogEvery = 250`

Suggested seed dataset:
1. Bucket `goex-dev`:
- `root-file.txt`
- `docs/readme.md`
- `docs/specs/v1.txt`
- `.hidden-root.txt`
- `configs/.secrets/app.env`
2. Bucket `media`:
- `images/logo.png`
- `images/icons/app.svg`
- `videos/demo.txt`
3. Additional baseline buckets for scale/perf parity:
- `finance`
- `logs`
- `reports`
- `archive`
- `datasets`

Acceptance criteria:
1. Seeder exits successfully when rerun.
2. App can browse seeded structures through GCS backend.
3. Seeder object volume is comparable to existing MinIO/Azurite seeders for realistic navigation and pagination testing.

Review + refactor gate:
1. Review duplication between app GCS client setup and seeder setup.
2. Refactor shared bootstrap/helper surfaces as needed.

## Step 5: Testing Expansion

Tasks:
1. Unit tests for GCS path state transitions (`buckets` <-> `objects`, parent rules).
2. Unit tests for virtual-folder extraction/filtering from object names.
3. Unit tests for hidden-segment filtering.
4. Integration tests against fake-gcs-server for list/enter/parent flows.
5. Keep emulator tests opt-in via `GOEX_RUN_GCS_TESTS=1` and `STORAGE_EMULATOR_HOST`.
6. Add deterministic integration helpers in `internal/app/gcs_integration_test.go`:
1. `putGCSObject(...)`
2. `cleanupGCSBucket(...)`
7. Keep test naming/style aligned with `s3_integration_test.go` and `azure_integration_test.go`.
8. Add model-level regression tests confirming `/` search, regex errors, highlight rendering, and `n`/`N` navigation still work after switching active pane to GCS.

Acceptance criteria:
1. `go test ./...` passes locally in devcontainer.
2. GCS-specific behavior has deterministic unit and integration coverage.
3. Search behavior remains consistent across all providers, including GCS-backed panes.

Review + refactor gate:
1. Review fixture duplication across Azure/S3/GCS tests.
2. Refactor common cloud-backend test helpers/builders.

## Step 6: CI fake-gcs-server Integration

Tasks:
1. Update `.github/workflows/ci.yml` to start fake-gcs-server service.
2. Add health-check/wait step for emulator readiness against `http://127.0.0.1:4443/storage/v1/b`.
3. Run tests with `GOEX_RUN_GCS_TESTS=1` and emulator host env.
4. Keep lint/vet/test pipeline stable.
5. Pin fake-gcs-server runtime to an explicit version tag in CI (no floating `latest`).

Acceptance criteria:
1. CI executes GCS integration tests reliably.
2. Failures provide actionable logs.
3. CI exports `STORAGE_EMULATOR_HOST=http://127.0.0.1:4443` for GCS test steps.

Review + refactor gate:
1. Review CI runtime and flakiness risk.
2. Refactor CI steps into clear unit/integration phases if needed.

## Step 7: Docs and Final Polish

Tasks:
1. Update README with GCS backend picker behavior and controls.
2. Document GCS seeding command and expected sample dataset.
3. Update architecture docs for new backend and location type.

Acceptance criteria:
1. New contributor can start emulator, seed data, and browse GCS successfully.
2. Docs match actual keybindings and behavior.

Review + refactor gate:
1. Review package boundaries for future backend extensions.
2. Refactor backend-specific leakage from shared UI layer.

## Delivery Checklist

1. [ ] GCS backend selectable from pane backend picker.
2. [ ] GCS root bucket listing and virtual folder navigation implemented.
3. [ ] Hidden by dot-segment filtering implemented for GCS.
4. [ ] GCS bootstrap package implemented with emulator env support.
5. [ ] `cmd/seed-gcs` implemented and idempotent.
6. [ ] Integration tests run locally and in CI with fake-gcs-server.
7. [ ] Docs updated.
8. [ ] Refactor recommendations applied after each major step.
9. [ ] App remains usable when GCS is unavailable (non-crashing degraded mode + retry).
10. [ ] `/` search and `n`/`N` navigation behave consistently for local, Azure, S3, and GCS panes.
