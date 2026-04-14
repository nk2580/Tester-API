# Implementation Plan - convert this API to use gRPC

## Phase 1 – Service Design & Proto Modeling
### Tasks
- Inventory current HTTP routes, payloads, and auth rules in `app.go` and `README.md` to ensure every behavior (pings, hello, time, auth) is represented.
- Define gRPC services and RPCs (e.g., `Pinger`, `Time`, `Auth`) plus message types mirroring existing JSON schemas for `Ping`, `User`, `AuthToken`.
- Decide error/status mapping (HTTP codes → `codes.InvalidArgument`, `codes.Unauthenticated`, etc.) and token delivery strategy (metadata vs response message) for parity.
- Create `api/proto/tester.proto` (and optionally a Buf workspace) describing services, imports, and options (package name, Go package path, `google.protobuf.Timestamp`).
- Review compatibility requirements (do we need streaming? keep `/hello`?) with stakeholders and capture non-gRPC clients' needs.
