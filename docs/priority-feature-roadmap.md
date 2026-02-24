# Priority Feature Roadmap

This roadmap translates the requested feature order into implementation milestones with concrete tasks, file-level change targets, and a verification matrix.

Priority order (fixed):
1. Copy between panes
2. Move between panes
3. File preview/open
4. Filtering with regex support
5. Diff between folders/files
6. Rename file/folder/bucket/container
7. Help window with shortcuts
8. Persistent config
9. Rename search to find next
10. Filtering like in yazi with fd and grep but this will need to be adjusted for cloud storages
11. Applying shell script when renaming!
12. Batch operations may be BIG WIN overall for cloud specific operations:
  * apply tag(s) to bunch of blobs, generally change some metadata

## Current Baseline

- Core pane backend contract supports list/navigation/delete only.
- Backends: local filesystem, Azure Blob, S3, GCS.
- Multi-select exists and delete modal/progress exists.
- Regex search/highlight exists, but not row filtering.

## Delivery Plan

## Milestone R1: Transfer Foundations + Copy + Move + Help

Goal: deliver the most impactful file-manager actions first.

Detailed issue checklist: `docs/r1-issue-checklist.md`.

### Scope

- Cross-pane copy (single/multi-select, recursive folder/prefix, whole bucket/container).
- Cross-pane move.
- Help window listing all shortcuts.

### Task Breakdown

1. Introduce operations layer (`transfer` domain)
- Define backend-neutral operation API for copy/move and progress events.
- Add conflict policy (`skip`, `overwrite`, `rename` future-ready).
- Add per-item result aggregation (success/failure lists).

2. Add data streaming/copy primitives per backend pair
- Local <-> Local
- Local <-> Azure/S3/GCS
- Azure/S3/GCS <-> Azure/S3/GCS
- Recursive traversal for directories/prefixes and bucket/container roots.

3. UI integration for copy/move
- New keybindings for copy and move actions.
- Reuse modal patterns (confirmation + progress + final summary).
- Destination is the opposite pane current location.

4. Help window
- Add `?` modal with grouped shortcuts by context.
- Generate help content from one centralized keymap/descriptor source.

### File-Level Change Targets

- `internal/app/backend.go`
  - Keep pane-navigation contract focused; add separate operations interfaces (avoid bloating pane list API).
- `internal/app/model.go`
  - New operation states/messages for copy/move lifecycle.
- `internal/app/view.go`
  - Transfer progress modal + help modal rendering.
- `internal/app/input_base.go`
  - Key handlers for copy/move/help actions.
- `internal/app/input_picker.go`, `internal/app/input_delete.go`
  - Shared modal behavior consistency.
- `internal/app/footer.go`
  - Surface operation status/result summaries.
- `internal/app/backend_local.go`
  - Reader/writer/list recursion helpers for local paths.
- `internal/app/backend_azure.go`
- `internal/app/backend_s3.go`
- `internal/app/backend_gcs.go`
  - Object stream read/write and recursive enumeration helpers.
- `internal/app/` (new files expected)
  - `operation_transfer.go`
  - `operation_copy.go`
  - `operation_move.go`
  - `operation_types.go`
  - `help.go`

### R1 Test Plan

- Unit tests:
  - conflict policy behavior
  - recursive expansion behavior (dirs/prefixes/bucket roots)
  - move workflow (`copy -> verify -> delete`)
  - help modal visibility and content sections
- Integration tests:
  - Local->S3 copy, S3->Local copy
  - Local->GCS copy, GCS->Local copy
  - Local->Azure copy, Azure->Local copy
  - cloud->cloud copy for at least one pair (S3->GCS)
  - move partial-failure reporting
- Regression tests:
  - existing navigation/search/delete unchanged

## Milestone R2: Preview/Open + Regex Filtering + Rename

Goal: improve inspection and in-place organization workflows.

### Scope

- File/object preview and external open.
- Regex filtering mode (actual row filtering, not highlight-only search).
- Rename for file/folder/object and bucket/container.

### Task Breakdown

1. Preview/Open
- Add preview modal with scroll support.
- Add max preview size and binary detection.
- Add external open command hook.

2. Regex filtering
- Add dedicated filter input mode and filtered row set.
- Keep existing search mode intact; define interaction rules between search/filter.
- Add visible indicators for active filter and match count.

3. Rename
- File/object rename: backend-native when available, fallback copy+delete.
- Folder/prefix rename: recursive copy+delete.
- Bucket/container rename: create target + recursive copy + source delete (explicit warning).

### File-Level Change Targets

- `internal/app/model.go`
  - preview/filter/rename states and messages.
- `internal/app/view.go`
  - preview modal, filter modal, rename modal.
- `internal/app/input_search.go`
  - split responsibilities between search and filter.
- `internal/app/search.go`
  - keep highlight logic; add filtered row pipeline in complementary file.
- `internal/app/pane.go`
  - filtered vs full entries handling.
- `internal/app/backend_local.go`
- `internal/app/backend_azure.go`
- `internal/app/backend_s3.go`
- `internal/app/backend_gcs.go`
  - rename/preview stream support.
- `internal/app/` (new files expected)
  - `input_filter.go`
  - `operation_preview.go`
  - `operation_rename.go`
  - `filter.go`

### R2 Test Plan

- Unit tests:
  - filter regex compile errors and empty-filter behavior
  - preview binary/size guardrails
  - rename path/key generation and safety checks
- Integration tests:
  - preview for local and at least one cloud backend
  - rename object/prefix on S3, GCS, Azure
  - rename file/dir on local backend
- Regression tests:
  - search `n/N` navigation unchanged
  - delete with selection unchanged after filter/rename additions

## Milestone R3: Diff + Persistent Config

Goal: complete advanced comparison workflows and persist user preferences/state.

### Scope

- File diff and folder diff summaries.
- Persistent config for UX and behavior defaults.

### Task Breakdown

1. Diff
- File diff view (inline initially).
- Folder diff summary by relative path with categories:
  - only-left
  - only-right
  - changed
- Comparison strategy:
  - local: size + mtime + optional checksum
  - cloud: size + updated/etag where available, checksum optional

2. Persistent config
- Add config schema and load/save path.
- Persist:
  - theme
  - hidden visibility default
  - default backends for left/right pane
  - external opener command
  - transfer conflict default
  - last visited locations per pane/backend
- Add schema versioning and migration scaffold.

### File-Level Change Targets

- `cmd/goex/main.go`
  - load config before model init, save on quit/events.
- `internal/app/model.go`
  - consume config defaults and last-known pane state.
- `internal/app/view.go`
  - diff modal rendering.
- `internal/app/` (new files expected)
  - `diff.go`
  - `input_diff.go`
  - `config.go`
  - `config_store.go`
  - `config_migrate.go`
- `configs/`
  - add example config template and field documentation.
- `README.md`
  - user-facing config and diff instructions.

### R3 Test Plan

- Unit tests:
  - config parse/validation/migration
  - diff engine path matching and classification
- Integration tests:
  - config round-trip save/load
  - diff between local folders and selected cloud pair
- Regression tests:
  - startup with missing/invalid config falls back safely

## Cross-Milestone Engineering Rules

1. Keep pane navigation API separate from transfer/diff APIs to avoid backend interface sprawl.
2. Reuse modal patterns (state machine + progress tick + result summary) from delete flow.
3. Preserve responsiveness via async commands for all network/file-heavy operations.
4. Always keep existing behavior backward compatible unless explicitly replaced.

## Backend Capability Matrix (Target End State)

| Capability | Local | Azure Blob | S3 | GCS |
|---|---|---|---|---|
| Copy in/out | Yes | Yes | Yes | Yes |
| Move in/out | Yes | Yes | Yes | Yes |
| Preview text | Yes | Yes | Yes | Yes |
| Regex filter | Yes | Yes | Yes | Yes |
| Diff file/folder | Yes | Yes | Yes | Yes |
| Rename file/object | Yes | Yes* | Yes* | Yes* |
| Rename folder/prefix | Yes | Yes* | Yes* | Yes* |
| Rename bucket/container | N/A | Yes* | Yes* | Yes* |
| Persistent settings | Yes | Yes | Yes | Yes |

`*` Implemented via copy+delete when native rename does not exist.

## Suggested Keybinding Allocation (Proposed)

- `c`: copy selection/highlight to opposite pane
- `m`: move selection/highlight to opposite pane
- `o`: open/preview highlighted file
- `f`: open filter modal
- `=`: diff selected/highlighted pair
- `R`: rename selected/highlighted entry
- `?`: help modal

## Acceptance Gate per Milestone

- All `go test ./...` pass in devcontainer.
- New feature tests added for each capability.
- README/docs updated for any new keybindings and behavior.
- No regressions in existing pane navigation/search/delete.
