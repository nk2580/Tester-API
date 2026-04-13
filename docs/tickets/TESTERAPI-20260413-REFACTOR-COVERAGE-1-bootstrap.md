# TESTERAPI-20260413-REFACTOR-COVERAGE-1 Bootstrap

## Ticket
- ID: `TESTERAPI-20260413-REFACTOR-COVERAGE-1`
- Title: `Refactor Codebase + Raise Test Coverage`
- Objective: refactor the codebase and raise test coverage to an extremely high level.

## Baseline Snapshot (2026-04-13)
- Branch: `ticket/testerapi-20260413-refactor-coverage-1-refactor-codebase-raise-test-coverage-2`.
- Working tree status at bootstrap start: clean.
- Application stack: Go 1.25.7, Gin, GORM, SQLite.
- Main runtime entrypoint: `main.go` initializes auth config, opens `db/data.db`, and serves on `:8080`.
- Domain models currently in scope: `Ping` and `User`.
- Route surface currently spans ping CRUD-lite, utility endpoints, and auth endpoints (`/auth/signup`, `/auth/login`, `/auth/me`).
- Existing test suite file: `auth_test.go`.
- Baseline statement coverage: `60.3%`.

## Setup Assumptions
- Local execution environment can run `go` commands and download modules.
- In this execution environment, default Go cache paths are not writable; use repo-local caches for deterministic runs:
  - `GOMODCACHE=$(pwd)/.tmp/gomodcache`
  - `GOCACHE=$(pwd)/.tmp/gocache`
- Auth configuration (`JWT_SECRET`, optional `JWT_TTL`) remains required for app startup behavior.
- SQLite file-based persistence (`db/data.db`) remains the local baseline datastore during refactor.

## Open Questions To Resolve During Planning
- Define what "extremely high" coverage means as a numeric target (for example: >=95% lines/statements for package scope).
- Decide refactor boundaries: structural cleanup only vs behavior changes and feature extraction.
- Choose test strategy mix: handler-level integration tests, unit tests for pure helpers, and any DB seam abstractions.
- Determine whether coverage gates should be enforced in CI and at what threshold.
- Clarify whether non-functional changes (linting/style/packaging) are in or out of ticket scope.

## Baseline Validation
- `GOMODCACHE=$(pwd)/.tmp/gomodcache GOCACHE=$(pwd)/.tmp/gocache go test ./... -coverprofile=.tmp/coverage.out` passes.
- `go tool cover -func=.tmp/coverage.out` reports total statement coverage at `60.3%`.
- Repository state was clean before and after bootstrap edits, excluding transient local cache artifacts removed after validation.

## Bootstrap Scope Guard
- This commit intentionally avoids implementation work and refactor changes.
- Only ticket bootstrap context is added to support planning and scope control.
