# Ticket Bootstrap

- Ticket: `TESTERAPI-AUTH-BUILD-20260412-3`
- Title: Production auth, registration, and password reset
- Objective: implement a production quality authentication, user registration and password reset system
- Bootstrap date: 2026-04-12

## Baseline Snapshot

- Branch: `ticket/testerapi-auth-build-20260412-3-production-auth-registration-and-password-reset-2`
- Current app type: Go API (Gin) with SQLite (GORM)
- Existing endpoints: `/ping`, `/pings`, `/hello`, `/time`
- Current auth features: none
- Current tests: none discovered (`go test ./...` executes but finds no test files)

## Setup Assumptions For Planning

1. Runtime and tooling
- Go `1.20` (from `go.mod`) is the baseline runtime for local and CI.
- SQLite remains the initial datastore during this ticket unless explicitly changed.

2. Application structure
- Current code is centered in `main.go`; refactoring into packages is expected before auth feature work.
- GORM auto-migration is currently used and may be expanded for user/auth models.

3. Security and auth scope
- "Production quality" implies secure password hashing, token/session invalidation strategy, and non-enumerating auth error behavior.
- Password reset must use single-use, expiring reset tokens and server-side validation.
- Registration must enforce uniqueness on identity fields (email/username if introduced).

4. Operational expectations
- Secrets (signing keys, token expiry settings, reset URL base, SMTP/provider config) should be injected via environment variables, not hardcoded.
- Logging should avoid leaking credentials, reset tokens, or sensitive user metadata.

5. Delivery and verification
- Feature work should include tests for auth, registration, reset flow, and invalid/abuse paths.
- A migration/backfill plan is needed if schema changes are introduced.

## Out Of Scope For Bootstrap Commit

- No auth implementation yet.
- No endpoint behavior changes yet.
- No schema changes yet.
