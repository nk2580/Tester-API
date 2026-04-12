# Implementation Plan: TESTERAPI-AUTH-20260412-3

## Scope
Build a production-grade authentication system for the current Gin + GORM service, including:
- User registration
- Login/logout with secure session handling
- Password reset via time-limited token flow
- Security hardening, observability, and test coverage

## Assumptions
- API remains backend-only (no UI work in this ticket).
- Email delivery will use an SMTP/provider abstraction with real provider config in non-local environments.
- SQLite is acceptable for local/dev; schema will be production-compatible for future Postgres migration.

## Phase 1: Foundation and Architecture
### Tasks
- Refactor `main.go` into clear modules: `config`, `db`, `models`, `handlers`, `services`, `middleware`, `routes`.
- Add environment-based configuration for secrets, token TTLs, hashing cost, email sender, and base URL.
- Add centralized error response format and request validation conventions.
- Add structured logging with request ID propagation.
- Add migration strategy (GORM migrations or migration tool) and stop relying on ad hoc auto-migrate only.

### Validation
- Service boots with modular wiring and health endpoint.
- Config validation fails fast on missing required secrets in non-dev mode.
- Logs include request ID and consistent error shape.

### Risks
- Refactor can break existing endpoints if route wiring changes.
- Configuration drift between local and deployment environments.

## Phase 2: Data Model and Security Primitives
### Tasks
- Create `users` model with fields: `id`, `email` (unique, normalized), `password_hash`, `is_verified`, `created_at`, `updated_at`, `last_login_at`.
- Create `password_reset_tokens` model with fields: `id`, `user_id`, `token_hash`, `expires_at`, `used_at`, `created_at`.
- Add indexes for `email`, `user_id`, `expires_at`.
- Implement password hashing with `bcrypt` (or `argon2id` if chosen), with configurable cost.
- Add secure random token generator and hash-at-rest pattern for reset tokens.
- Add email normalization and password policy validation.

### Validation
- DB constraints enforce unique email and valid relations.
- Unit tests verify hash/verify, token generation entropy, and expiration behavior.
- Reset tokens stored only as hashes, never plaintext.

### Risks
- Weak password policy can create security gap.
- Token handling bugs can allow replay or enumeration.

## Phase 3: Registration and Login
### Tasks
- Implement `POST /auth/register` with:
- Input validation (`email`, `password`)
- Duplicate email handling with safe error wording
- User creation with hashed password
- Implement `POST /auth/login` with:
- Credential verification
- Account state checks
- Session/token issuance
- Choose auth strategy and implement securely:
- Preferred: short-lived access token + refresh token rotation
- Alternative: secure server-side session cookie
- Add auth middleware for protected routes (`Authorization` bearer or cookie).
- Add `POST /auth/logout` and token/session invalidation flow.

### Validation
- Integration tests for success and failure paths.
- No user enumeration in registration/login responses.
- Protected endpoint rejects unauthenticated requests and accepts valid auth context.

### Risks
- Incorrect token/session invalidation can keep compromised sessions active.
- Clock skew and TTL misconfiguration can cause false auth failures.

## Phase 4: Password Reset Flow
### Tasks
- Implement `POST /auth/password-reset/request`:
- Accept email
- Always return generic success response
- If user exists, create single-use reset token record and send email with reset link
- Implement `POST /auth/password-reset/confirm`:
- Accept token + new password
- Validate token hash, expiry, and unused state
- Mark token used atomically
- Update password hash
- Invalidate active sessions/refresh tokens for that user
- Add optional throttling per IP/email for reset requests.

### Validation
- Token cannot be reused after success.
- Expired tokens fail reliably.
- Password update invalidates prior sessions.
- Integration tests cover unknown email, expired token, replay attempt, and happy path.

### Risks
- Race conditions can allow token replay if not atomic.
- Email delays can reduce reset usability near expiry.

## Phase 5: Hardening, Abuse Prevention, and Ops
### Tasks
- Add rate limiting for register, login, and reset endpoints.
- Add optional account lockout/backoff for repeated failed logins.
- Add security headers and strict CORS policy.
- Ensure secrets are not logged; redact sensitive fields in logs.
- Add audit events for auth lifecycle actions (register/login/reset/logout).
- Add OpenAPI/API docs for auth endpoints and error contracts.

### Validation
- Load test auth endpoints with limiter behavior checks.
- Security test checklist: brute force, enumeration, replay, weak password bypass.
- Manual verification in staging with real email provider and realistic TTLs.

### Risks
- Overly strict limits can block legitimate users.
- Missing observability can hide abuse patterns.

## Phase 6: Testing, Release, and Rollout
### Tasks
- Add unit tests for services, validators, token logic, middleware.
- Add integration tests using test DB for end-to-end auth flows.
- Add regression tests for existing `/ping`, `/pings`, `/hello`, `/time`.
- Add deployment checklist with secret provisioning and migration order.
- Roll out behind feature flag or staged release gates if possible.

### Validation
- CI passes: unit + integration + lint/static checks.
- Migration and rollback tested in staging.
- Post-deploy smoke tests verify register/login/reset/logout behavior.

### Risks
- Insufficient automated coverage can miss edge-case auth failures.
- Migration mistakes can block startup in production.

## Deliverables
- New auth module(s) with documented endpoints.
- DB schema/migrations for users and reset tokens.
- Secure token/session implementation with middleware.
- Password reset email flow and templates.
- Test suite updates and runbook for deployment and incident response.
