# TESTERAPI-20260413-REFACTOR-COVERAGE-1 Implementation Plan

## Goal
- Refactor for maintainability without changing API behavior.
- Raise coverage to an enforceable “extremely high” bar.

## Coverage Targets
- Total statement coverage: `>=95%`.
- Critical auth/config/error paths: `>=98%`.
- Merge gate fails below `95%`.

## Phase 0: Baseline + Mapping
### Concrete Tasks
- Capture baseline coverage using repo-local caches.
- Map route-to-handler-to-helper flow and identify low-coverage seams.
- Inventory untested branches (`main`, duplicate-error path, auth edge cases).

### Validation Steps
- Run `GOMODCACHE=$(pwd)/.tmp/gomodcache GOCACHE=$(pwd)/.tmp/gocache go test ./... -coverprofile=.tmp/coverage.out`.
- Run `go tool cover -func=.tmp/coverage.out`.
- Save baseline artifact under `.tmp/`.

### Risks
- Baseline drift during refactor.
- SQLite behavior coupling making tests brittle.

## Phase 1: Behavior-Preserving Refactor
### Concrete Tasks
- Split routing composition from business logic where needed.
- Extract small focused helpers to improve unit-test surface.
- Keep error payload format centralized via `writeError`.
- Avoid broad abstractions unless they directly reduce test complexity.

### Validation Steps
- Run `go test ./...` after each refactor slice.
- Spot-check endpoint status codes and response shapes remain unchanged.

### Risks
- Accidental auth behavior drift.
- Over-refactor adding complexity.

## Phase 2: Coverage Expansion
### Concrete Tasks
- Add table-driven tests for config/env parsing and helper validation.
- Add exhaustive auth middleware/token claim branch tests.
- Add handler branch tests for `/ping`, `/pings`, `/time`, `/auth/*`.
- Add startup-path coverage strategy for `main` via seam/smoke approach.
- Keep deterministic time and isolated in-memory DB per test.

### Validation Steps
- Run targeted tests, then full suite with coverage profile.
- Verify `go tool cover -func=.tmp/coverage.out` meets `>=95%` total and `>=98%` critical-path goals.

### Risks
- High branch coverage pursuit may produce low-value tests.
- DB failure branch simulation may need controlled seams.

## Phase 3: Coverage Gate + Workflow
### Concrete Tasks
- Add coverage-check command/script (and CI hook if in scope).
- Parse total coverage and hard-fail below threshold.
- Document quick test run vs full coverage run in project docs.

### Validation Steps
- Verify normal pass behavior.
- Verify failure behavior by temporarily raising threshold.

### Risks
- Over-strict gates blocking urgent fixes.
- Environment differences causing minor percentage variance.

## Phase 4: Stabilization + Exit
### Concrete Tasks
- Run full regression and final coverage gate on clean state.
- Remove dead abstractions and finalize refactor notes.
- Confirm objective achieved with measurable evidence.

### Validation Steps
- Final `go test ./... -coverprofile=.tmp/coverage.out`.
- Final `go tool cover -func=.tmp/coverage.out`.
- Confirm no endpoint contract regressions.

### Risks
- Late edits dropping coverage.
- Docs and commands drifting from actual workflow.
