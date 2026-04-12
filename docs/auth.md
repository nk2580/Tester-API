# Authentication API

## Endpoints

- `POST /auth/register`
  - Request: `{ "email": "user@example.com", "password": "StrongPassw0rd!" }`
  - Responses: `201`, `400`, `409`, `429`

- `POST /auth/login`
  - Request: `{ "email": "user@example.com", "password": "StrongPassw0rd!" }`
  - Response: `{ "access_token": "...", "token_type": "Bearer", "expires_in": 900, "user": {...} }`
  - Responses: `200`, `400`, `401`, `403`, `429`

- `POST /auth/logout`
  - Header: `Authorization: Bearer <access_token>`
  - Responses: `200`, `401`

- `GET /auth/me`
  - Header: `Authorization: Bearer <access_token>`
  - Responses: `200`, `401`

- `POST /auth/password-reset/request`
  - Request: `{ "email": "user@example.com" }`
  - Generic success response to prevent account enumeration.
  - Responses: `200`, `400`, `429`

- `POST /auth/password-reset/confirm`
  - Request: `{ "token": "...", "new_password": "NewStrongPassw0rd!" }`
  - Responses: `200`, `400`, `429`

## Error Contract

All errors follow this envelope:

```json
{
  "error": {
    "code": "validation_error",
    "message": "Invalid login payload",
    "request_id": "<request-id>"
  }
}
```
