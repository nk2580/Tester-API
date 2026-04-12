# Tester-API

Production-ready Gin + GORM API with:
- Ping endpoints (`/ping`, `/pings`, `/hello`, `/time`)
- Authentication (`/auth/register`, `/auth/login`, `/auth/logout`, `/auth/me`)
- Password reset flow (`/auth/password-reset/request`, `/auth/password-reset/confirm`)

## Run

```bash
go run .
```

## Key Environment Variables

- `APP_ENV` (`dev`, `test`, `prod`)
- `LISTEN_ADDR` (default `:8080`)
- `DB_PATH` (default `db/data.db`)
- `APP_SECRET`
- `BCRYPT_COST` (10-14)
- `ACCESS_TOKEN_TTL` (default `15m`)
- `PASSWORD_RESET_TOKEN_TTL` (default `30m`)
- `ALLOWED_ORIGINS` (comma-separated)
- `EMAIL_PROVIDER` (`log` or `smtp`)

See [auth docs](docs/auth.md) and [deployment checklist](docs/deployment-checklist.md).
