# Auth Deployment Checklist

1. Set production env vars:
   - `APP_ENV=prod`
   - `APP_SECRET` (>= 32 chars)
   - `DB_PATH`
   - `BASE_URL`
   - `BCRYPT_COST`
   - `ACCESS_TOKEN_TTL`
   - `PASSWORD_RESET_TOKEN_TTL`
   - `ALLOWED_ORIGINS`
   - `EMAIL_PROVIDER` and SMTP credentials when `smtp`
2. Roll out migration-compatible version before traffic cutover.
3. Run smoke tests:
   - `GET /health`
   - register/login/logout flow
   - password reset request/confirm flow
   - legacy endpoints `/ping`, `/pings`, `/hello`, `/time`
4. Verify logs include `request_id` and no secrets.
5. Monitor rate-limit and auth failure metrics post-deploy.
