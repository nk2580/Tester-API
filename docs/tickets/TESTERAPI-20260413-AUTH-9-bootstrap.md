# TESTERAPI-20260413-AUTH-9 Bootstrap

## Ticket
- ID: `TESTERAPI-20260413-AUTH-9`
- Title: `Auth + Signup`
- Objective: Add an authentication system and user signup system to this API.

## Baseline State (Pre-Planning)
- Branch: `ticket/testerapi-20260413-auth-9-auth-signup-2`
- Working tree status: clean (`git status --short` at bootstrap time)
- Last baseline validation: `go test ./...` passed on 2026-04-13 (no test files yet)
- Current API footprint: `main.go` with `GET /hello`, `GET /time`, `POST /ping`, `GET /pings`
- Persistence: SQLite file at `db/data.db` via GORM

## Setup Assumptions
- Runtime uses Go `1.20` as declared in `go.mod`.
- Server starts with `go run main.go` and listens on `:8080`.
- Existing API consumers (if any) expect current routes to remain functional unless explicitly changed in planning.
- Auth design decisions are not yet defined in code and must be resolved during planning:
  - Session vs token-based auth (recommended to decide JWT vs server sessions early).
  - Password hashing strategy (recommended: `bcrypt` or `argon2id`).
  - User schema fields and uniqueness constraints (at minimum: email/username uniqueness).
  - Access control scope for existing routes (`/ping`, `/pings`) and new signup/login endpoints.

## Out of Scope For Bootstrap
- No functional auth or signup code changes.
- No API behavior changes.
- No schema migration for user/auth tables.
