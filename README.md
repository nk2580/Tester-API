# Tester-API
Tester API Written in Go, this is a means of validating another tool

Run the single new unit test:

    go test -run TestPingEndpoints ./...

The test runs entirely in-process using an in-memory SQLite database and does not open network ports.
