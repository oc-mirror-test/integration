# AGENTS.md

## What this project is

Integration test suite for [oc-mirror v2](https://github.com/openshift/oc-mirror), an OpenShift tool that mirrors OpenShift Container Platform releases, Operators, helm charts, and other images between registries. Tests run real mirror operations using the oc-mirror CLI against a local distribution/distribution registry.

## Structure

- `pkg/ocmirror/runner.go` — executes oc-mirror commands, captures output
- `pkg/registry/registry.go` — manages a local registry lifecycle
- `tests/integration/` — Ginkgo v2 test suite
  - `integration_suite_test.go` — suite setup, globals, per-test registry lifecycle
  - `helpers_test.go` — assertion helpers (`expect*` functions) and utilities

## Basic Repo Guidelines

- This repo contains efficient and robust tests with well-defined boundaries.
- Ginkgo v2 with Gomega. Package is always `integration_test`.
- Each `It` block gets a fresh, empty registry — don't assume state across tests.
- Use `setupWorkDir()`/`cleanupWorkDir()` in `BeforeEach`/`AfterEach` — never inline.
- Check `helpers_test.go` for existing assertions before writing new ones.
- Check existing tests for duplicate coverage before adding new ones.
- Implement meaningful assertions, do not stay at surface level.
- When implementing new assertions, use the ImageSetConfig/DeleteImageSetConfig as a source of truth instead of passing individual expected values.

## Running tests

```bash
# The `registry` binary (distribution/distribution) must be in PATH.
# Ask the user for its location if unknown.
PATH=$PATH:<path-to-registry-bin> go test -v ./tests/integration/ --ginkgo.focus "test name"
```

Env vars: `OC_MIRROR_BINARY`, `ARTIFACTS_DIR`, `TEST_TIMEOUT` (default 30m).
