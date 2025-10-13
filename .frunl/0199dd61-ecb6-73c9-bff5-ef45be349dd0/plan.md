# SPEC

Detailed Feature Description
This feature introduces a reproducible, ephemeral development and CI environment for the API using Docker Compose. The goal is to provide a lightweight, easy-to-run environment that mimics the application's runtime (the API service backed by a persistent DB volume) for non-production workflows (local development, pull request CI jobs, and short-lived integration/smoke test runs). The Compose setup is explicitly an analogy to the Kubernetes production deployment: it does not attempt to be a drop-in Kubernetes manifest but provides parity for the API behavior (binding address, environment configuration, DB persistence) so developers and CI can exercise the same code paths that will run in production.

Key qualities:
- Local-first: a developer can run the API inside a container with a single script and get a deterministic environment that does not require installing Go locally.
- CI-friendly: a script that CI jobs can call to build the image, bring up an ephemeral environment, run smoke tests, and tear down — without requiring changes to existing workflow files in this plan (we provide the script and instructions; integrating the script into CI remains a one-line change to CI workflows by repository maintainers).
- Minimal and safe: the default Compose profile uses the application's existing sqlite configuration (no new external services required) and persists DB to a named volume so each ephemeral environment is isolated.
- Configurable: the application is made configurable through environment variables (PORT, DATA_DB_PATH, EPHEMERAL_READY_FILE) to allow compose-based orchestration and health checks.
- Non-invasive: no changes to production/deployment manifests are required. The plan avoids touching .github/workflows (per constraints) and keeps the ephemeral environment opt-in.

Why this exists and value:
- Eliminates “works on my machine” problems by providing reproducible containers with the same binary and DB initialization logic.
- Enables CI to run integration/smoke tests in an environment close to production without requiring Kubernetes.
- Speeds onboarding: new contributors can run the API without installing Go or managing toolchains.
- Simplifies PR validation: reviewers can rely on CI-run smoke tests against the same compose stack.

Scope and limitations
- This feature targets local development and CI ephemeral environments only; it does not create or modify Kubernetes manifests for production.
- Default implementation uses SQLite (the repo's current database) to minimize code change surface. An optional Postgres profile is noted as an extension (requires code changes and configuration).
- The plan contains only repository file changes (Dockerfile, docker-compose files, scripts, docs, minor code tweaks). It does not (and must not) modify GitHub Actions workflow files in this plan.

User Stories

Feature: Local ephemeral API environment
- As a developer, I want to run the API in a containerized ephemeral environment using Docker Compose so that I can test behavior without installing Go or managing dev dependencies.
- As a reviewer, I want CI to start an ephemeral environment for each PR so that automated smoke tests validate the API before merging.
- As a CI operator, I want a single script I can call from CI to build, bring up, test, and tear down the ephemeral environment so that CI jobs remain simple and repeatable.
- As an ops engineer, I want the ephemeral environment to provide health/readiness signals so that automated checks can decide when the API is ready for tests.

Cucumber Scenarios

Scenario: Developer starts local ephemeral environment
Given I have the repository checked out locally
And Docker Engine and Docker Compose are installed on my machine
When I run the provided scripts/start-local.sh
Then a docker image for the API is built
And a container named "api" is started and bound to port 8080 on the host
And the API responds 200 to GET /healthz
And the local db directory or named volume is created for development use

Scenario: CI job runs ephemeral environment and executes smoke tests
Given CI runner has Docker and Docker Compose available
And the repository contains scripts/ci-smoke.sh
When CI invokes scripts/ci-smoke.sh
Then docker-compose builds the API image, starts the compose stack in detached mode
And the script waits until the /healthz returns 200
And the script makes a POST /ping with JSON {"message":"ci-smoke"} and receives 200
And the script makes a GET /pings and sees the "ci-smoke" entry
And the script tears down the compose stack and exits with code 0

Scenario: Health and readiness checks used by orchestration
Given the API binary writes a readiness marker file once DB migrations succeed
When the container runs
Then docker-compose healthcheck succeeds only after the readiness marker file exists
And GET /healthz returns success only after the marker is present

Scenario: Feature toggle disables ephemeral environment in CI
Given maintainers want to skip ephemeral environment in a particular CI job
When the environment variable EPHEMERAL_DISABLED=true is set in CI
Then scripts/ci-smoke.sh exits early and reports "ephemeral environment disabled" as a success condition

# DESIGN

Technical Overview
We will:
1. Make minimal code changes to the Go application so it can be configured by environment variables and report readiness:
   - Read PORT and DATA_DB_PATH from environment variables (with sensible defaults).
   - Ensure the directory for the configured DB path exists (mkdir -p).
   - Add /healthz and /ready HTTP endpoints.
   - Create a readiness marker file (path configurable via EPHEMERAL_READY_FILE) once AutoMigrate completes successfully so that compose healthchecks can rely on a file test (avoids requiring curl/wget in the runtime image).
2. Add Docker artifacts and orchestration files:
   - Dockerfile (multi-stage) to build the Go binary and produce a small runtime image.
   - .dockerignore to keep images small.
   - docker-compose.yml for local development (bind-mount ./db to container path to make DB visible to developer).
   - docker-compose.ci.yml for CI (named volume for DB to keep CI ephemeral).
3. Add orchestration scripts:
   - scripts/start-local.sh — builds and runs the local compose stack (with bind mount).
   - scripts/ci-smoke.sh — run in CI: builds, starts CI compose stack, waits for readiness, runs smoke tests (curl-based), then tears down.
   - scripts/wait-for.sh — small portable shell loop to poll health endpoint (POSIX sh).
4. Add documentation (docs/ephemeral.md) and README.md snippets describing how to use the ephemeral environment, CI integration instructions, rollback, and kill-switch environment variables.
All modifications are made by editing repository files (apply_patch) and will not require the assistant to run external commands.

Component Breakdown (new/modified files)
- main.go (modified)
  - Responsibilities: read env config, ensure DB path directory exists, AutoMigrate, create readiness marker file, register /healthz and /ready endpoints, run the server on configured PORT.
- config.go (new)
  - Responsibilities: small helper to read env vars with defaults and exported config struct used by main.go. (Optionally embed sanity-check functions.)
- Dockerfile (new)
  - Multi-stage Go build and runtime image. Exposes PORT and declares a default DATA_DB_PATH and readiness file path environment variable.
- .dockerignore (new)
  - Exclude db/data.db, .git, .frunl, and other developer artifacts.
- docker-compose.yml (new)
  - Local development compose file:
    - service: api built from local Dockerfile
    - volumes: ./db:/data (so the sqlite file is preserved on the host)
    - environment: PORT, DATA_DB_PATH, EPHEMERAL_READY_FILE
    - healthcheck: test for readiness marker file (no curl dependency)
- docker-compose.ci.yml (new)
  - CI compose file:
    - same service but uses a named volume db-data:/data, no host bind mount
    - keep healthcheck and env variables
- scripts/start-local.sh (new)
  - Build and run docker-compose.yml (dev usage instructions)
- scripts/ci-smoke.sh (new)
  - Build and run docker-compose.ci.yml, wait for readiness, run curl smoke tests for /healthz, POST /ping and GET /pings, tear down stack; exit non-zero on failure
- scripts/wait-for.sh (new)
  - Small POSIX shell helper to poll HTTP endpoint with timeout (uses /bin/sh and /bin/grep/curl as available in CI)
- docs/ephemeral.md (new)
  - Tutorial, optional CI integration snippet, roll-back instructions, and observability checklist
- README.md (modified)
  - Add a short section "Ephemeral local environment (Docker Compose)"

UI/UX Wireframes (CLI oriented)
Local developer flow (ASCII)
$ ./scripts/start-local.sh
[build] Building api image...
[compose] Starting api container...
[wait] Waiting for API readiness at http://localhost:8080/healthz
[ok] API ready; mapped to localhost:8080
[info] To stop: docker compose down

Component hierarchy (ASCII)
repo/
  - main.go (server)
  - config.go (env parsing)
  - Dockerfile (build)
  - docker-compose.yml (local)
  - docker-compose.ci.yml (ci)
  - scripts/
    - start-local.sh
    - ci-smoke.sh
    - wait-for.sh
  - docs/
    - ephemeral.md

Data Flow Diagrams (ASCII)

Developer machine
   |
   | docker-compose (local) builds image from Dockerfile
   v
api container (binary runs)
   |-- reads ENV: DATA_DB_PATH -> /data/data.db (mounted or volume)
   |-- writes readiness file -> /tmp/ephemeral-ready (configured)
   |-- serves HTTP on PORT (8080)
   v
db file persisted in bind-mount ./db or named volume (docker)

Logic & Business Rules (mapping to SPEC scenarios)
- Rule: API must not report ready until DB open and migrations succeed.
  - Implementation: After db.Open and AutoMigrate succeed, create readiness marker file and return 200 on /ready and /healthz.
  - Scenarios: Developer starts local ephemeral; CI job runs ephemeral environment.
- Rule: Default configuration should preserve current behavior if no env vars set.
  - Implementation: Defaults: PORT=8080, DATA_DB_PATH=db/data.db (same as current), EPHEMERAL_READY_FILE=/tmp/ephemeral-ready.
  - Scenarios: Local developer using start-local.sh with no env changes.
- Rule: Compose healthchecks must succeed by checking a file (avoid curl/wget in the runtime image).
  - Implementation: Health marker file created by server; compose healthcheck uses test -f on that file.
  - Scenarios: Health and readiness checks used by orchestration.
- Rule: CI can skip ephemeral environment via EPHEMERAL_DISABLED=true.
  - Implementation: scripts/ci-smoke.sh checks EPHEMERAL_DISABLED and exits early.
  - Scenarios: Feature toggle disables ephemeral environment in CI.

API Endpoints (new/modified)
- Existing endpoints remain unchanged:
  - POST /ping (same request/response)
  - GET /pings
- New endpoints:
  - GET /healthz
    - Success: 200, body {"status":"ok"} when the app is running (basic health).
    - Failure: 500 or non-200 when critical internal components (DB) are not ready.
  - GET /ready
    - Success: 200 when readiness file exists (post migrations).
    - Failure: 503 if not ready.
- No changes to request/response schema for /ping endpoints.

Database Schema Changes / Migrations
- No schema changes required. The existing GORM AutoMigrate call will continue to run and create the Ping table on first start.
- Implementation ensures the DB path directory exists before opening the database.

Non-Functional Requirements
- Security:
  - Do not commit secrets or credentials. Compose files should not store sensitive env vars in plaintext; prefer CI secrets or environment configuration in the CI system.
  - The runtime image runs as non-root where feasible (optional extension; initial MVP may run as root for simplicity).
- Performance:
  - Compose-based ephemeral environment is not optimized for high load; it's for smoke/integration testing. No heavy performance tuning is required.
- Observability:
  - Application logs write to stdout/stderr (gin.Default does this). CI collects container logs.
  - Health and readiness endpoints added for orchestration and automated checks.
- Feature flags & rollout:
  - EPHEMERAL_DISABLED environment variable can disable CI orchestration scripts.
  - The compose artifacts are opt-in; no automatic changes to CI workflows are performed by this plan.
- Rollout strategy:
  - Keep changes small and isolated: configuration defaults to existing behavior.
  - Merge code+compose+scripts behind tests and an agreed-on CI invocation.
  - If issues occur, revert the PR or disable CI invocation with EPHEMERAL_DISABLED.

Assumptions & Open Questions
- Assumptions:
  - CI runners will have Docker and docker-compose (or docker compose plugin) available; repository maintainers will add a single command in CI to invoke scripts/ci-smoke.sh.
  - Production uses Kubernetes, but we do not attempt to simulate cluster behavior beyond the API runtime and DB.
  - Developers and CI are allowed to pull Docker base images from the internet; builds will require network.
- Open Questions:
  - Should we provide an optional Postgres compose profile to mimic production databases? (If yes, additional code changes are required: support gorm Postgres driver and migration config.)
  - Do we need to run full end-to-end tests that require external services? (Out of scope; ephemeral env is for API only.)
  - Do we want the runtime image to run as non-root for security? (optional improvement)

# TASKS

- [ ] **Task 1.1: Make the app configuration-driven and add health/readiness behavior**
- **Status:** Pending
- **Context:**
 - Modify main.go to:
   - Read PORT, DATA_DB_PATH, EPHEMERAL_READY_FILE, and EPHEMERAL_DISABLED from environment variables with sensible defaults (PORT=8080, DATA_DB_PATH=db/data.db, EPHEMERAL_READY_FILE=/tmp/ephemeral-ready).
   - Ensure the directory containing DATA_DB_PATH exists before opening SQLite (os.MkdirAll).
   - After successful AutoMigrate, create the EPHEMERAL_READY_FILE to signal readiness.
   - Add endpoints GET /healthz (returns 200 when server running and DB accessible) and GET /ready (200 only if readiness marker exists, otherwise 503).
 - Add a small helper file config.go (or inline helpers) to centralize env parsing and defaults.
 - Add lightweight unit tests (config_test.go and main_init_test.go) that:
   - Validate config defaulting behavior.
   - Validate that the DB initialization path calls mkdir and can use in-memory SQLite for tests (sqlite.Open("file::memory:?cache=shared")).
- **Dependencies:** None
- **Validation:**
 - [ ] main.go contains code to read env vars (os.Getenv/os.LookupEnv) and uses them for sqlite.Open and r.Run.
 - [ ] config.go exists and exposes a Config struct with fields Port, DataDBPath, ReadyFile, EphemeralDisabled.
 - [ ] Unit tests exist: config_test.go and main_init_test.go in the repo.
 - [ ] Unit tests exercise config parsing and DB init using in-memory sqlite, and are runnable via `go test ./...` (high-level); include expected asserts in test files.

- [ ] **Task 1.2: Add Dockerfile and .dockerignore; create compose files for local and CI profiles**
- **Status:** Pending
- **Context:**
 - Add Dockerfile (multi-stage):
   - builder stage: golang builder that downloads modules and builds a static binary (CGO_ENABLED=0).
   - final stage: small base (alpine) copying the binary, exposing PORT, and setting default ENV values for DATA_DB_PATH and EPHEMERAL_READY_FILE.
   - Dockerfile should document that building requires network (go modules).
 - Add .dockerignore to exclude db/data.db, .git, .frunl, and other transient files.
 - Add docker-compose.yml (local):
   - service: api
   - build: .
   - volumes: ./db:/data
   - environment mapping for PORT, DATA_DB_PATH=/data/data.db, EPHEMERAL_READY_FILE=/tmp/ephemeral-ready
   - healthcheck: test -f /tmp/ephemeral-ready (use CMD-SHELL: test -f /tmp/ephemeral-ready || exit 1)
 - Add docker-compose.ci.yml (CI):
   - same service but uses a named volume db-data:/data (no host bind mount)
   - declares volumes: db-data
 - Note: Do NOT modify .github/workflows in this task.
- **Dependencies:** Task 1.1 (readiness marker file must be created by the app for compose healthchecks)
- **Validation:**
 - [ ] Files exist: Dockerfile and .dockerignore, docker-compose.yml, docker-compose.ci.yml.
 - [ ] Dockerfile uses a multi-stage build with a builder and a small runtime stage, exposes the default PORT environment.
 - [ ] docker-compose.yml contains a healthcheck which tests the existence of EPHEMERAL_READY_FILE (/tmp/ephemeral-ready) rather than invoking curl/wget.
 - [ ] docker-compose.ci.yml defines a named volume db-data and no host bind mount.

- [ ] **Task 1.3: Add orchestration scripts and smoke-test script for CI**
- **Status:** Pending
- **Context:**
 - scripts/wait-for.sh: small portable loop (POSIX sh) that polls HTTP /healthz until ready or times out (used by both local and CI scripts).
 - scripts/start-local.sh:
   - Runs `docker compose -f docker-compose.yml up --build -d`
   - Calls wait-for.sh to poll http://localhost:${PORT:-8080}/healthz
   - Prints instructions to tail logs or shut down
 - scripts/ci-smoke.sh:
   - If EPHEMERAL_DISABLED=true, print message and exit 0
   - Runs `docker compose -f docker-compose.ci.yml up --build -d`
   - Uses wait-for.sh against container host port (or uses `docker compose exec` to access service) to wait for ready
   - Executes smoke tests using curl (POST /ping {"message":"ci-smoke"}; GET /pings and grep for ci-smoke)
   - On success, runs `docker compose -f docker-compose.ci.yml down -v` and exits 0; on failure, prints logs and exits non-zero
 - Scripts must be POSIX-sh friendly (start-local.sh and ci-smoke.sh with #!/usr/bin/env sh).
- **Dependencies:** Task 1.2
- **Validation:**
 - [ ] scripts/start-local.sh, scripts/ci-smoke.sh, scripts/wait-for.sh exist with executable shebangs and clear usage comments.
 - [ ] scripts/ci-smoke.sh implements EPHEMERAL_DISABLED guard.
 - [ ] scripts/ci-smoke.sh contains the curl-based smoke tests (POST /ping and GET /pings) and returns non-zero on failed HTTP responses.
 - [ ] README and docs reference invoking ./scripts/start-local.sh and scripts/ci-smoke.sh.

- [ ] **Task 1.4: Documentation, observability checklist, and rollout/runbook**
- **Status:** Pending
- **Context:**
 - Add docs/ephemeral.md:
   - Quickstart for local dev (`./scripts/start-local.sh`).
   - CI integration snippet (example: single job step to checkout code and run ./scripts/ci-smoke.sh).
   - Rollback and kill-switch instructions (how to set EPHEMERAL_DISABLED).
   - Observability checklist (what to watch in logs, health endpoints, container restarts, named volume health).
 - Update README.md:
   - Short section "Ephemeral local environment (Docker Compose)" with minimal commands.
 - Add notes for maintainers on how to optionally extend to Postgres and how to run the compose stack in a non-networked environment (offline caching of images) — keep as TODOs not implemented.
- **Dependencies:** Tasks 1.1, 1.2, 1.3
- **Validation:**
 - [ ] docs/ephemeral.md exists and contains quickstart examples and CI integration snippet.
 - [ ] README.md contains a "Ephemeral local environment" section with a one-liner to start local environment and mention EPHEMERAL_DISABLED.
 - [ ] The runbook contains a clear rollback path: `docker compose -f docker-compose.ci.yml down -v` and a description of disabling ephemeral env via EPHEMERAL_DISABLED=true in CI.

Notes about task count and grouping
- Tasks are intentionally grouped to keep total task count ≤ 4. Each task bundles code, artifacts, tests, observability, and documentation changes as contextual validation checks so all necessary cross-cutting concerns are covered.

# VERIFICATION

Artifact Check
- Confirm this plan file exists at:
  - .frunl/0199dd61-ecb6-73c9-bff5-ef45be349dd0/plan.md

Traceability Table (Scenario → Design Element(s) → Task ID(s))

- "Developer starts local ephemeral environment"
  - Design elements: docker-compose.yml (local), Dockerfile, scripts/start-local.sh, config changes in main.go (PORT/DATA_DB_PATH)
  - Task IDs: 1.1, 1.2, 1.3, 1.4

- "CI job runs ephemeral environment and executes smoke tests"
  - Design elements: docker-compose.ci.yml (CI), scripts/ci-smoke.sh, readiness marker in main.go, smoke tests
  - Task IDs: 1.1, 1.2, 1.3, 1.4

- "Health and readiness checks used by orchestration"
  - Design elements: /healthz and /ready endpoints, readiness file creation, compose healthcheck on /tmp/ephemeral-ready
  - Task IDs: 1.1, 1.2

- "Feature toggle disables ephemeral environment in CI"
  - Design elements: scripts/ci-smoke.sh check EPHEMERAL_DISABLED, documentation
  - Task IDs: 1.3, 1.4

Test Strategy (high-level unit tests only)
- Scope: small set of unit tests to validate the most important new behaviors. These tests are "high-level" unit tests (fast, no network) and avoid heavy integration dependencies:
  - TestConfigDefaults (config_test.go)
    - Arrange: no env vars set
    - Act: load config
    - Assert: config.Port == "8080", config.DataDBPath == "db/data.db", config.ReadyFile == "/tmp/ephemeral-ready"
  - TestConfigEnvOverrides (config_test.go)
    - Set PORT and DATA_DB_PATH env vars, load config, assert values match
  - TestDBInitUsingInMemorySQLite (main_init_test.go)
    - Arrange: create a Config pointing to "file::memory:?cache=shared" and call initialization helper that opens DB and AutoMigrates
    - Assert: AutoMigrate returns nil and a ping record can be inserted/retrieved using GORM
  - TestReadinessFileCreatedOnSuccessfulInit (main_init_test.go)
    - Arrange: use a temporary directory for ready file, call initialization, assert ready file path exists after init
- Example commands for the integrator (run locally or in CI):
  - go test ./...
  - docker compose -f docker-compose.yml up --build -d  # local (manual)
  - ./scripts/ci-smoke.sh  # CI or local smoke-run (needs docker)

Notes about executing tests and builds (constraints)
- This plan intentionally limits changes to repository files. The implementer must apply file changes using repository editing tools (apply_patch) and then run builds/tests locally or in CI. Because the assistant environment is offline and does not have tooling installed, no runtime commands are executed by this plan; the human operator or CI runner will run `go test`, `docker compose build`, or `./scripts/ci-smoke.sh`.
- Building Docker images requires network access to fetch base images and modules; ensure CI runner has network access or caches images in advance.

Operational Readiness (metrics, logs, traces, rollback)
- Logs:
  - Application must log startup parameters (PORT, DATA_DB_PATH) and log migration success/failure to stdout/stderr. Standard container log collection is sufficient.
- Health & readiness:
  - /healthz: used by scripts and for quick checks.
  - /ready: reflects creation of the readiness marker file; docker-compose healthcheck uses file presence.
- Metrics/traces:
  - Not part of the MVP. Recommend adding Prometheus metrics and an /metrics endpoint in a follow-up if required.
- Feature flag kill-switch:
  - EPHEMERAL_DISABLED=true causes scripts/ci-smoke.sh to exit early (success), so maintainers can disable ephemeral runs without changing workflows.
- Rollback plan:
  - If the ephemeral environment causes CI instability, remove or disable the CI step that calls scripts/ci-smoke.sh and revert any faulty commit. Locally, run:
    - docker compose -f docker-compose.ci.yml down -v
    - docker image rm <image> (if needed)
- Observability checklist for a failing ephemeral run:
  - Check container logs: docker compose logs api
  - Check health endpoint: curl -sS localhost:8080/healthz
  - Check readiness marker file exists inside container: docker compose exec api ls -l /tmp/ephemeral-ready
  - Collect docker compose ps output and include with issue report

End of plan.

Location: .frunl/0199dd61-ecb6-73c9-bff5-ef45be349dd0/plan.md
