# SPEC

Detailed Feature Description
This feature adds a first-class suite of unit tests and a small refactor to make core business logic testable without requiring the Gin or GORM runtime in unit tests. The repository currently contains a minimal Go HTTP service (main.go) that uses gin and gorm to provide two endpoints: POST /ping and GET /pings. The goal is to introduce a clean separation between business logic and framework/database implementation so that:

- Unit tests can exercise core logic (validation, persistence contract calls, error mapping) deterministically and fast using an in-memory implementation of storage.
- Tests do not require network access, external services, or runtime DB access; they should use only standard-library features (sync, testing, net/http/httptest) plus repository-local code.
- Production behavior is preserved: at runtime the existing GORM-backed store is used and AutoMigrate remains in place.
- The changes are limited to repository edits only (no CI workflow or external system changes) and include documentation describing how to run tests locally or in CI when Go toolchain is available.

Value:
- Immediate developer confidence: catches regressions in core behaviour (validation and storage contract).
- Enables safe refactors: business rules are decoupled from framework code.
- Faster feedback loop: in-memory tests are quick and deterministic.
- Clear path for later CI integration (no workflow edits required here).

Sub-features
1. Storage abstraction and in-memory store
   - Introduce a Store interface describing the persistence contract (SavePing, GetPings).
   - Provide an in-memory, concurrency-safe MemoryStore implementation for use by unit tests.
   - Provide a GormStore adapter for production that wraps the existing gorm DB calls (keeps same behavior).
2. Business logic layer
   - Add small, framework-agnostic functions (CreatePing, ListPings) that implement validation and call the Store interface. These functions are the primary test targets.
3. Tests
   - Unit tests targeting the business logic with MemoryStore, covering success, validation error, storage errors, and list semantics.
   - Tests are pure Go tests (testing package) and do not require gin/gorm during test runtime.
4. Documentation and rollout notes
   - Document how to run tests locally and guidance for CI integration without modifying repository workflows.

User Stories
- As a developer, I want a Store interface and an in-memory implementation, so that I can run tests without a real database.
- As a maintainer, I want business logic functions separated from Gin/GORM, so that I can write unit tests with standard-library tooling that are fast and deterministic.
- As a reviewer, I want unit tests that cover success, validation failure, and storage error cases, so that regressions are visible in PRs.
- As an operator, I want instructions and a kill-switch to rollback or disable in-memory behavior, so I can safely roll out changes.

Cucumber Scenarios
Scenario: Create ping successfully using business API
  Given an empty in-memory store
  And a valid ping payload with message "hello, world"
  When the CreatePing business function is invoked with that payload
  Then the function returns no error
  And the store contains exactly one Ping whose Message == "hello, world"

Scenario: Create ping fails when message is empty
  Given an empty in-memory store
  And a ping payload with message ""
  When the CreatePing business function is invoked with that payload
  Then the function returns a validation error (client-side error)
  And the store remains empty

Scenario: List pings returns existing records
  Given an in-memory store seeded with two pings ("a", "b")
  When the ListPings business function is invoked
  Then the function returns an array with two pings matching the seeded messages

Scenario: Create ping surfaces storage errors as internal errors
  Given a store implementation that always returns an error on Save
  When the CreatePing business function is invoked with a valid payload
  Then the function returns an error indicating an internal storage failure
  And the store did not persist the ping

# DESIGN

Technical Overview
To allow unit testing without framework/database dependencies, add a small abstraction layer and refactor main to use it:

- Introduce an internal "store" package (internal/store) that declares:
  - type Store interface { SavePing(*Ping) error; GetPings() ([]Ping, error) }
  - MemoryStore: in-memory concurrent-safe implementation used by unit tests
  - GormStore: adapter that wraps *gorm.DB and uses the same behavior as the current code

- Introduce an internal "api" package (internal/api) with framework-agnostic functions:
  - func CreatePing(s store.Store, p *store.Ping) error
  - func ListPings(s store.Store) ([]store.Ping, error)
  These functions perform validation and call the store.

- Modify main.go to:
  - Move the existing Ping struct to internal/store/ping.go (so both production and tests share the same model).
  - Create a GormStore backed by the current gorm.Open(...) and pass it to handlers.
  - Keep Gin handlers but make them thin adapters that call internal/api functions and map errors to HTTP codes.

- Tests will import only the internal/api and internal/store (memory store) packages and the standard library testing package — avoiding gin and gorm in tests.

Component Breakdown (files to add/modify)
Note: exact file paths are proposed; implementers will edit repository files using patches.

- internal/store/ping.go
  - Responsibility: define Ping struct with json and gorm tags (shared model).

- internal/store/store.go
  - Responsibility: declare Store interface and shared error types (e.g., ErrValidation).

- internal/store/mem_store.go
  - Responsibility: provide MemoryStore implementing Store, concurrency-safe (mutex), deterministic behavior, optional hooks for error injection to simulate storage failures in tests.

- internal/store/gorm_store.go
  - Responsibility: provide GormStore that wraps *gorm.DB and implements Store; uses the existing db.Create & db.Find semantics.

- internal/api/api.go
  - Responsibility: contain CreatePing and ListPings functions containing validation and logging; return descriptive errors (typed where useful) so tests can assert behavior.

- internal/api/api_test.go
  - Responsibility: unit tests for CreatePing and ListPings using MemoryStore. Tests should cover: success, empty message validation, storage error injection, list behavior.

- main.go (modified)
  - Responsibility: wire up GormStore and Gin routes; handlers adapt to api functions and translate errors to HTTP status codes. Keep AutoMigrate call.

UI/UX Wireframes
- Not applicable (API-only feature). No UI wireframes required.

ASCII component hierarchy
- Client
  - -> Gin router (main.go)
    - -> Handler adapters (map request -> internal/api)
      - -> internal/api (business functions)
        - -> internal/store.Store (interface)
          - -> internal/store.GormStore (production)
          - -> internal/store.MemoryStore (tests)

Data Flow Diagram (ASCII)
Client POST /ping
  |
  v
Gin handler (main.go) --- parses JSON ---> internal/api.CreatePing
                                              |
                                              v
                                      store.Store.SavePing(p)
                                            /      \
                                           /        \
                                  GormStore (SQLite)  MemoryStore (tests)

Logic & Business Rules (mapping to SPEC scenarios)
- Rule: Ping.Message must be a non-empty string (trim whitespace). If empty -> validation error mapped to HTTP 400. (Scenarios: Create ping fails when message is empty)
- Rule: CreatePing must call Store.SavePing exactly once and return nil on success. (Scenario: Create ping successfully)
- Rule: On store.SavePing returning an error, CreatePing returns an internal error (mapped to HTTP 500 by handlers). (Scenario: Create ping surfaces storage errors)
- Rule: ListPings returns the full list from Store.GetPings; if none, return empty array (HTTP 200). (Scenario: List pings returns existing records)

API Endpoints (no change to public surface)
- POST /ping
  - Request JSON: { "message": "string" }
  - Success: 200 OK, body: { "message": "Ping registered successfully!" } (preserve current behavior)
  - Client errors: 400 Bad Request with JSON { "error": "validation: message required" } for validation failures.
  - Server errors: 500 Internal Server Error with JSON { "error": "Failed to save ping" } for persistence errors.

- GET /pings
  - Success: 200 OK, body: [{ "id": uint, "message": "string" }, ...]
  - Server errors: 500 Internal Server Error with JSON { "error": "Failed to retrieve pings" }

Request/Response schemas
- Ping (shared model)
  - id: uint (may be 0 for not-yet-created items)
  - message: string (required, non-empty after trim)

Database Schema Changes / Migrations
- No schema changes necessary. The existing Ping model is preserved; AutoMigrate remains in main.go using the same Ping model. The migration step remains unchanged.

Non-Functional Requirements
- Security: Validate user input (trim+non-empty). No changes to authentication (none present).
- Performance: MemoryStore tests should be low overhead. GormStore retains same performance characteristics.
- Observability: Add structured log messages at points: validation failures, SavePing success/failure, GetPings success/failure. Keep logs minimal and safe for production.
- Feature flags: Add a simple env var toggle (e.g., USE_IN_MEMORY_STORE) only for local experimentation; do not wire this into production defaults. By default, production uses GormStore.
- Rollout strategy: All changes are additive and tested in a branch. Since CI/workflow editing is prohibited, integrate tests in code and document how to enable CI test execution.
- Constraints: The environment described (vanilla Debian container) might not have Go toolchain or network access. The plan assumes code modifications are made via repo edits; actual test execution requires a local developer or CI environment with Go installed.

Assumptions & Open Questions
- Assumption: The repo's go.mod already lists gin and gorm; production build remains unchanged.
- Assumption: Developers running tests locally or in CI have Go toolchain available. The current container (described in request) lacks go tooling — creating files is still permitted.
- Open question: Do we want to vendor dependencies (vendor/) so tests and builds can run offline? This requires repo changes and may be heavy — left out of this plan.
- Open question: Preferred module path for internal packages is inferred from go.mod; implementers must ensure import paths in new files use the module path declared in go.mod.

# TASKS

- [ ] **Task 1.1: Add store abstraction and memory store; relocate Ping model**
- **Status:** Pending
- **Context:**
 - Implement internal/store/ping.go with the Ping model (json & gorm tags).
 - Implement internal/store/store.go declaring Store interface:
   - SavePing(*Ping) error
   - GetPings() ([]Ping, error)
 - Implement internal/store/mem_store.go: MemoryStore with:
   - slice backing store, sync.Mutex for concurrency
   - optional error injection (a field or function hook to simulate Save errors)
 - Implement internal/store/gorm_store.go: GormStore adapter wrapping *gorm.DB and implementing Store (uses db.Create & db.Find).
 - Update main.go to import internal/store, construct GormStore with the same db object produced by gorm.Open, and use it when handling endpoints. Keep AutoMigrate pointing to the shared Ping model.
- **Dependencies:** None
- **Validation:**
 - [ ] internal/store/ping.go exists and defines a Ping struct with json and gorm tags
 - [ ] internal/store/store.go exists and defines Store interface with required methods
 - [ ] internal/store/mem_store.go exists and contains MemoryStore with a mutex and an in-memory slice
 - [ ] internal/store/gorm_store.go exists and contains GormStore implementing Store by delegating to *gorm.DB (db.Create, db.Find)
 - [ ] main.go has been updated to construct a GormStore and references the shared Ping model (search for "GormStore" and "AutoMigrate" usages)
 - [ ] (Optional manual) The new code compiles successfully when run with the Go toolchain in a developer environment: go build ./...

- [ ] **Task 1.2: Add business logic API and unit tests using MemoryStore**
- **Status:** Pending
- **Context:**
 - Implement internal/api/api.go with:
   - func CreatePing(s store.Store, p *store.Ping) error
     - validation: trim message; if empty return validation error
     - call s.SavePing(p); propagate or wrap storage errors
   - func ListPings(s store.Store) ([]store.Ping, error)
 - Implement internal/api/api_test.go:
   - Tests using MemoryStore only (no Gin/GORM imports)
   - Tests to add:
     - TestCreatePing_Success: verify SavePing is called and stored
     - TestCreatePing_ValidationError: verify empty-message returns validation error and store unchanged
     - TestCreatePing_StorageError: configure MemoryStore to inject save error and assert CreatePing returns error
     - TestListPings_ReturnsSeeded: seed MemoryStore and assert ListPings returns seeded items
 - Tests must use only the standard library (testing, sync) and internal packages.
- **Dependencies:** Task 1.1
- **Validation:**
 - [ ] internal/api/api.go exists and defines CreatePing and ListPings functions
 - [ ] internal/api/api_test.go exists and contains tests named at least:
   - TestCreatePing_Success
   - TestCreatePing_ValidationError
   - TestCreatePing_StorageError
   - TestListPings_ReturnsSeeded
 - [ ] Tests use MemoryStore and do not import gin or gorm (search test files for "gin" or "gorm")
 - [ ] When run in a developer environment with Go installed, tests pass:
   - Command: go test ./internal/api -v
 - [ ] Each test maps to a SPEC scenario (see Traceability in VERIFICATION)

- [ ] **Task 1.3: Add minimal logging, error mapping in handlers, and documentation**
- **Status:** Pending
- **Context:**
 - Update main.go Gin handlers to:
   - Parse input, call api.CreatePing / api.ListPings
   - Map validation errors to 400 responses and storage/internal errors to 500 responses, preserving current HTTP surface
   - Add structured/log messages for success and error cases (use log.Printf or standard logger)
 - Add a short test-run and developer guide file: TESTS.md (or update README.md) documenting:
   - How to run unit tests locally: go test ./internal/... and specific packages
   - Note about environment: Go toolchain required; no network in the container used for planning — developers must run tests in their environment or CI
   - How to simulate storage errors using MemoryStore hooks in tests
 - Add an env var description for a local dev toggle (USE_IN_MEMORY_STORE) only for local experimentation (documented but not enabled by default).
- **Dependencies:** Task 1.1, Task 1.2
- **Validation:**
 - [ ] Handlers in main.go call internal/api functions and contain error mapping for validation (400) and storage (500)
 - [ ] main.go contains additional log statements around CreatePing and ListPings operations
 - [ ] Document file (TESTS.md or README.md) exists and contains:
   - Test run commands (go test ...)
   - Notes on required Go toolchain and lack of network in the planning container
   - Guidance on how to inject storage errors using MemoryStore
 - [ ] (Manual) A reviewer can follow the doc to run tests in an environment with Go: go test ./... yields the test results for the new tests

- [ ] **Task 1.4: Rollout, observability checks, and rollback plan**
- **Status:** Pending
- **Context:**
 - Add small observability checklist and a rollback / kill-switch note in TESTS.md and README.md:
   - Logs to monitor: "save ping success", "save ping failure", "validation failed"
   - Metrics to observe in production (if available): ping creation rate, ping save error rate, GET /pings latency
   - Rollback plan: revert the commit(s) that added the store/interface if immediate rollback is needed; alternatively, set USE_IN_MEMORY_STORE=0 and redeploy (note: USE_IN_MEMORY_STORE is for dev only; production uses GormStore by default)
 - Ensure all edits are limited to repository files; no workflow edits are required or performed.
- **Dependencies:** Task 1.1, Task 1.2, Task 1.3
- **Validation:**
 - [ ] Documentation contains an observability checklist and a clear rollback plan
 - [ ] Code contains the USE_IN_MEMORY_STORE env var read and default behavior documented (but production default is unchanged)
 - [ ] A reviewer can identify logging points in code for the observability items

Notes on task consolidation and scope:
- Tasks are intentionally coarse-grained to remain ≤ 4 tasks. Each task bundles closely related micro-steps. Implementation should be done via repository edits (apply_patch) only, and no network operations are required to add files. Running tests requires a developer machine or CI with Go toolchain.

# VERIFICATION

Artifact Check
- Confirm this plan file exists at: .frunl/019b06cc-dcc5-7097-8926-70423d6120d0/plan.md

Traceability Table: Scenario → Design Element(s) → Task ID(s)

| Scenario (from SPEC)                                 | Design Elements (files, components)                                    | Task ID(s)      |
|------------------------------------------------------|-------------------------------------------------------------------------|-----------------|
| Create ping successfully using business API          | internal/api/api.go, internal/store/mem_store.go                        | Task 1.2        |
| Create ping fails when message is empty              | internal/api/api.go (validation), internal/store/ping.go                | Task 1.2        |
| List pings returns existing records                  | internal/api/api.go (ListPings), internal/store/mem_store.go            | Task 1.2        |
| Create ping surfaces storage errors as internal error| internal/api/api.go, internal/store/mem_store.go (error injection hook) | Task 1.2        |
| Store abstraction and production adapter             | internal/store/store.go, internal/store/gorm_store.go, main.go          | Task 1.1        |
| Logging and error mapping in HTTP handlers           | main.go (handler changes), internal/api/api.go (errors)                | Task 1.3        |
| Docs, run instructions, rollback/observability       | TESTS.md or README.md updates                                           | Task 1.3 / 1.4  |

Test Strategy (high-level unit tests only)
- Scope: Only core business logic functions (internal/api) and the MemoryStore behavior are unit-tested. No end-to-end tests are included here (they require Gin/GORM runtime and possibly network/db access).
- Test names (examples to add in internal/api/api_test.go):
  - TestCreatePing_Success
  - TestCreatePing_ValidationError
  - TestCreatePing_StorageError
  - TestListPings_ReturnsSeeded
- Test approach:
  - Use MemoryStore for deterministic behavior; MemoryStore should expose a way to seed items and optionally inject an error on Save.
  - Use standard testing.T assertions (t.Fatalf / t.Errorf).
  - Avoid importing gin/gorm in tests.
- Example commands (to be executed by developers/CI with the Go toolchain available):
  - Run package tests (api): go test ./internal/api -v
  - Run all new tests: go test ./... -run TestCreatePing -v
  - To run full suite (if environment has network and modules cached): go test ./... -v
- Note about the planning environment: The planning container described in the request is missing network and external toolchain. The tests must be added to the repository; running them requires a developer machine or CI with Go installed and (if modules are not vendored) network access to fetch dependencies listed in go.mod.

Operational Readiness
- Metrics & Logs to observe after rollout:
  - PingCreationSuccessCount (increment on Save success)
  - PingCreationErrorCount (increment on Save error)
  - PingValidationErrorCount (increment on validation failure)
  - GET /pings latency (histogram or p95 if available)
  - Logs:
    - "validation failed: message empty" with request info (no PII)
    - "save ping succeeded: id=%d" on successful persistence
    - "save ping failed: err=%v" on storage errors
- Feature flag / kill-switch:
  - USE_IN_MEMORY_STORE (env var) — documented for local experimentation only. Default: false. This allows developers to start the service pointed at MemoryStore for quick local runs without touching production configuration. Do not enable in production.
- Rollback plan:
  - If problems are detected, revert the commits that added the store/api refactors and re-deploy the previous commit (standard git revert or branch rollback in source control). Because we do not modify workflows in this plan, rollback is limited to repository content revert.
  - Alternatively, if an emergency hotfix is needed to disable new logging or logic, create a minimal commit that restores previous main.go handler bodies (the changes are small and isolated to main.go and new packages).
- Observability checks (pre-merge checklist):
  - [ ] Logs added in main.go and api functions present
  - [ ] TESTS.md contains run instructions and notes about the planning environment constraints
  - [ ] New tests present in internal/api and passing in a local developer environment with Go

High-level acceptance criteria (manual)
- All files described in TASKS exist in the repository.
- internal/api/api_test.go contains the four test functions listed and they are implemented using MemoryStore without importing gin or gorm.
- Documentation updated to explain how to run tests locally (go test ./...) and to describe the USE_IN_MEMORY_STORE toggle.
- A code reviewer can run the tests in an environment with the Go toolchain and see the new tests run and pass.

Location: .frunl/019b06cc-dcc5-7097-8926-70423d6120d0/plan.md
