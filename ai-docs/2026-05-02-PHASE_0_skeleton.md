# Phase 0 — Skeleton and Infrastructure

- **Date**: 2026-05-02
- **Status**: In Progress

## Goal

Create a buildable, testable Go CLI skeleton for `tang`: module setup, Cobra root command, version command, basic output renderer, Makefile, lint config, CI, README, license, and ignore rules.

## Constraints

- Module path starts as `tangled.org/onev.cat/tang`.
- Pin `tangled.org/core` to `37303f21368b8e8567e3b96205a72481190ab232`.
- Keep implementation minimal and reviewable; no feature behavior beyond skeleton.
- Avoid hardcoded PDS defaults.

## Plan

1. Initialize `go.mod` and fetch the pinned Tangled core dependency.
2. Add the planned directory structure and package placeholders.
3. Implement `cmd/tang/main.go`, `internal/cli/root.go`, `completion`, and `version`.
4. Add the initial output renderer and tests for JSON field filtering.
5. Add Makefile, lint config, CI workflow, README, LICENSE, and `.gitignore`.
6. Run automated validation: `go test ./...`, `make build`, `./bin/tang version`, `./bin/tang --help`, lint when available, and cross-build smoke.

## Validation Log

- Completed: `go get tangled.org/core@37303f21368b8e8567e3b96205a72481190ab232` succeeded through the module version `v1.13.0-alpha.0.20260502074102-37303f21368b`.
- Completed: `make test` passed for all packages.
- Completed: `make build && ./bin/tang version` passed and printed the injected version/commit.
- Completed: `./bin/tang --help` lists `version` and `completion`.
- Completed: `GOOS=linux GOARCH=amd64 go build ./cmd/tang` passed.
- Completed: installed `golangci-lint` v2 locally and `make lint` passed with 0 issues.

## Completion

Phase 0 is complete. No human-only validation remains for this phase.
