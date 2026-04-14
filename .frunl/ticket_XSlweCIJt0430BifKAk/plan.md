## Implementation Plan
### Phase 1 – Architecture Assessment & Boundaries
- **Tasks**
  - Inspect `app.go`, `main.go`, and related tests to catalog handlers, data models, and helper functions that must be preserved.
  - Document desired target layout (e.g., `cmd/api`, `internal/config`, `internal/server`, `internal/auth`, `internal/ping`, `internal/middleware`) and capture dependencies each module will own.
  - Identify shared utilities (error writer, time provider, DB connection) that should move to dedicated packages to avoid circular imports.
  - Capture open questions about configuration, deployment, or upcoming features that might affect file/module boundaries and review with the team.
- **Validation**
  - Baseline `go test ./...` to ensure current behavior is green before refactorings start.
  - Share/confirm the proposed package diagram or summary comment in the ticket for sign-off.
- **Risks**
  - Hidden coupling in `app.go` may slow extraction or force redesign of handler signatures.
  - Missing agreement on boundaries could cause churn or partial rewrites later in the project.

### Phase 2 – Package Skeleton & Shared Types
- **Tasks**
  - Create the target directory structure and move `main.go` into `cmd/api`, leaving `package main` as the composition root only.
  - Add `internal/config` for env loading and `internal/data` (or `internal/store`) for `User`, `Ping`, and DB setup helpers; ensure migrations call sites are updated.
  - Introduce `internal/server/app.go` (or similar) to host the `App` struct, injecting dependencies via interfaces so handlers no longer reach into globals.
  - Extract reusable helpers such as `writeError`, `normalizeEmail`, and password validation into focused files with doc comments.
- **Validation**
  - `go build ./...` succeeds from repo root with the new package layout.
  - Lint/static analysis (e.g., `golangci-lint` or `go vet ./...`) remains clean for the refactored files.
- **Risks**
  - Import cycles could appear if shared utilities stay in the wrong package; be prepared to introduce an `internal/platform` layer.
  - Moving files without updating module paths or build scripts could break CI unexpectedly.

### Phase 3 – Handler & Service Extraction
- **Tasks**
  - Split HTTP handlers by domain (`internal/ping/handler.go`, `internal/time/handler.go`, `internal/auth/handler.go`) and wire them through a central router builder in `internal/server/router.go`.
  - Factor DB access behind small interfaces (`UserStore`, `PingStore`) implemented in `internal/data/sqlite` so handler packages only depend on interfaces.
  - Move JWT creation/validation and middleware into `internal/auth/token.go` and `internal/auth/middleware.go`, accepting a time provider for deterministic tests.
  - Update existing tests (e.g., `auth_test.go`) to target the new packages or add focused unit tests per handler/service.
- **Validation**
  - `go test ./...` including new package scopes, especially auth and ping routes.
  - Manual smoke test: run `cmd/api` server locally, hit `/ping`, `/pings`, `/auth/signup`, `/auth/login`, and `/auth/me` with curl to ensure responses are unchanged.
- **Risks**
  - Behavior/JSON regressions if handlers stop using the same error helpers or forget to set status codes.
  - Interface abstractions might add unnecessary complexity if not kept thin, slowing future feature work.

### Phase 4 – Hardening, Cleanup & Documentation
- **Tasks**
  - Remove the now-empty legacy `app.go` pieces and ensure each package has concise files rather than large monoliths.
  - Expand automated coverage: add tests for config loading edge cases, middleware failure paths, and repository behavior against in-memory SQLite.
  - Refresh `README.md` (and any docs under `docs/`) to describe the new layout and how to add routes without reintroducing monolith files.
  - Add CI checks (or update existing ones) to enforce formatting/linting so future contributions keep files scoped correctly.
- **Validation**
  - Final `go test ./...` and any integration test suites green; capture results in the ticket.
  - Optional: run benchmark or simple load test to ensure modularization didn’t noticeably impact latency.
- **Risks**
  - Residual unused code or duplicate helpers might remain if cleanup is rushed.
  - Tests depending on file locations (relative paths, fixtures) could fail if references aren’t updated.
