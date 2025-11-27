# Quotient Testing Makefile

.PHONY: help test test-unit test-integration test-e2e test-security test-chaos test-coverage test-all clean

# Default target
help:
	@echo "Quotient Test Suite"
	@echo "=================="
	@echo ""
	@echo "Available targets:"
	@echo "  test              - Run all tests (unit + integration)"
	@echo "  test-unit         - Run unit tests only"
	@echo "  test-integration  - Run integration tests (requires Docker)"
	@echo "  test-e2e          - Run end-to-end tests (requires Docker Compose)"
	@echo "  test-security     - Run security tests"
	@echo "  test-chaos        - Run chaos engineering tests"
	@echo "  test-properties   - Run property-based tests with extended iterations"
	@echo "  test-coverage     - Run tests with coverage report"
	@echo "  test-all          - Run ALL test categories"
	@echo "  clean             - Clean test artifacts"
	@echo ""

# Run unit tests (fast, no external dependencies)
test-unit:
	@echo "Running unit tests..."
	@go test -v -race -short ./engine/... ./www/... ./runner/...

# Run integration tests (requires Docker for Testcontainers)
test-integration:
	@echo "Running integration tests..."
	@go test -v -race ./tests/integration/... -timeout 10m

# Run E2E tests (requires Docker Compose)
test-e2e:
	@echo "Running E2E tests..."
	@echo "Starting Docker Compose stack..."
	@docker compose up -d --wait
	@go test -v ./tests/e2e/... -timeout 15m || (docker compose down && exit 1)
	@docker compose down

# Run security tests
test-security:
	@echo "Running security tests..."
	@go test -v ./tests/security/... -timeout 10m

# Run chaos engineering tests
test-chaos:
	@echo "Running chaos engineering tests..."
	@go test -v ./tests/chaos/... -timeout 10m

# Run property-based tests with extended iterations
test-properties:
	@echo "Running property-based tests (10000 iterations)..."
	@go test -v ./engine/... ./www/... -run Property -rapid.checks=10000

# Default test target (unit + integration)
test:
	@echo "Running standard test suite..."
	@go test -v -race ./...

# Run all test categories
test-all: test-unit test-integration test-e2e test-security test-chaos

# Generate coverage report
test-coverage:
	@echo "Generating coverage report..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@go tool cover -func=coverage.out | grep total

# Clean test artifacts
clean:
	@echo "Cleaning test artifacts..."
	@rm -f coverage.out coverage.html
	@rm -rf tests/testdata/tmp
	@docker compose down -v 2>/dev/null || true
	@echo "Clean complete"

# Install test dependencies
install-deps:
	@echo "Installing test dependencies..."
	@go get github.com/stretchr/testify
	@go get pgregory.net/rapid
	@go get github.com/playwright-community/playwright-go
	@go get github.com/testcontainers/testcontainers-go
	@go mod tidy
	@echo "Dependencies installed"

# Install Playwright browsers
install-playwright:
	@echo "Installing Playwright browsers..."
	@go run github.com/playwright-community/playwright-go/cmd/playwright install --with-deps
	@echo "Playwright browsers installed"

# Run tests in CI environment
ci: test-unit test-integration test-security
	@echo "CI test suite complete"
