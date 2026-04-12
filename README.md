# Tester-API

Production-oriented Gin API with authentication, registration, email verification, and password reset flows.

## Run

```bash
go run .
```

Server defaults to `:8080` and SQLite at `db/data.db`.

## Authentication Endpoints

- `POST /auth/register`
- `POST /auth/verify-email`
- `POST /auth/resend-verification`
- `POST /auth/login`
- `POST /auth/logout`
- `POST /auth/forgot-password`
- `POST /auth/reset-password`
- `GET /auth/me` (requires authenticated session cookie)

## Security Controls Implemented

- Argon2id password hashing
- Opaque random auth/session and reset/verification tokens
- Hashed token storage (SHA-256) with single-use + expiry checks
- Rate limiting on all auth mutation endpoints
- Account lockout/backoff after repeated failed logins
- Uniform forgot-password and duplicate registration responses to reduce user enumeration
- Session invalidation on password reset
- HttpOnly strict-samesite session cookie
- Audit event table for auth-sensitive actions

## Configuration

All values can be overridden by environment variables:

- `DB_PATH` (default `db/data.db`)
- `SERVER_ADDR` (default `:8080`)
- `SESSION_COOKIE_NAME` (default `session_token`)
- `SESSION_TTL` (default `24h`)
- `VERIFICATION_TOKEN_TTL` (default `24h`)
- `PASSWORD_RESET_TOKEN_TTL` (default `30m`)
- `RESEND_VERIFICATION_COOLDOWN` (default `60s`)
- `LOCKOUT_DURATION` (default `15m`)
- `MAX_FAILED_LOGIN_ATTEMPTS` (default `5`)
- `COOKIE_SECURE` (default `false`; set `true` in production)
- `COOKIE_DOMAIN` (optional)
- `PASSWORD_MIN_LENGTH` (default `12`)

## Tests

```bash
go test ./...
```
