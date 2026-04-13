# TESTERAPI-20260413-AUTH-10 Bootstrap

## Ticket
- ID: `TESTERAPI-20260413-AUTH-10`
- Title: `Auth + Signup`
- Objective: add authentication and user signup to the API.

## Baseline Snapshot (2026-04-13)
- Branch: `ticket/testerapi-20260413-auth-10-auth-signup-2`
- Working tree status at bootstrap start: clean.
- Existing API stack: Go (`gin-gonic/gin`), GORM, SQLite (`db/data.db`).
- Current routes in `main.go`: `POST /ping`, `GET /pings`, `GET /hello`, `GET /time`.
- Current persistence model: single `Ping` table auto-migrated at startup.

## Setup Assumptions
- Go toolchain available and compatible with modules in `go.mod`.
- SQLite file-based database remains acceptable for local development.
- No auth-related environment variables are currently configured.
- No existing user/account domain model exists yet.
- Backward compatibility for current non-auth routes should be preserved unless planning explicitly changes them.

## Open Questions To Resolve During Planning
- Authentication mode: session, JWT, or another token approach.
- Password hashing algorithm and policy requirements.
- Signup constraints: unique fields, verification requirements, and minimum profile fields.
- Route protection scope: which endpoints become authenticated vs public.
- Error/response contract for auth failures and signup validation.

## Baseline Validation
- `go test ./...` passes (no test files currently present).
- Repository state verified clean before bootstrap edits.

## Bootstrap Scope Guard
- This commit intentionally avoids implementation work.
- Only ticket bootstrap context is added to support planning.
