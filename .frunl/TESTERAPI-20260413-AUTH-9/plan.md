# TESTERAPI-20260413-AUTH-9 Implementation Plan

## Objective
- Add authentication and signup to the existing Go/Gin API with safe defaults and test coverage.

## Proposed Auth Approach
- Use JWT bearer access tokens (stateless validation).
- Use `bcrypt` for password hashing.
- Use normalized email as unique login identifier.
- Keep `/hello` and `/time` public.
- Protect `/ping` and `/pings` (unless product decision changes this in Phase 0).

## Phase 0: Contract + Decisions
- Tasks:
- Confirm token TTL and auth error contract.
- Confirm signup/login payloads and validation rules.
- Confirm protected-route scope.
- Validation:
- API contract doc with sample request/response bodies.
- Explicit decision log for unresolved tradeoffs.
- Risks:
- Unclear access policy causes rework.
- Late auth-model decision blocks implementation.

## Phase 1: Data Model + Persistence
- Tasks:
- Add `User` model (`ID`, `Email`, `PasswordHash`, `CreatedAt`, `UpdatedAt`).
- Add unique index/constraint on normalized email.
- Extend migration (`AutoMigrate`) without breaking `Ping` data.
- Validation:
- App startup migrates cleanly.
- Duplicate email creation fails predictably.
- Existing `pings` data remains intact.
- Risks:
- Email case-normalization bugs create duplicate accounts.
- Migration strategy may be fragile for future schema changes.

## Phase 2: Auth Services
- Tasks:
- Implement password helpers (`HashPassword`, `CheckPassword`).
- Implement token helpers (`GenerateAccessToken`, `ValidateAccessToken`).
- Add env-driven auth config (`JWT_SECRET`, token TTL, bcrypt cost).
- Validation:
- Unit tests for hash/verify and token lifecycle.
- Tampered/expired token rejection verified.
- Startup fails fast on missing critical secret.
- Risks:
- Weak secret management compromises auth.
- Incomplete claim checks create authorization flaws.

## Phase 3: HTTP Endpoints + Middleware
- Tasks:
- Add `POST /auth/signup` (validate, normalize, hash, persist).
- Add `POST /auth/login` (credential check, token issue).
- Add `GET /auth/me` (authenticated profile view).
- Add auth middleware for bearer token parsing + user context loading.
- Apply middleware to protected route group.
- Validation:
- Handler tests for success and failure paths.
- Protected routes return `401` without valid token.
- Authenticated flows work end-to-end.
- Risks:
- Inconsistent error shape slows client integration.
- Middleware context/load ordering issues can cause `500`s.

## Phase 4: Security Hardening
- Tasks:
- Enforce input validation (email/password rules).
- Standardize non-enumerating auth failure responses.
- Ensure no password/token leakage in logs.
- Define rate-limit strategy for auth endpoints (or explicit defer).
- Validation:
- Negative tests for malformed payloads and invalid creds.
- Manual log review confirms no secret leakage.
- Risks:
- Brute-force exposure without rate limiting.
- Overly strict validation may reject legitimate clients.

## Phase 5: Testing, Docs, Rollout
- Tasks:
- Add end-to-end test: signup -> login -> protected call.
- Add regressions for unchanged public routes.
- Update `README.md` with auth env vars and curl examples.
- Add rollout notes and rollback guidance.
- Validation:
- `go test ./...` passes.
- Manual smoke checks align with docs.
- App boot behavior with auth config verified.
- Risks:
- Gaps in unhappy-path tests hide regressions.
- Docs drift from implementation details.

## Concrete Checklist
- Add `User` model and migration.
- Add email normalization.
- Add bcrypt utilities.
- Add JWT utilities.
- Add auth config loader.
- Add signup/login/me handlers.
- Add auth middleware and protected route grouping.
- Add unit + handler/integration tests.
- Update docs and run validation suite.

## Validation Matrix
- Unit: password hashing, token validation, normalization.
- Integration: signup/login/me success and failures.
- Authorization: missing/invalid token rejection.
- Regression: existing public endpoint behavior unchanged.
- Operational: startup fails clearly on invalid critical auth config.
