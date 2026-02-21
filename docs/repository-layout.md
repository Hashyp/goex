# Repository Layout

This repository follows a standard Go application structure:

- `cmd/goex`: main package and process startup.
- `internal/app`: non-exported app logic, UI state, and tests.
- `docs`: architecture notes and contributor-facing documentation.
- `scripts`: helper scripts for run/test workflows.
- `configs`: non-secret config examples and templates.
- `.github/workflows`: GitHub Actions for validation and automation.

Root-level files should stay limited to repository metadata and top-level configuration (`README.md`, `AGENTS.md`, `go.mod`, `go.sum`, `.gitignore`).
