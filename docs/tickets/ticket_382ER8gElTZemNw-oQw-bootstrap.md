# ticket_382ER8gElTZemNw-oQw Bootstrap

## Ticket
- ID: `ticket_382ER8gElTZemNw-oQw`
- Title: `convert this API to use gRPC`
- Objective: convert this API to use gRPC.

## Baseline Snapshot (2026-04-14)
- Branch: `ticket/nk2580-tester-api-3`.
- Working tree status at bootstrap start: clean.
- Go toolchain per `go.mod`: module `github.com/nk2580/Tester-API`, Go 1.21 (local toolchain `go version go1.25.7 darwin/arm64`).
- Current API stack: HTTP server via `gin-gonic/gin`, persistence via GORM + SQLite file `db/data.db`, JWT auth using `github.com/golang-jwt/jwt/v5` with bcrypt password hashing, env-driven config loader (`JWT_SECRET`, optional `JWT_TTL`).
- Active HTTP routes exposed from `App.Router()` today:
  - Public: `POST /ping`, `GET /pings`, `GET /hello`, `GET /time`, `POST /auth/signup`, `POST /auth/login`.
  - Protected (JWT bearer via `/auth` group): `GET /auth/me`.
- Domain models auto-migrated on startup: `Ping` (message log) and `User` (email/password auth records).
- SQLite file is shared between app runs and tests, so prior data persists unless cleaned manually.

## Setup Assumptions
- Go toolchain capable of building modules in `go.mod` is installed; Go >=1.21 recommended.
- Local execution expects `JWT_SECRET` env var; planning will need to determine secrets handling for any gRPC services.
- SQLite remains acceptable as the storage backend during planning; switching databases is out of bootstrap scope.
- Converting to gRPC still needs parity with existing HTTP capabilities unless planning explicitly deprecates them.
- TLS, service discovery, and client upgrade paths will be clarified during planning; none exist yet.

## Open Questions To Resolve During Planning
- What is the desired gRPC service surface (e.g., Ping service, Auth service) and how does it map to existing routes?
- Should the HTTP/gin server be retired, coexist, or proxy to the new gRPC handlers during migration?
- How will authentication/JWTs be represented in gRPC metadata (and do we need session/token changes)?
- Do we need protobuf definitions for current response contracts or can they be redesigned?
- Will the persisted SQLite database remain or should a new persistence strategy be introduced with the gRPC shift?

## Baseline Validation
- `go test ./...` currently fails: `TestAuthMiddlewareMissingInvalidExpiredAndValidToken` returns HTTP 401 instead of the expected 200 (likely due to persistent SQLite data causing duplicate users / token mismatches). This will need cleanup before planning-driven work is validated.
- No other automated validation run during bootstrap.

## Bootstrap Scope Guard
- Only bootstrap documentation has been added; no implementation work has begun.
- Planning will build on this baseline snapshot and captured assumptions.
