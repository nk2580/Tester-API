# Tester-API

Go API using Gin + GORM + SQLite.

## Configuration

Set the following environment variables before running the API:

- `JWT_SECRET` (required): secret key used to sign JWT access tokens.
- `JWT_TTL` (optional): access token TTL in seconds. Default: `3600`.
- `DB_PATH` (optional): path to the SQLite database file. Default: `db/data.db`.
- `HTTP_ADDRESS` (optional): address passed to `gin.Engine.Run`. Default: `:8080`.

## Project Layout

The monolithic `main.go`/`app.go` combo has been split into focused packages:

- `cmd/api` – the composition root that loads configuration, opens the database, and runs the server.
- `internal/config` – environment-driven configuration loading with sane defaults.
- `internal/data` – data models, store interfaces, and the SQLite-backed implementation.
- `internal/auth`, `internal/ping`, `internal/time` – domain-specific HTTP handlers and middleware.
- `internal/server` – dependency injection, router creation, and test-facing helpers.
- `internal/httputil` – shared HTTP helpers (e.g., JSON error responses).

## Run

```bash
export JWT_SECRET="replace-me"
go run ./cmd/api
```

## Endpoints

Public endpoints:

- `POST /ping`
- `GET /pings`
- `GET /hello`
- `GET /time`
- `POST /auth/signup`
- `POST /auth/login`

Protected endpoints:

- `GET /auth/me` (requires `Authorization: Bearer <token>`)

## Auth API

### `POST /auth/signup`

Request body:

```json
{
  "email": "user@example.com",
  "password": "supersecret"
}
```

Success (`201`):

```json
{
  "user": {
    "id": 1,
    "email": "user@example.com",
    "created_at": "2026-04-13T12:00:00Z"
  },
  "token": {
    "access_token": "<jwt>",
    "token_type": "Bearer",
    "expires_in": 3600
  }
}
```

### `POST /auth/login`

Request body:

```json
{
  "email": "user@example.com",
  "password": "supersecret"
}
```

Success (`200`) returns the same payload shape as signup.

### `GET /auth/me`

Returns the authenticated user profile.

### Error format

Errors return a consistent schema:

```json
{
  "error": {
    "code": "invalid_credentials",
    "message": "invalid email or password"
  }
}
```

## Security Notes

- Passwords are hashed using `bcrypt`.
- Access tokens are JWTs (HMAC SHA-256) with `sub`, `iat`, and `exp` claims.
- Baseline abuse mitigation (rate limiting) is not implemented in this ticket and should be added at API gateway or middleware level.
