# Tester-API
Tester API Written in Go, this is a means of validating another tool

## Time Endpoint

`GET /time` returns the current time in RFC3339 format.

- Header: `X-Timezone` (optional, IANA timezone name like `America/New_York`)
- Default timezone: `UTC` when header is missing
- Invalid timezone: returns `400` with JSON error

Examples:

```bash
# Default UTC
curl -s http://localhost:8080/time
```

```bash
# Custom timezone
curl -s -H 'X-Timezone: America/New_York' http://localhost:8080/time
```

```bash
# Invalid timezone
curl -s -i -H 'X-Timezone: Invalid/Zone' http://localhost:8080/time
```
