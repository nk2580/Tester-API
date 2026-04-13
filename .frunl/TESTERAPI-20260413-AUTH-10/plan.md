# TESTERAPI-20260413-AUTH-10 Implementation Plan

## Objective
- Add auth + signup to the Gin/GORM API using secure password hashing, JWT access tokens, and protected routes.

## Phase 1: Design and Contracts
### Concrete Tasks
- Define endpoints: `POST /auth/signup`, `POST /auth/login`, `GET /auth/me`.
- Define request/response payloads and consistent error schema.
- Finalize auth standards: `bcrypt`, JWT claims (`sub`, `iat`, `exp`), token TTL.
- Define config/env requirements: `JWT_SECRET`, `JWT_TTL`.

### Validation Steps
- Contract review for required fields and status codes.
- Confirm env var behavior for local/dev/prod.

### Risks
- Ambiguous contracts causing client/backend drift.
- Token semantics changing later and forcing rework.

## Phase 2: Data Model and Persistence
### Concrete Tasks
- Add `User` model with unique `Email`, `PasswordHash`, timestamps.
- Add `User` to startup migration flow.
- Add persistence helpers for lookup by email and user creation.
- Normalize unique-constraint failures to stable API errors.

### Validation Steps
- Verify migration creates user schema.
- Verify duplicate email signup maps to expected conflict behavior.

### Risks
- SQLite uniqueness errors varying by driver details.
- Model logic crowding `main.go` and reducing maintainability.

## Phase 3: Auth + Signup Implementation
### Concrete Tasks
- Implement signup validation, email normalization, password hashing.
- Implement login validation, password check, JWT issuance.
- Return token response fields: `access_token`, `token_type`, `expires_in`.
- Ensure sensitive fields are never returned.

### Validation Steps
- Test signup success and duplicate-email failure.
- Test login success and invalid-credential failure.
- Verify response payloads never contain password data.

### Risks
- Weak password policy may fail security expectations.
- JWT signing/claim errors may break downstream auth.

## Phase 4: Middleware and Protected Access
### Concrete Tasks
- Add bearer-token middleware for parse, verify, expiry checks.
- Attach authenticated identity to request context.
- Add protected `GET /auth/me`.
- Apply middleware to protected group while keeping public routes open.

### Validation Steps
- Verify `401` for missing token.
- Verify `401` for malformed/invalid/expired token.
- Verify `200` for valid token on protected endpoint.
- Confirm `/hello` and `/time` still work without auth.

### Risks
- Header parsing edge cases causing incorrect 500s.
- Time/expiry handling causing intermittent auth failures.

## Phase 5: Tests, Hardening, Documentation
### Concrete Tasks
- Add tests for signup/login success and failure paths.
- Add middleware tests for missing/invalid/expired/valid tokens.
- Document abuse-mitigation baseline (rate-limit note) in scope notes.
- Update `README.md` with endpoint usage and env vars.

### Validation Steps
- Run `go test ./...` with all tests passing.
- Run end-to-end smoke flow: signup -> login -> protected call.
- Check errors and logs for secret/sensitive leakage.

### Risks
- Missing branch coverage hiding auth regressions.
- Missing ops guidance causing insecure deployment defaults.

## Cross-Cutting Decisions
- JWT-only vs access+refresh token model.
- Password policy baseline.
- Signup returns token vs requires separate login.
- Route namespace now (`/auth/*`) vs versioned (`/v1/auth/*`).

## Deliverables
- Signup and login endpoints implemented.
- JWT middleware and at least one protected endpoint.
- Automated auth-path test coverage.
- Updated auth/config docs.
