# Violin Quest — Claude Code Guidelines

## Testing

After any major change (new feature, refactor, schema change, new endpoint, dependency update), always run the full test suite before declaring the work done:

```bash
make test
```

This runs:
- `make test-backend` — Go integration tests (in-memory SQLite, no server needed, fast)
- `make test-e2e` — Playwright E2E tests (auto-starts backend on port 18081 + frontend on port 5174)

The backend tests are the fast safety net (~1s). The E2E tests are the source of truth for user-facing behaviour. Both must pass.
