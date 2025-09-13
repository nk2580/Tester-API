# SPEC

Detailed Feature Description

This feature adds Kubernetes-style health endpoints to the API so the platform (kubelet, service mesh, load balancer) can reliably determine liveness and readiness of the running process.

Motivation and value
- Kubernetes expects two distinct probe types:
  - Liveness probe: answers whether the process is alive and should be restarted if it is not.
  - Readiness probe: answers whether the process is ready to receive traffic (all required dependencies are reachable and the app is ready).
- Without these endpoints, Kubernetes may route traffic to an instance that cannot serve requests (leading to errors) or may not detect a hung process in time (leading to extended downtime).
- Adding structured, observable health endpoints reduces downtime, improves rolling upgrades and safe deployments, and enables simple SRE automation and alerting.

Scope
- Implement two HTTP endpoints:
  - GET /healthz — Liveness probe (fast, indicates process is alive).
  - GET /readyz — Readiness probe (runs a small set of checks; exposes per-check status).
- Integrate with existing Go HTTP server (main.go) and existing SQLite DB (db/data.db).
- Provide graceful-shutdown integration so readiness flips to "not ready" immediately on shutdown.
- Provide lightweight Prometheus metrics and structured logs for observability.
- Provide feature flags / environment configuration to tune timeouts and to allow an emergency kill-switch.

Out of scope
- Deep application-level checks beyond DB/connectivity (e.g., external third-party SaaS checks) — these can be added later as additional check implementations.
- UI changes or browser-facing pages.

Sub-features
- Liveness endpoint: extremely fast, returns process-up status.
- Readiness endpoint: composite checks (DB ping), timeouts, JSON detail body.
- Graceful-shutdown readiness flip and integration.
- Observability: metrics + logs for health checks.
- Configuration and feature-flag controls.

User Stories

Liveness
- As a cluster operator, I want a liveness probe endpoint GET /healthz, so that the kubelet can detect and restart unhealthy container processes.
- As a developer, I want the liveness endpoint to be extremely fast and deterministic, so that Kubernetes does not restart a healthy process due to expensive checks.

Readiness
- As a cluster operator, I want a readiness probe endpoint GET /readyz that verifies the API's critical dependencies (SQLite DB), so that traffic is only routed to instances that can serve requests.
- As a release manager, I want the readiness check behavior to be toggleable via an environment flag during maintenance windows, so that I can drain traffic manually without failing probes.

Graceful shutdown
- As a developer, I want the service to become immediately not-ready after receiving a termination signal, so Kubernetes will stop routing new requests before it terminates the process.

Observability
- As an SRE, I want health-check metrics and structured logs so that failures and regressions are visible in monitoring and alerting systems.

Cucumber Scenarios

Liveness - success
Given the API process is running and not in the midst of shutdown
When a client (kubelet) performs GET /healthz
Then the API returns HTTP 200 with a JSON body containing "status":"ok" within 200ms

Readiness - success (DB reachable)
Given the API process is running and the SQLite DB at db/data.db is reachable
When a client performs GET /readyz
Then the API returns HTTP 200 with JSON body:
  { "status":"ok", "checks":[{"name":"db","status":"ok","duration_ms":<n>}], "timestamp": "<iso8601>" }
and each check duration is < configured readiness timeout (default 500ms)

Readiness - failure (DB unreachable)
Given the API process is running and the SQLite DB is unreachable (connection fails or SELECT 1 times out)
When a client performs GET /readyz
Then the API returns HTTP 503 with JSON body:
  { "status":"fail", "checks":[{"name":"db","status":"fail","error":"<short message>","duration_ms":<n>}], "timestamp": "<iso8601>" }

Graceful shutdown readiness flip
Given the API has received SIGTERM and is in graceful shutdown mode
When a client performs GET /readyz
Then the API returns HTTP 503 and the JSON body marks "status":"fail" (readiness false) immediately

Feature flag disables readiness checks
Given the environment variable HEALTH_READINESS_DISABLED=true
And the SQLite DB is unreachable
When a client performs GET /readyz
Then the API returns HTTP 200 with JSON {"status":"ok", "checks":[]} (or a short note showing checks are disabled)

Observability - metrics increment
Given metrics are enabled and instrumented
When /healthz or /readyz are called
Then a Prometheus counter increments and a histogram records the duration for the health check

# DESIGN

Technical overview

What must change
- Add a small health subsystem (package internal/health) that provides:
  - HTTP handlers for /healthz and /readyz
  - A pluggable check interface (so new checks can be added later)
  - A configuration layer via environment variables
  - Prometheus-compatible metrics (optional toggled)
  - Integration points for graceful shutdown (an atomic readiness flag)
- Wire the health handlers into the existing HTTP server in main.go.
- Implement a DB readiness check that uses the existing *sql.DB or opens a lightweight connection to db/data.db and runs a simple SELECT 1 (or db.PingContext).
- Add unit tests and a small integration smoke test (httptest or in-memory sqlite).

What can be reused
- Existing HTTP server setup in main.go, existing database/sql usage (if main.go already opens DB). Reuse the same *sql.DB instance by passing it to health.NewReadinessChecker(db).
- Standard library (net/http, database/sql, context, sync/atomic).
- Existing logging library (if present) — fall back to log.Printf if no structured logger is configured.

Constraints
- Keep check duration low (default readiness timeout 500ms).
- Avoid heavy dependencies; prefer standard library and a small Prometheus client dependency only if already used by the project. Make metrics optional behind an env var.
- Do not expose sensitive internal error details in production logs/responses by default; responses should include concise error messages for debugging but not leak secrets.

Component breakdown (files and responsibilities)

- internal/health/
  - health.go
    - Types: HealthStatus, CheckResult, Check interface
    - Registry for checks and orchestration of check execution (parallel with context timeout)
  - checks_db.go
    - DB checker implementation: DBChecker struct that uses *sql.DB or DSN and PingContext/SELECT 1
  - handlers.go
    - HTTP handlers for /healthz (livenessHandler) and /readyz (readinessHandler)
    - JSON response marshalling and mapping status->HTTP codes
  - config.go
    - Read configuration from env vars:
      - HEALTH_READINESS_TIMEOUT_MS (default 500)
      - HEALTH_METRICS_ENABLED (default false)
      - HEALTH_READINESS_DISABLED (default false)
      - HEALTH_LISTEN_PATHS (optional override)
  - metrics.go
    - Optional Prometheus metrics (counters: health_requests_total{endpoint,status}, histograms: health_duration_seconds)
    - Expose instrumentation hooks but keep optional
  - health_test.go
    - Unit tests for handlers and checks using httptest and in-memory sqlite or mocks

- main.go (modified)
  - Import internal/health package, instantiate checks (DBChecker)
  - Register /healthz and /readyz
  - Wire graceful shutdown so readiness flips to false on SIGTERM and waits for in-flight requests to drain

- README.md (modified)
  - Add usage and Kubernetes probe snippet examples (liveness/readiness YAML)

- ci/ or .github/workflows (optional)
  - Add a quick smoke-test step that runs the server and validates both endpoints (optional, recommended)

ASCII component hierarchy
- api-server (main.go)
  - internal/health
    - handlers.go (healthz, readyz)
    - health.go (orchestrator)
    - checks_db.go (DBChecker)
    - metrics.go (optional instrumentation)
    - config.go (env parsing)
  - existing components (DB, routers, business logic)

ASCII JSON response wireframes

/healthz (liveness) - success
{
  "status": "ok",
  "timestamp": "2025-09-13T12:34:56Z"
}

/readyz (readiness) - success
{
  "status": "ok",
  "checks": [
    {"name":"db","status":"ok","duration_ms":12}
  ],
  "timestamp": "2025-09-13T12:34:56Z"
}

/readyz (readiness) - fail
{
  "status": "fail",
  "checks": [
    {"name":"db","status":"fail","error":"connection timeout","duration_ms":500}
  ],
  "timestamp": "2025-09-13T12:34:56Z"
}

ASCII data flow diagram

kubelet / load‑balancer
        |
        v
   HTTP GET /healthz, /readyz
        |
        v
   net/http -> health.handlers
        |
   +----+-----+
   |          |
   v          v
liveness   readiness orchestrator
             |
    +--------+--------+
    |                 |
 DBChecker (db.Ping)  (possible future checks)
    |
    v
  sqlite (db/data.db)

Logic & business rules (mapping to SPEC scenarios)
- Liveness:
  - Rule: /healthz returns 200 if process is running and not in a fatal state.
  - Implementation: livenessHandler returns status "ok" and HTTP 200 unless the process has crashed; very fast, no blocking checks.
  - Scenario mapping: "Liveness - success".

- Readiness:
  - Rule: /readyz aggregates a configurable list of checks (AND semantics). If any check fails or times out, readiness returns HTTP 503 and "fail".
  - Rule: Each check runs with the same per-check timeout (default 500ms). If a check exceeds timeout, mark check as fail with error "timeout".
  - Rule: Optionally disable checks by setting HEALTH_READINESS_DISABLED=true; when disabled, /readyz returns 200 and an explicit marker in the JSON that checks are skipped.
  - Scenario mapping: "Readiness - success", "Readiness - failure", "Feature flag".

- Graceful shutdown:
  - Rule: On SIGTERM, set the readiness flag to false immediately before waiting for server shutdown. /readyz must begin returning 503 as soon as shutdown begins.
  - Implementation: atomic boolean "isReady" starts true; on shutdown set isReady=false.
  - Scenario mapping: "Graceful shutdown readiness flip".

- Observability:
  - Rule: Each probe call increments a counter and records a duration histogram; failures produce structured log entries with check names, error and duration.
  - Scenario mapping: "Observability - metrics increment".

API Endpoints

1) GET /healthz
- Purpose: Liveness probe
- Method: GET
- Path: /healthz
- Query params: none
- Request body: none
- Response:
  - 200 OK
    - Body: { "status":"ok", "timestamp":"<iso8601>" }
  - 500 Internal Server Error (rare, only if handler fails)
    - Body: { "status":"fail", "error":"<short message>" }
- Headers: Content-Type: application/json
- Notes: Fast path, no dependency checks.

2) GET /readyz
- Purpose: Readiness probe (composite)
- Method: GET
- Path: /readyz
- Query params: none
- Response:
  - 200 OK
    - Body: { "status":"ok", "checks":[...], "timestamp": "<iso8601>" }
  - 503 Service Unavailable
    - Body: { "status":"fail", "checks":[...], "timestamp":"<iso8601>" }
  - 504 Gateway Timeout (if overall check orchestration times out) — prefer 503 with per-check timeout details
- Check result schema:
  - name: string
  - status: "ok" | "fail"
  - duration_ms: int
  - error: optional string
- Notes:
  - The handler should use a context with timeout (configured by HEALTH_READINESS_TIMEOUT_MS).
  - Default overall behavior: all checks run in parallel and are combined with AND semantics.

Database schema changes
- No database schema changes are required.
- The readiness check performs a lightweight connectivity check (db.PingContext or SELECT 1).
- Migration tasks: none required; include a verify/no-migration-needed task.

Non-functional requirements

Security
- Health endpoints should not return sensitive configuration or secrets. Errors should be short and suitable for logs; avoid stack traces in responses.
- In Kubernetes, probes will originate from within the cluster; network policies can be used to restrict access if necessary.
- Optional: bind health endpoints to the same HTTP server; if additional isolation is required, provide a configuration option to bind them to localhost or a separate port.

Performance
- Readiness default timeout: 500ms per check (configurable).
- Health handlers must be non-blocking beyond configured timeouts.
- Checks execute in parallel (bounded by number of checks), to keep readiness latency low.

Observability
- Expose metrics (Prometheus) for:
  - health_request_total{endpoint="/readyz" | "/healthz", outcome="ok"|"fail"}
  - health_duration_seconds histogram
  - health_check_failures_total{name="<check>"}
- Logs:
  - Structured log entries per failed check: { "component": "health", "check":"db", "error":"...","duration_ms":123 }
- Tracing:
  - If the project uses tracing (OpenTelemetry), add spans around composite health checks.

Feature flags & config
- Configuration via environment variables:
  - HEALTH_READINESS_TIMEOUT_MS (default 500)
  - HEALTH_METRICS_ENABLED (default false)
  - HEALTH_READINESS_DISABLED (default false)
  - HEALTH_LOG_LEVEL (reuse existing)
- Emergency kill-switch:
  - HEALTH_READINESS_DISABLED=true: readiness checks are skipped and /readyz returns OK (use cautiously).

Rollout strategy
- Implement behind a feature branch; open PR and code review.
- Canary rollout: apply new image to a small subset of pods, verify metrics and readiness behavior over 30 minutes.
- Use feature flag to disable readiness checks in emergencies (toggle env var via k8s patch) without changing code.

Assumptions & Open questions
- Assumption: main.go has a single HTTP server and (optionally) already opens a *sql.DB; plan includes wiring the DB handle into health package. If the repo opens DB lazily, health package will accept a DSN fallback.
- Open: Does the project already have a metrics library (Prometheus)? If not, metrics are optional and can be added later.
- Open: Is there a preferred structured logging library (zap/logrus)? If present, re-use; otherwise fall back to log.Printf in health package.
- Open: Should /readyz perform a single DB ping or also check migration status? Currently implemented as a connectivity ping; migration-aware readiness can be added later.

# TASKS

## Liveness & readiness core implementation

- [ ] **Task 1.1: Create health package skeleton**
- **Status:** Pending
- **Context:**
 - Create internal/health package with files: health.go, handlers.go, checks_db.go, config.go, metrics.go, health_test.go.
 - Matches DESIGN: component breakdown and pluggable checks.
- **Dependencies:** None
- **Validation:**
 - [ ] internal/health/health.go exists
 - [ ] internal/health/handlers.go exists
 - [ ] internal/health/checks_db.go exists
 - [ ] internal/health/config.go exists
 - [ ] internal/health/metrics.go exists (may be empty until Task 1.9)
 - [ ] health_test.go file scaffold exists

- [ ] **Task 1.2: Implement liveness handler (/healthz)**
- **Status:** Pending
- **Context:**
 - Implement livenessHandler in internal/health/handlers.go per API spec.
 - No checks; returns 200 if service running and not in fatal state.
 - Map to SPEC scenario "Liveness - success".
- **Dependencies:** Task 1.1
- **Validation:**
 - [ ] internal/health/handlers.go exports livenessHandler
 - [ ] Unit test TestLivenessHandler verifies HTTP 200 and JSON body
 - [ ] go test ./... passes for health package tests

- [ ] **Task 1.3: Implement readiness orchestrator and DB check**
- **Status:** Pending
- **Context:**
 - Implement Check interface and orchestrator in health.go that runs registered checks in parallel with context timeout.
 - Implement DBChecker in checks_db.go which uses *sql.DB or DSN and PingContext/SELECT 1.
 - Default timeout from config.go: HEALTH_READINESS_TIMEOUT_MS (500ms).
 - Map to SPEC scenarios "Readiness - success" and "Readiness - failure".
- **Dependencies:** Task 1.1
- **Validation:**
 - [ ] internal/health/health.go contains Check interface and RunChecks(ctx) function
 - [ ] internal/health/checks_db.go contains DBChecker with Ping-based check
 - [ ] Unit test TestDBCheckerSuccess and TestDBCheckerTimeout exist and pass

- [ ] **Task 1.4: Implement readiness handler (/readyz)**
- **Status:** Pending
- **Context:**
 - Implement readinessHandler which:
   - Checks the "isReady" atomic flag (set to false when shutdown begins)
   - If HEALTH_READINESS_DISABLED=true then short-circuit to OK
   - Otherwise runs registered checks, collects CheckResult, builds response JSON and returns 200 or 503
 - Matches API endpoint schema in DESIGN.
- **Dependencies:** Tasks 1.1, 1.3
- **Validation:**
 - [ ] internal/health/handlers.go exports readinessHandler
 - [ ] Unit tests:
   - TestReadinessSuccess (DB reachable) passes
   - TestReadinessFail (DB unreachable) passes
 - [ ] JSON response conforms to schema (status, checks, timestamp)

- [ ] **Task 1.5: Wire health handlers into main.go and graceful shutdown**
- **Status:** Pending
- **Context:**
 - Modify main.go to:
   - Instantiate health components (pass existing *sql.DB or DSN)
   - Register /healthz and /readyz handlers on the server mux
   - On SIGTERM/SIGINT set health.IsReady=false (atomic) before server.Shutdown
 - Ensures readiness flips immediately during graceful shutdown.
- **Dependencies:** Tasks 1.2, 1.3, 1.4
- **Validation:**
 - [ ] main.go import and register handlers
 - [ ] Local run: start server and confirm /healthz and /readyz reachable on configured port
 - [ ] Graceful shutdown simulation (send SIGTERM) and check /readyz returns 503 afterwards

## Configuration, feature flags & docs

- [ ] **Task 2.1: Add configuration env parsing**
- **Status:** Pending
- **Context:**
 - Implement config.go: parse HEALTH_READINESS_TIMEOUT_MS, HEALTH_METRICS_ENABLED, HEALTH_READINESS_DISABLED.
 - Provide sensible defaults and validation.
- **Dependencies:** Task 1.1
- **Validation:**
 - [ ] internal/health/config.go exists and exports config struct
 - [ ] Unit test TestConfigDefaults verifies defaults

- [ ] **Task 2.2: Add README and k8s snippet**
- **Status:** Pending
- **Context:**
 - Update README.md with usage, env vars, and recommended Kubernetes liveness/readiness probe examples (httpGet path, periodSeconds, timeoutSeconds).
- **Dependencies:** Tasks 1.2, 1.4, 2.1
- **Validation:**
 - [ ] README.md updated with health endpoint section and sample YAML
 - [ ] The sample YAML contains the configured probe paths and recommended timeouts

## Testing & CI

- [ ] **Task 3.1: Unit tests for health package**
- **Status:** Pending
- **Context:**
 - Write unit tests in internal/health/health_test.go:
   - TestLivenessHandler returns 200
   - TestReadiness_AllOk
   - TestReadiness_DBFail
   - TestReadiness_DisabledFlag
 - Use httptest and in-memory sqlite (DSN=:memory:) where applicable.
- **Dependencies:** Tasks 1.2, 1.3, 1.4
- **Validation:**
 - [ ] go test ./internal/health passes
 - [ ] Tests cover both success and failure paths for readiness

- [ ] **Task 3.2: Integration smoke test (CI)**
- **Status:** Pending
- **Context:**
 - Add a CI step (script) that:
   - Builds the server
   - Runs it in background with DB in-memory or temp file
   - Performs curl checks for /healthz and /readyz expecting 200
 - This is a minimal smoke test for PRs.
- **Dependencies:** Task 3.1
- **Validation:**
 - [ ] CI job script added (e.g., .github/workflows/health-smoke.yml or a script in ci/)
 - [ ] Job runs locally (manual test) and passes

## Observability & metrics

- [ ] **Task 4.1: Add Prometheus metrics hooks (optional)**
- **Status:** Pending
- **Context:**
 - Implement metrics.go that registers counters/histograms if HEALTH_METRICS_ENABLED=true.
 - Expose metrics via existing /metrics endpoint if present, otherwise document how to scrape.
- **Dependencies:** Tasks 1.4, 2.1
- **Validation:**
 - [ ] metrics are registered when enabled
 - [ ] Calling /readyz increments a health_request_total metric

- [ ] **Task 4.2: Add structured logging for failures**
- **Status:** Pending
- **Context:**
 - Ensure health handlers and checkers log structured entries for failures including check name, error and duration.
 - Re-use project logging library if available.
- **Dependencies:** Tasks 1.3, 1.4
- **Validation:**
 - [ ] Log entries produced on check failure contain check name, error and duration_ms
 - [ ] Unit test asserts logs include expected fields (or manual verification)

## Migrations, release & rollout

- [ ] **Task 5.1: Verify DB schema migrations not required**
- **Status:** Pending
- **Context:**
 - Document that no DB schema migration is needed for health checks; readiness uses db.Ping/SELECT 1.
- **Dependencies:** None
- **Validation:**
 - [ ] README or migration notes state "No DB migration required for health checks"

- [ ] **Task 5.2: Add k8s deployment probe examples and rollout plan**
- **Status:** Pending
- **Context:**
 - Provide recommended liveness/readiness probe YAML and rollout steps (canary, monitor metrics, scale).
- **Dependencies:** Task 2.2
- **Validation:**
 - [ ] YAML snippet present in README
 - [ ] Rollout checklist added (canary, monitor, rollback steps)

- [ ] **Task 5.3: Rollout & rollback runbook**
- **Status:** Pending
- **Context:**
 - Document how to enable / disable readiness checks via env var, how to revert the deployment or set the kill-switch if health checks cause unintended failures.
- **Dependencies:** Task 2.1, Task 5.2
- **Validation:**
 - [ ] Runbook added to README or ops/ directory
 - [ ] Revert steps include "set HEALTH_READINESS_DISABLED=true" and redeploy or roll back image tag

# VERIFICATION

Artifact Check
- Confirm file exists:
  - .frunl/019942bc-60a4-73e8-a673-032840fec801/plan.md

Traceability Table (Scenario → Design Element(s) → Task ID(s))

- Liveness - success
  - Design elements: internal/health/handlers.go (livenessHandler)
  - Tasks: 1.2, 1.1, 1.5

- Readiness - success (DB reachable)
  - Design elements: internal/health/health.go (orchestrator), internal/health/checks_db.go (DBChecker), internal/health/handlers.go (readinessHandler)
  - Tasks: 1.3, 1.4, 1.1, 1.5

- Readiness - failure (DB unreachable)
  - Design elements: internal/health/checks_db.go, handlers.go error path
  - Tasks: 1.3, 1.4, 3.1

- Graceful shutdown readiness flip
  - Design elements: main.go wiring to set readiness flag, handlers check isReady
  - Tasks: 1.5

- Feature flag disables readiness checks
  - Design elements: internal/health/config.go, handlers.go short-circuit
  - Tasks: 2.1, 1.4

- Observability - metrics increment
  - Design elements: internal/health/metrics.go
  - Tasks: 4.1, 4.2

Test Strategy (high-level unit tests only)

Unit tests to implement (high-level descriptions)
- TestLivenessHandler
  - Start a httptest server with livenessHandler, call GET /healthz, expect HTTP 200 and JSON with status "ok".

- TestDBCheckerSuccess
  - Use in-memory sqlite (DSN=:memory:) with db.PingContext confirming DB is reachable; DBChecker returns ok.

- TestDBCheckerTimeout
  - Mock or wrap a checker to sleep beyond timeout and confirm orchestrator marks check as fail (timeout) and overall readiness is fail.

- TestReadinessAllOk
  - Register DBChecker and call readinessHandler via httptest; assert HTTP 200 and checks array status "ok".

- TestReadinessFail
  - Simulate DB unreachable (bad DSN or closed connection); call readinessHandler and assert HTTP 503 and check error message present.

- TestReadinessDisabledFlag
  - Set HEALTH_READINESS_DISABLED=true in test config; call readinessHandler with DB unreachable; assert HTTP 200 and no failing checks.

Example commands
- Run unit tests:
  - go test ./internal/health -v
  - go test ./...
- Smoke curl checks (after starting the server locally on :8080):
  - curl -i -s http://localhost:8080/healthz
  - curl -i -s http://localhost:8080/readyz
  - Expected codes: 200 for both in normal healthy state; 503 for readyz when DB unreachable.

Operational readiness (metrics, logs, feature flag, rollback)

Metrics & alerts
- Required metrics to expose (Prometheus):
  - health_requests_total{endpoint, outcome}
  - health_check_failures_total{name}
  - health_duration_seconds (histogram)
- Recommended alert rule examples:
  - Alert when readyz failure rate > 5% over 5 minutes across all pods.
  - Alert when a single pod has consecutive readiness failures (3 in a row).
- Observability logs:
  - Log on every failed check: {component:"health", check:"db", error:"<short msg>", duration_ms:123}
  - Log level: warn for transient failures, error for persistent failures.

Feature flag kill-switch and rollback
- Kill-switch:
  - HEALTH_READINESS_DISABLED=true — skip checks and return OK for /readyz.
  - HEALTH_READINESS_FORCE_FAIL=true — force readiness to fail (useful for draining).
- Rollback plan:
  - Immediate: If rollout causes mass failures, set HEALTH_READINESS_DISABLED=true on the Deployment via kubectl set env or patch to avoid 503s while investigating.
  - Revert: Roll back Deployment to previous image tag and verify /readyz returns OK.
  - Complete rollback steps:
    1. kubectl set env deployment/<name> HEALTH_READINESS_DISABLED=true
    2. Monitor pods; if OK, roll back deployment: kubectl rollout undo deployment/<name>
    3. If unable to patch, scale down new replicas and scale up previous replicaset (kubectl rollout undo).
    4. Once stable, investigate logs and revert the change locally.

Manual smoke verification steps (post-deploy)
1. Start the service locally or deploy to a canary pod.
2. Check liveness:
   - curl -s -o /dev/null -w "%{http_code}\n" http://<host>:<port>/healthz
   - Expect 200
3. Check readiness (healthy DB):
   - curl -s -o /dev/null -w "%{http_code}\n" http://<host>:<port>/readyz
   - Expect 200
4. Simulate DB failure (stop DB or configure bad DSN), then:
   - curl -s -o /dev/null -w "%{http_code}\n" http://<host>:<port>/readyz
   - Expect 503
5. Graceful shutdown:
   - Send SIGTERM to the process and call /readyz — expect 503 quickly.
6. Metrics:
   - Query Prometheus for health_requests_total and health_duration_seconds.

Acceptance criteria
- Both endpoints exist and are registered in main.go.
- Liveness (/healthz) always responds quickly (200) when process is running.
- Readiness (/readyz) returns 200 only when DB check is passing and the service is ready; otherwise returns 503 with per-check details.
- Graceful shutdown flips readiness immediately.
- Unit tests for the above are present and pass (go test ./internal/health).
- README updated with usage and k8s probe examples.
- A simple CI smoke test runs and validates endpoints after build.

Location: .frunl/019942bc-60a4-73e8-a673-032840fec801/plan.md
