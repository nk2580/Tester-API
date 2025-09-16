# SPEC

Detailed Feature Description
This feature adds a single, deterministic unit test to the repository to increase confidence in the core HTTP handler behavior without introducing flakiness or external dependencies. The application is a small Go service (Gin + GORM + SQLite) providing two endpoints: POST /ping to create a Ping record and GET /pings to list them. The requested unit test will verify that the POST /ping handler correctly persists a Ping record to the database and returns the expected HTTP response when given a valid JSON payload.

Goals and value:
- Provide a minimal, fast, deterministic unit test that exercises the handler + persistence path.
- Ensure tests do not touch the on-disk sqlite file (db/data.db) to avoid side-effects and make them safe to run locally and in CI.
- Make a one-time, small refactor to main.go to expose a router setup function so the handler can be tested in isolation.
- Keep the change minimal: exactly one unit test file added, one small function extracted/introduced in main.go, and no modifications to CI workflows.

User Stories
- As a Developer, I want a unit test that exercises the POST /ping handler so that regressions to handler logic or persistence are detected early.
- As a Code Reviewer, I want the test to run deterministically and not touch disk so that test runs are reliable and reproducible on CI.
- As a Maintainer, I want the change to be small and reversible so that adding the test does not introduce maintenance burden.

Cucumber Scenarios

Scenario: POST /ping with valid JSON persists a Ping
Given an in-memory sqlite database and a Gin router wired to it
When I POST to /ping with JSON body {"message":"hello unit test"} and Content-Type: application/json
Then the HTTP response status is 200 OK
And the database contains exactly one Ping with Message == "hello unit test"

Scenario: POST /ping with invalid JSON returns 400
Given an in-memory sqlite database and a Gin router wired to it
When I POST to /ping with an invalid JSON body (e.g. "not json") and Content-Type: application/json
Then the HTTP response status is 400 Bad Request
And the database contains zero Ping records

Scenario: Test runs without touching the on-disk database
Given the repository contains db/data.db on disk
When the unit test runs using an in-memory sqlite DSN
Then db/data.db is not created or modified by the test run

# DESIGN

Technical Overview
- Minimal refactor: add a SetupRouter(db *gorm.DB) *gin.Engine function in main.go that registers POST /ping and GET /pings routes. main() will call SetupRouter(db) and run the server as before.
- Create a single unit test file main_test.go (package main) containing one test named TestPingHandler_PersistsPing that:
  - Opens an in-memory sqlite database via gorm (sqlite.Open("file::memory:?cache=shared")).
  - Runs AutoMigrate(&Ping{}) to ensure the Ping table exists in-memory.
  - Calls SetupRouter(db) to get a router with handlers bound to that DB.
  - Uses httptest to perform an HTTP POST /ping with a valid JSON body.
  - Asserts HTTP 200.
  - Queries the in-memory DB and asserts a single Ping exists with the expected message.

What must change
- main.go: add function SetupRouter(db *gorm.DB) *gin.Engine and move route registrations into it. main() should call SetupRouter(db) and then r.Run as before.
- Add new file: main_test.go with the single test (TestPingHandler_PersistsPing).
- No changes to go.mod or CI configuration are required.

Reusability
- Existing code (Ping struct, imports) will be reused.
- The test leverages the same handler code via SetupRouter to ensure parity with runtime behavior.

Component Breakdown
- main.go (modified)
  - Responsibilities:
    - Provide Ping struct (unchanged)
    - Provide SetupRouter(db *gorm.DB) *gin.Engine (new)
    - Keep main() as entrypoint: open disk DB, AutoMigrate, call SetupRouter and run the server
- main_test.go (new)
  - Responsibilities:
    - Create in-memory DB
    - AutoMigrate Ping schema
    - Use SetupRouter(db) to create router
    - Make HTTP request to /ping and assert behavior and DB state

ASCII Component Hierarchy
- main (package)
  - Ping (struct)
  - SetupRouter(db *gorm.DB) *gin.Engine  <-- new, registers handlers and returns router
  - main()                                <-- entrypoint, uses SetupRouter
  - main_test.go                          <-- new test, uses SetupRouter and in-memory DB

Data Flow Diagram (ASCII)
Client --> httptest.Request --> Gin Router (SetupRouter) --> POST /ping handler --> GORM (in-memory sqlite) --> Persist Ping
And:
POST /ping handler --> writes HTTP 200 response -> httptest.Recorder -> test assertions

Logic & Business Rules (mapped to SPEC scenarios)
- Rule: Valid JSON with "message" field should be accepted.
  - Maps to: Scenario "POST /ping with valid JSON persists a Ping"
  - Implementation: handler uses c.ShouldBindJSON(&ping); on success, db.Create(&ping) and returns 200.
- Rule: Malformed JSON should be rejected with HTTP 400 (Gin will return binding error).
  - Maps to: Scenario "POST /ping with invalid JSON returns 400"
  - Implementation: handler checks c.ShouldBindJSON(...) and returns StatusBadRequest.
- Rule: Unit test must not modify disk-based DB.
  - Maps to: Scenario "Test runs without touching the on-disk database"
  - Implementation: test uses sqlite in-memory DSN and AutoMigrate only against that in-memory DB.

API Endpoints (existing; documented precisely for the test)
- POST /ping
  - Method: POST
  - Path: /ping
  - Request JSON schema:
    - { "message": "string" } (message required by handler binding; empty string is accepted by existing handler)
  - Success Response:
    - Status: 200 OK
    - Body: { "message": "Ping registered successfully!" }
  - Error Responses:
    - 400 Bad Request (binding error)
    - 500 Internal Server Error (when db.Create fails)
- GET /pings
  - Method: GET
  - Path: /pings
  - Response: 200 OK with JSON array of Ping objects

Database Schema Changes
- None to repository schema or migrations.
- The test will call db.AutoMigrate(&Ping{}) against an in-memory DB to ensure the schema is available for the test.

Non-Functional Requirements
- Determinism: test must be deterministic; no external network or disk writes.
- Speed: test should be fast (<200ms typical on modern CI).
- Isolation: test uses in-memory DB and leaves no artifacts on disk.
- Observability: test logs should be minimal; failures must print informative errors.
- Rollout: This is a development-only change (a unit test). If the test causes CI failures, revert/modify the single commit that added the test.
- Feature flag / kill-switch: Not applicable for runtime features. For CI breakages, the test can be disabled or removed; we provide a rollback plan.

Assumptions & Open Questions
- Assumption: The repo uses Go tooling; go test ./... is available in CI.
- Assumption: It's acceptable to slightly refactor main.go to add SetupRouter.
- Open question: If maintainers prefer not to modify main.go, an alternative test could re-create identical route handlers in the test; this is less preferred because it duplicates logic.
- Open question: Whether an empty message should be permitted — current handler allows it; the test uses a non-empty message.

# TASKS

- [ ] **Task 1.1: Minimal refactor — add SetupRouter(db *gorm.DB) and wire main() to use it**
- **Status:** Pending
- **Context:**
 - Modify main.go to introduce a function: func SetupRouter(db *gorm.DB) *gin.Engine
 - Move the route registration (POST /ping and GET /pings) into SetupRouter so tests can instantiate the router with an injected DB.
 - Keep main() behavior unchanged other than calling SetupRouter(db) and calling r.Run(":8080").
- **Dependencies:** None
- **Validation:**
 - [ ] main.go contains the function signature: "func SetupRouter(db *gorm.DB) *gin.Engine"
 - [ ] main.go still calls SetupRouter(db) in main() (i.e., "r := SetupRouter(db)" exists)
 - [ ] No other behavioral changes to route registration (POST /ping and GET /pings present)
 - [ ] Code compiles: run `go build ./...` locally (expected: no compile errors)

- [ ] **Task 1.2: Add a single unit test file main_test.go with TestPingHandler_PersistsPing**
- **Status:** Pending
- **Context:**
 - Create file main_test.go in repository root (package main).
 - Implement one test: TestPingHandler_PersistsPing which:
   - Opens an in-memory sqlite DB: gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
   - Runs db.AutoMigrate(&Ping{})
   - Calls router := SetupRouter(db)
   - Issues a POST /ping with {"message":"hello unit test"} using httptest.NewRecorder and httptest.NewRequest
   - Asserts HTTP 200
   - Queries the in-memory DB to assert exactly one Ping exists and its Message equals the sent value
 - Use only stdlib testing + httptest, encoding/json, bytes and existing gorm/sqlite imports.
- **Dependencies:** Task 1.1
- **Validation:**
 - [ ] File path: ./main_test.go exists
 - [ ] main_test.go contains the test function: "func TestPingHandler_PersistsPing(t *testing.T)"
 - [ ] The test uses an in-memory DSN (e.g., "file::memory:?cache=shared") — confirm the string appears in the test
 - [ ] Running `go test -run TestPingHandler_PersistsPing -v ./...` returns exit code 0 and shows the test passing
 - [ ] Running `go test ./...` returns exit code 0 (no unintended failures caused)
 - [ ] Confirm db/data.db on disk (if present) is not modified by the test run (validate by file modtime or absence of new files)

- [ ] **Task 1.3: Verify, document, and provide rollback instructions**
- **Status:** Pending
- **Context:**
 - Run the repository test suite locally or in CI: `go test ./...`
 - Add a one-line note to README.md (optional) to document how to run the single test locally: `go test -run TestPingHandler_PersistsPing -v ./...` — if maintainers prefer not to edit README.md, record this in PR description.
 - Prepare a brief rollback plan and an explanation in PR notes describing why SetupRouter was added and that it is safe.
- **Dependencies:** Task 1.2
- **Validation:**
 - [ ] `go test ./...` succeeds and the new test is listed as passing
 - [ ] README.md or the PR description contains a short instruction line for running the single test (or a TODO to add it to docs)
 - [ ] A rollback plan exists in the PR notes: remove main_test.go and the SetupRouter change in main.go (confirm steps in PR description)
 - [ ] Observability: test output is concise and logs failures with actionable messages (manual review of test failure messages)

# VERIFICATION

Artifact Check
- Confirm this plan file exists at: .frunl/019945d5-cd29-730a-a132-c3cf71e1cdad/plan.md

Traceability Table (Scenario → Design Element(s) → Task ID(s))
- "POST /ping with valid JSON persists a Ping"
  - Design Elements: SetupRouter(db *gorm.DB), POST /ping handler, in-memory DB AutoMigrate, main_test.go -> TestPingHandler_PersistsPing
  - Task ID(s): Task 1.1, Task 1.2
- "POST /ping with invalid JSON returns 400"
  - Design Elements: POST /ping handler binding error path (c.ShouldBindJSON), test could be extended but is not required for single test
  - Task ID(s): Task 1.1 (handler kept), Task 1.2 (test focuses on valid path; invalid path is documented)
- "Test runs without touching the on-disk database"
  - Design Elements: in-memory sqlite DSN in test, AutoMigrate as in-memory only
  - Task ID(s): Task 1.2, Task 1.3

Test Strategy (high-level unit test(s) only)
- Single focused unit test:
  - TestPingHandler_PersistsPing (main_test.go)
    - Core path: POST /ping with valid JSON => 200 and persisted DB record
    - Critical edge case: uses in-memory DB to avoid disk writes
- Example commands:
  - Run only the new test: go test -run TestPingHandler_PersistsPing -v ./...
  - Run entire test suite: go test ./...
- Expected outputs:
  - The single test passes and reports PASS
  - No other tests are expected; if other tests exist, they should not be affected
- Notes:
  - Keep tests fast and deterministic (no network, no disk writes)

Operational Readiness
- Metrics/logs/traces:
  - No runtime metric changes. For test failures, the test's t.Fatalf messages should describe the failure (HTTP status, response body, DB query result).
- Feature flag / kill-switch:
  - If the test causes CI failures, maintainers can:
    1. Temporarily disable the test by renaming the file to main_test.go.disabled (or comment out the test), or
    2. Revert the commit that added main_test.go and the SetupRouter modification.
  - A safer pattern is to gate the test behind a build tag (not necessary here) if you want to temporarily exclude it from CI; e.g., `// +build unit` header and run `go test -tags=unit`.
- Rollback plan:
  - Revert the commit that introduces main_test.go and the SetupRouter function in main.go. The revert will restore the repository to its previous state; this is a low-risk change because it only adds a test and a tiny refactor.
- Observability during rollout:
  - After merging, observe CI for any unexpected failures.
  - If CI fails due to the test, examine failure output; if test is flaky, temporarily revert and open an investigation ticket.

End of plan.

Location: .frunl/019945d5-cd29-730a-a132-c3cf71e1cdad/plan.md
