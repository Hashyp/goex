# GCS Emulator in Devcontainer

This repository uses `fsouza/fake-gcs-server` for local Google Cloud Storage API emulation.

## Runtime Defaults

Inside the devcontainer:

- emulator endpoint: `http://127.0.0.1:4443`
- `STORAGE_EMULATOR_HOST=http://127.0.0.1:4443`
- `GOEX_GCS_EMULATOR_HOST=http://127.0.0.1:4443`

The process is started by `.devcontainer/devcontainer.json` in `postStartCommand`.
Logs are written to `/tmp/fake-gcs/fake-gcs.log`.

## Verify the Emulator

Run from host:

```bash
docker exec goex sh -lc 'curl -fsS http://127.0.0.1:4443/storage/v1/b'
```

An empty bucket list response confirms the emulator is reachable.

## Use in Go Tests

For Go clients that support the Cloud Storage emulator, set:

```bash
STORAGE_EMULATOR_HOST=http://127.0.0.1:4443
```

Helper script:

```bash
./scripts/test-gcs.sh
```

This exports `GOEX_RUN_GCS_TESTS=1` and `STORAGE_EMULATOR_HOST` for the test command.

## Tradeoffs

- Pros: fast local feedback loop, no cloud billing, no credentials needed for emulator paths.
- Cons: not an official Google Cloud Storage emulator, so API edge cases can differ from real GCS.
- Recommendation: keep fast emulator tests in CI, and run a smaller real-GCS test suite before release.
