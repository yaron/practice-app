.PHONY: test test-backend test-e2e build-backend

# Run all tests
test: test-backend test-e2e

# Backend integration tests (in-memory SQLite, no server needed)
test-backend:
	cd backend && go test -v ./...

# Build the backend binary (optional; speeds up repeated E2E runs)
build-backend:
	cd backend && go build -o violin-quest-api .

# Playwright E2E tests (starts backend + frontend dev server automatically)
# First-run: go run compiles the backend (~10-20s). Subsequent runs reuse the
# existing server when not in CI (PLAYWRIGHT_REUSE=true is the default).
test-e2e:
	cd frontend && npm run test:e2e

# Open Playwright's interactive UI mode
test-e2e-ui:
	cd frontend && npm run test:e2e:ui
