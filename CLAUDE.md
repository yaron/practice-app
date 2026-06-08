# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Violin Quest** — a gamified practice tracker for a child learning violin. A prize wheel picks the practice style each session. The child submits completed sessions; a parent approves them, unlocking XP, levels, and streaks. The app targets Android (tablet for child, phone for parent) and Windows as a PWA.

The UI is in **Dutch** — all user-facing strings, error messages, and button labels are in Dutch.

---

## Commands

### Running tests

```bash
make test              # full suite: backend integration + E2E
make test-backend      # Go integration tests only (~1s, in-memory SQLite)
make test-e2e          # Playwright E2E (auto-starts backend + frontend dev server)
make test-e2e-ui       # Playwright interactive UI mode
```

Run a single Go test by name:
```bash
cd backend && go test -v -run TestApproveSession ./...
```

### Development servers

```bash
# Backend (port 8080, uses ./violin-quest.db by default)
cd backend && go run .

# Frontend dev server (port 5173)
cd frontend && npm run dev
```

### Build

```bash
cd backend && go build -o violin-quest-api .   # backend binary
cd frontend && npm run build                   # frontend static assets
```

---

## Architecture

This is a monorepo with two top-level directories:

- `backend/` — Go API server (Gin + SQLite via mattn/go-sqlite3)
- `frontend/` — Svelte 5 + Vite SPA

### Backend structure

```
backend/
  main.go          # router setup, CORS middleware, server start
  db/
    db.go          # SQLite connection singleton (db.DB *sql.DB); GetAdminTokens/GetChildTokens/DeleteFCMToken
    schema.go      # Migrate() — CREATE TABLE IF NOT EXISTS for every table
    seed.go        # Seed() — inserts default child + admin on first run; exports DefaultWheelOptions
  handlers/        # one file per domain, each handler is a gin.HandlerFunc
    admin.go       # session approval/rejection, history
    auth.go        # login, refresh token
    fcm.go         # FCM token registration (CHILD: open, ADMIN: requires JWT)
    options.go     # wheel options CRUD
    session.go     # session submit
    stats.go       # stats + weekly streak query
    users.go       # children + admins CRUD
  fcm/
    fcm.go         # HTTP v1 FCM client; Send() is a no-op if FCM_PROJECT_ID is unset
  middleware/
    auth.go        # JWT validation; sets "admin_id" in gin context
  models/
    models.go      # Go structs matching DB tables + API response shapes
  reminder/
    reminder.go    # StartReminders() — single cron instance, 16:00 daily per child timezone
  integration_test.go  # all backend tests in one file, uses in-memory SQLite
```

The DB connection is a package-level `db.DB *sql.DB` — handlers import `violin-quest-api/db` and use it directly. There is no ORM or repository layer.

**Schema key points:**
- `session_tasks` table stores the individual task names for each session (many-to-many via `session_id`). `sessions.tasks_completed` is a derived integer count.
- `wheel_options` has both `text` (full label) and `short_text` (abbreviated, shown on the canvas wheel).
- `weekly_streaks` uses ISO week keys (`"2026-W23"`, Monday-start via `time.ISOWeek()`).
- Approval workflow runs inside a SQLite transaction to prevent race conditions — see `ApproveSession` in `handlers/admin.go`.

### Frontend structure

```
frontend/
  src/
    main.js            # Svelte app mount; registers firebase-messaging-sw.js service worker
    App.svelte         # root: routes by pathname, owns child-view state
    Admin.svelte       # admin shell: auth state, tab switching, polling, logout
    firebase.js        # Firebase app + Messaging init (no-op if VITE_FIREBASE_VAPID_KEY unset)
    app.css            # global styles
    components/
      Wheel.svelte         # HTML5 Canvas prize wheel
      SessionTracker.svelte
      SuccessPanel.svelte
      Hud.svelte           # XP bar, level badge, weekly streak icons
      AdminLogin.svelte
      PendingCards.svelte
      SessionHistory.svelte
      UserManagement.svelte
  public/
    firebase-messaging-sw.js  # background notification handler; __VITE_FIREBASE_*__ placeholders
                               # replaced at build time by the inject-sw-firebase-config Vite plugin
  e2e/
    app.spec.js        # Playwright tests (child view + admin workflows)
```

**Routing** is path-based in `App.svelte` (no router library):
- `/child/:id` — child view; `CHILD_ID` parsed from path
- `/admin` — admin view
- Anything else shows an error

**Auth:** JWT (15 min) held in memory (`jwt` reactive state). Refresh token (30 days) stored in `sessionStorage` under key `vq_refresh_token`. On admin mount, a stored refresh token is silently exchanged for a fresh JWT. Auto-refresh is scheduled 1 minute before expiry via `setInterval`.

**API base URL:** `VITE_API_URL` env var, falls back to `http://localhost:8080`.

---

## Environment Variables

### Backend

| Variable | Default | Description |
|---|---|---|
| `DB_PATH` | `./violin-quest.db` | SQLite file path |
| `PORT` | `8080` | HTTP listen port |
| `JWT_SECRET` | dev fallback (insecure) | HS256 signing key — **required in production** |
| `JWT_EXPIRY_MINUTES` | `15` | JWT lifetime |
| `REFRESH_TOKEN_EXPIRY_DAYS` | `30` | Refresh token lifetime |
| `ALLOWED_ORIGINS` | `*` (all) | Comma-separated CORS allowed origins — set in production |
| `FCM_PROJECT_ID` | — | Firebase project ID (from service account JSON) |
| `GOOGLE_APPLICATION_CREDENTIALS` | — | Path to Firebase service account JSON file |

### Frontend (Vite env vars)

| Variable | Description |
|---|---|
| `VITE_API_URL` | Backend base URL (no trailing slash) |
| `VITE_FIREBASE_API_KEY` | Firebase web app API key |
| `VITE_FIREBASE_PROJECT_ID` | Firebase project ID |
| `VITE_FIREBASE_MESSAGING_SENDER_ID` | Firebase messaging sender ID |
| `VITE_FIREBASE_APP_ID` | Firebase app ID |
| `VITE_FIREBASE_VAPID_KEY` | VAPID key for web push (Firebase Console → Cloud Messaging → Web configuration) |

See `frontend/.env.example` for a template. FCM is silently disabled if `VITE_FIREBASE_VAPID_KEY` is empty.

---

## Testing

After any major change (new feature, refactor, schema change, new endpoint, dependency update), always run the full test suite before declaring the work done:

```bash
make test
```

This runs:
- `make test-backend` — Go integration tests (in-memory SQLite, no server needed, fast)
- `make test-e2e` — Playwright E2E tests (auto-starts backend on port 18081 + frontend on port 5174)

The backend tests are the fast safety net (~1s). The E2E tests are the source of truth for user-facing behaviour. Both must pass.

The E2E backend uses a fresh DB at `/tmp/vq-e2e.db` (deleted before each CI run). In dev, `reuseExistingServer: true` means the servers stay up between test runs for speed.

---

## Build Plan Context

The app is built in phases (defined in `build-plan.md`). Phases 1–5 are complete. The next phase is **Phase 6 — PWA/Installation**. Key upcoming phases: 7 (wheel content management + confetti), 8–9 (in-game store).

### Phase 5 (Push Notifications) — implementation notes
- `backend/fcm/fcm.go` — HTTP v1 FCM client; `Send()` is a no-op when `FCM_PROJECT_ID` is unset
- `backend/reminder/reminder.go` — single cron instance with per-child timezone specs (`TZ=... 0 16 * * *`)
- `frontend/src/firebase.js` — Firebase Messaging init; no-op if `VITE_FIREBASE_VAPID_KEY` is empty
- `frontend/public/firebase-messaging-sw.js` — background notification handler; `__VITE_FIREBASE_*__` placeholders replaced at build time
- `frontend/src/main.js` — registers the service worker on startup
- FCM token registration: child registrations are open (validates child_id exists); admin registrations require a valid JWT
- Notifications fired on: session submit (→ admins), session approved (→ child), session rejected (→ child with note)
- Reminders: daily 16:00 per child's timezone; skipped if child already has a session that day

### Production checklist
Before deploying, ensure:
1. `JWT_SECRET` is set to a strong random value
2. `ALLOWED_ORIGINS` is set to the frontend domain
3. All `VITE_FIREBASE_*` vars are set in the build environment
4. `FCM_PROJECT_ID` and `GOOGLE_APPLICATION_CREDENTIALS` are set on the backend server
5. Default admin password (`changeme`) has been changed
