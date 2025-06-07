# Test Plan for Quotient

This document outlines the proposed tests for the Quotient scoring system.  All tests are written using Go's standard `testing` package along with helpers such as `httptest` for HTTP endpoints.  Mocking libraries (e.g. `github.com/go-redis/redismock/v9` or `github.com/DATA-DOG/go-sqlmock`) may be used where appropriate.

## 1. Configuration Handling
- **Loading and validation** (`engine/config/config.go`)
  - Successful load of a valid `event.conf`.
  - Failure scenarios for missing required fields or invalid values.
  - Detection of duplicate box or runner names.
  - Defaults for values such as `Points`, `Timeout`, `Delay`, `SlaThreshold`, etc.
- **Hot reload** (#18)
  - `WatchConfig` should trigger `SetConfig` when the file is modified.

## 2. Credentials
- **Initial credential loading** (`engine/credentials.go`)
  - `LoadCredentials` copies configured credlists to team submission directories.
  - Handles missing or unreadable source files gracefully.
- **PCR updates and feedback** (#13)
  - `UpdateCredentials` correctly updates existing entries and returns the number of changed records.
  - Concurrent updates are synchronized via mutex.
- **Retrieving credlists**
  - `GetCredlists` returns expected metadata for each configured list.

## 3. Scoring Engine
- **Round execution** (`engine/engine.go`)
  - `Start` queues tasks for active teams and collects results from Redis.
  - `processCollectedResults` records service checks and updates SLA/uptime data.
- **Engine state changes**
  - `PauseEngine` and `ResumeEngine` toggle the pause flag correctly.
  - `ResetScores` clears DB tables and publishes the `reset` event (#41/#40).
- **Active task reporting**
  - `GetActiveTasks` aggregates task status from Redis (#35).

## 4. Runner Service
- **Task creation** (`runner/runner.go`)
  - `createRunner` instantiates the correct check type from task data.
- **Task execution**
  - `handleTask` respects deadlines and writes results back to Redis.
  - Runner exits on `reset` event broadcast (#41).

## 5. Service Checks
Each check under `engine/checks/` should have unit tests covering:
- `Verify` – ensures defaults are applied and invalid configurations return an error.
- `Run` – using mocks to simulate network services (e.g. mock SMTP, DNS, ping). Focus on edge cases noted in issues such as timeouts or incorrect record formats.
- Custom checks – verify command substitution for `ROUND`, `TARGET`, `USERNAME`, etc.

## 6. Database Layer
- Functions in `engine/db/*.go` should be exercised using a test database (SQLite or sqlmock):
  - Team and box creation and retrieval.
  - Round insertion and retrieval.
  - Aggregation helpers like `GetServiceCheckSumByTeam`.
  - Manual score modifications (#48) once implemented.

## 7. Web API
- **Authentication** (`www/api/authentication.go`)
  - Login success and failure create/clear the session cookie correctly.
  - Role detection when LDAP settings are present.
- **Announcements & Injects APIs**
  - CRUD operations enforce required fields and access control.
  - File download endpoints reject paths outside the allowed directory.
- **PCR API**
  - Submitting PCR updates returns the updated count as feedback (#13).
- **Graphs and service status**
  - Endpoints return expected JSON structure for scores and uptimes.

## 8. Middleware & Routing
- Middleware chain correctly attaches request IDs, handles CORS, and enforces authentication.
- Router routes map to handlers for each role (public, team, red, admin) and serve static assets.

## 9. Security and Error Handling
- Ensure helper functions like `GetFile` do not permit directory traversal (see TODO in `helpers.go`).
- Verify that invalid input to API endpoints results in appropriate HTTP status codes without leaking sensitive information.

## Running Tests
All tests can be executed with:

```bash
go test ./...
```

Integration tests requiring Redis or PostgreSQL can be configured to use Docker containers within the CI environment.

