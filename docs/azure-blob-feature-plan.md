# Azure Blob Right Pane Feature Plan

## Scope and Constraints

- Left pane remains local filesystem.
- Right pane is Azure Blob-backed at startup.
- Runtime target is local Azurite emulator using a hardcoded connection string (temporary).
- Azure Blob flat namespace must be projected as virtual folders for navigation parity.
- Root Azure view must account for multiple containers.
- Right pane columns remain: `Name`, `Size`, `Date`, `Time`.
- No avoidable code duplication; prioritize maintainable abstractions for future backends.
- App must not crash if Azure backend is unavailable.

## Startup and Failure Behavior

- If Azure initialization or list fails, app still starts.
- Right pane enters an explicit error state (no rows shown for that pane).
- Footer/status shows concise error (for example: `Azure unavailable: connection refused`).
- User can retry with `r` (active pane retry action).
- On retry success, pane exits error state and loads rows normally.
- Left pane remains fully functional regardless of Azure state.

## UX and Navigation Model

### Right Pane Root

- Right pane starts in Azure root mode showing available containers.
- Containers are rendered as directory-like entries.
- Enter container with `l` or `Enter`.

### Inside Container

- View represents virtual path prefix within that container.
- Use delimiter `/` to derive virtual folders from blob names.
- Render virtual folders as directories (`<DIR>`).
- Render blobs directly under current prefix as files.
- `l`/`Enter` enters virtual folder.
- `h`/`Backspace` goes to parent virtual folder.
- At container root, `h` returns to container list.

### Hidden Handling for Azure

- Hidden means any path segment starts with `.`.
- If hidden toggle is off, hide entries where any segment is hidden.

## Architecture Design

### 1. Introduce Shared Entry Model

Create a storage-agnostic entry representation used by both panes before table rendering.

Proposed shape:
- `Name string`
- `FullPath string`
- `IsDir bool`
- `SizeBytes int64`
- `ModTime time.Time`
- `HasModTime bool`
- `Kind EntryKind` (`container`, `directory`, `blob`, optional for styling/logic)

### 2. Add Pane Backend Abstraction

Define backend interface to avoid branching logic in `Model`/`Pane`:

- `List(ctx context.Context, state Location, showHidden bool) ([]Entry, error)`
- `Enter(ctx context.Context, state Location, highlighted Entry) (Location, error)`
- `Parent(state Location) (Location, bool)`
- `DisplayPath(state Location) string`

`Location` should be a sealed sum type pattern:
- `type Location interface { isLocation() }`
- `type LocalLocation struct { Path string }`
- `type AzureLocation struct { Mode AzureMode; Container string; Prefix string }`

`AzureMode` values:
- `AzureModeContainers`
- `AzureModeObjects`

### 3. Backend Implementations

- `LocalBackend`: wraps current filesystem behavior.
- `AzureBlobBackend`: Azurite client, container listing, blob/prefix listing, virtual navigation.

### 4. Table Rendering Pipeline

Single conversion function:
- `[]Entry` -> `[]table.Row` with existing columns and formatting rules.

This keeps all sort/format behavior centralized.

Selection identity:
- Selection keys must use stable unique entry IDs (`FullPath`/logical key), not display name.

### 5. Client Construction

Add Azure package (latest stable at implementation time):
- `github.com/Azure/azure-sdk-for-go/sdk/storage/azblob`

Create Azure client provider with hardcoded Azurite connection string for now.

### 6. Async, Loading, and Commands

- Pane loads and navigation-triggered list operations should be async (`tea.Cmd`).
- Add per-pane loading state (`isLoading`, optional spinner model).
- Add result messages (`paneLoadSuccessMsg`, `paneLoadErrorMsg`) keyed by pane ID.
- While loading, keep prior rows visible if available and show loading indicator.

### 7. Pagination Strategy

Initial decision:
- Fetch all pages for current location before replacing rows.
- Keep this behavior behind internal helper boundaries so future incremental/streaming rendering can be added without changing pane logic.
- Add safety cap guardrail (configurable) to prevent runaway memory use on extremely large listings.

### 8. Error UX Strategy

- Show errors in status/footer with pane context.
- Preserve last successful rows (if any) on transient failures after initial load.
- On initial load failure: empty row set + explicit error state.
- Add retry key `r` for active pane.

### 9. Hidden and Sorting Ownership

- Hidden filtering happens in backend entry generation (not UI layer).
- Shared entry sorter applies to both local and Azure:
1. containers/directories first
2. blobs/files second
3. alphabetical by name within group

### 10. Explicitly Out of Scope (This Feature)

- Cross-pane copy/move/sync.
- Blob upload/download/edit.
- Container create/delete/rename.
- Real Azure authentication and account discovery.
- Background auto-refresh/watch mode.

## Implementation Plan with Review/Refactor Gates

## Step 0.5: Async Foundation

Tasks:
1. Introduce async pane loading command flow (`tea.Cmd` + success/error messages).
2. Add pane loading and error state models.
3. Add loading indicator in view for pane-level operations.
4. Add retry keybinding (`r`) to reload active pane.

Acceptance criteria:
1. Existing local behavior preserved with async flow.
2. Pane operations do not block UI thread.
3. Retry flow works for simulated load errors.

Review + refactor gate:
1. Review message type clarity and command ownership.
2. Refactor to avoid duplicating load command wiring across panes.

## Step 1: Extract Shared Entry + Row Mapping (No Behavioral Change)

Tasks:
1. Introduce `Entry` model and shared formatter.
2. Refactor local listing path to produce `[]Entry` then rows.
3. Keep current tests passing unchanged.
4. Add unit tests for entry->row mapping and selection key behavior.

Acceptance criteria:
1. Existing local behavior and tests unchanged.
2. No duplicated size/date/name row formatting logic.

Review + refactor gate:
1. Review coupling between pane and table model.
2. Refactor naming/package placement to keep boundaries clear.
3. Remove temporary adapters if redundant.

## Step 2: Introduce Backend Interface + Wire Panes

Tasks:
1. Add pane backend interface and location state.
2. Migrate pane navigation (`enter`, `parent`, `reload`) to backend methods.
3. Configure left pane with local backend.

Acceptance criteria:
1. Left pane behavior remains equivalent.
2. Navigation tests remain green.

Review + refactor gate:
1. Review `Pane` responsibilities (state vs behavior).
2. Refactor to keep backend-specific logic out of input handler.

## Step 3: Implement Azure Backend (Containers + Virtual Folders)

Tasks:
1. Add Azurite client bootstrap with hardcoded connection string.
2. Implement `List` for container root mode.
3. Implement `List` for container object mode using prefix+delimiter.
4. Implement `Enter` and `Parent` transitions across modes.
5. Implement hidden-segment filtering.
6. Pin/test against Azurite image `mcr.microsoft.com/azure-storage/azurite:3.33.0`.

Acceptance criteria:
1. Can browse containers then virtual folders/blobs.
2. `l`/`h` parity with local pane behavior.
3. Sorting parity (dirs/containers first, then files, alpha).

Review + refactor gate:
1. Review Azure-specific error handling and retries.
2. Refactor parsing/listing helpers for readability and testability.

## Step 4: Wire Right Pane to Azure Backend at Startup

Tasks:
1. Initialize model with left=local backend, right=azure backend.
2. Ensure footer displays meaningful Azure location (container/prefix/root).
3. Keep search/theme/header/selection behavior working in right pane.

Acceptance criteria:
1. App starts with right pane showing Azure containers.
2. Existing interaction keys work consistently on right pane.

Review + refactor gate:
1. Review startup error strategy (right pane unavailable, degraded mode/status).
2. Refactor initialization flow to keep constructor clean.

## Step 5: Add `cmd/seed-azurite` Data Seeder

Tasks:
1. Create `cmd/seed-azurite/main.go`.
2. Connect to Azurite via same hardcoded connection string.
3. Create multiple containers if missing.
4. Seed realistic blob keys including nested prefixes and hidden segments.
5. Provide idempotent behavior (safe re-run).

Suggested seed dataset:
1. Container `goex-dev`:
- `root-file.txt`
- `docs/readme.md`
- `docs/specs/v1.txt`
- `.hidden-root.txt`
- `configs/.secrets/app.env`
2. Container `media`:
- `images/logo.png`
- `images/icons/app.svg`
- `videos/demo.txt`

Acceptance criteria:
1. Seeder exits successfully when rerun.
2. App can browse seeded structures.

Review + refactor gate:
1. Review duplication between app Azure client setup and seeder setup.
2. Refactor shared connection/bootstrap into reusable internal package.

## Step 6: Testing Expansion

Tasks:
1. Unit tests for Azure path state transitions (`containers` <-> `objects`, parent rules).
2. Unit tests for virtual-folder extraction/filtering from blob names.
3. Unit tests for hidden-segment filtering.
4. Integration tests against Azurite for list/enter/parent flows.

Acceptance criteria:
1. `go test ./...` passes locally in devcontainer.
2. Azure-specific behavior covered with deterministic tests.

Review + refactor gate:
1. Review test fixture duplication.
2. Refactor common test helpers/builders.

## Step 7: CI Azurite Integration

Tasks:
1. Update `.github/workflows/ci.yml` to start Azurite service.
2. Ensure integration tests run in CI.
3. Keep lint/vet/test pipeline stable.

Acceptance criteria:
1. CI executes all tests including Azurite-dependent coverage.
2. Failures provide actionable logs.

Review + refactor gate:
1. Review CI job runtime and flakiness risk.
2. Refactor job steps for clarity (separate unit/integration phases if needed).

## Step 8: Docs and Final Polish

Tasks:
1. Update README with Azure right-pane behavior and controls.
2. Document Azurite requirements and seeding command.
3. Document architecture decisions for backend abstraction.

Acceptance criteria:
1. New contributor can run Azurite, seed data, and browse right pane successfully.
2. Docs match actual keybindings and behavior.

Review + refactor gate:
1. Review package boundaries for future backend extensions (S3, real Azure).
2. Refactor any remaining backend-specific leakage from shared UI layer.

## Delivery Checklist

1. [x] Right pane Azure-at-startup with container root listing.
2. [x] Virtual folder navigation (`l`/`h`) parity implemented.
3. [x] Hidden by any dot-segment filtering implemented.
4. [x] Shared entry/table pipeline with minimal duplication.
5. [x] `cmd/seed-azurite` implemented and idempotent.
6. [x] Integration tests run locally and in CI with Azurite.
7. [x] Docs updated.
8. [x] Refactor recommendations applied after each major step.
9. [x] App remains usable when Azure is unavailable (non-crashing degraded mode + retry).
