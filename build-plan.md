All projects
practice app
Create an app that helps my kids by gamifying practicing


How can I help you today?

    Application plan feedback
    Last message 1 hour ago

Instructions

Add instructions to tailor Claude’s responses
Files
1% of project capacity used

violin-quest-build-plan.md
# Violin Quest — Technical Build Plan
 
## Project Summary
 
Violin Quest is a single-family, self-hosted gamified practice tracker for a child learning violin. A digital prize wheel dictates the practice style each session. The child submits completed sessions from a tablet or browser; a parent approves them from their phone, unlocking XP, levels, and streaks. The app is built as a Progressive Web App (PWA) installable on Android and Windows.
 
### Confirmed Decisions
 
- **Single family, two admins maximum** (hard-capped in the API)
- **Multi-child ready from day one** — `children` table and `child_id` FKs exist even if only one child is ever added
- **Timezone stored per child** in the `children` table, used by the reminder cronjob
- **In-game store** (shields, themes) is a later phase; schema stubs are included from the start
- **No multi-tenant support** — no `families` table, no data isolation concerns
---
 
## Final Database Schema (SQLite)
 
```sql
-- Administrative users. Hard cap of 2 enforced in API logic, not the schema.
CREATE TABLE admins (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT    UNIQUE NOT NULL,
    password_hash TEXT NOT NULL
);
 
-- One row per child. timezone uses IANA format e.g. "Europe/Amsterdam".
CREATE TABLE children (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    name     TEXT    NOT NULL,
    timezone TEXT    NOT NULL DEFAULT 'Europe/Amsterdam'
);
 
-- Global progression stats, one row per child.
CREATE TABLE user_stats (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    child_id          INTEGER NOT NULL REFERENCES children(id),
    total_points      INTEGER DEFAULT 0,
    current_level     INTEGER DEFAULT 1,
    experience_points INTEGER DEFAULT 0,  -- XP progress within current level
    shield_count      INTEGER DEFAULT 0   -- Purchased streak protections (store phase)
);
 
-- Practice sessions submitted by the child.
CREATE TABLE sessions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    child_id        INTEGER NOT NULL REFERENCES children(id),
    date            TEXT    NOT NULL,                    -- Format: YYYY-MM-DD
    tasks_completed INTEGER NOT NULL,
    status          TEXT    NOT NULL DEFAULT 'PENDING',  -- PENDING | APPROVED | REJECTED
    rejection_note  TEXT,                               -- Optional parent note on rejection
    is_first_of_day INTEGER DEFAULT 0,                  -- 1 if this session counts toward streak
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_sessions_child_date_status ON sessions(child_id, date, status);
 
-- Weekly compliance tracking. One row per child per ISO week.
CREATE TABLE weekly_streaks (
    child_id          INTEGER NOT NULL REFERENCES children(id),
    year_week         TEXT    NOT NULL,                  -- Format: "2026-W23" (Monday-start)
    session_count     INTEGER DEFAULT 0,                 -- Unique approved days this week
    milestone_reached INTEGER DEFAULT 0,                 -- 1 once the 3x/week bonus is awarded
    PRIMARY KEY (child_id, year_week)
);
 
-- Wheel task options. Scoped per child so each child can have their own wheel.
CREATE TABLE wheel_options (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    child_id INTEGER NOT NULL REFERENCES children(id),
    text     TEXT    NOT NULL,
    is_bonus INTEGER DEFAULT 0  -- Landing here triggers confetti on the frontend
);
 
-- FCM tokens for push notifications.
CREATE TABLE fcm_tokens (
    token      TEXT     PRIMARY KEY,
    child_id   INTEGER  REFERENCES children(id),  -- NULL for ADMIN role tokens
    role       TEXT     NOT NULL,                 -- 'CHILD' | 'ADMIN'
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
 
-- Store items catalogue (populated in Phase 8+).
CREATE TABLE store_items (
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT    NOT NULL,
    type TEXT    NOT NULL,    -- 'SHIELD' | 'THEME'
    cost INTEGER NOT NULL
);
```
 
---
 
## API Reference
 
### Public / Child Endpoints
 
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/stats?child_id=` | Level, total_points, XP, shield_count, weekly session_count |
| GET | `/api/options?child_id=` | Active wheel task list for this child |
| POST | `/api/session` | Submit a completed session |
| POST | `/api/fcm/register` | Register or refresh an FCM token |
 
**POST /api/session payload:**
```json
{ "child_id": 1, "tasks_completed": 4 }
```
On success: saves a PENDING session, fires async FCM notification to all ADMIN tokens.
 
**POST /api/fcm/register payload:**
```json
{ "token": "fcm-token-string", "role": "CHILD", "child_id": 1 }
```
`child_id` is required for CHILD role, ignored for ADMIN.
 
---
 
### Authentication Endpoints
 
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/auth/login` | Validate credentials, receive JWT + refresh token |
| POST | `/api/auth/refresh` | Exchange refresh token for new JWT |
 
**POST /api/auth/login payload:**
```json
{ "username": "dad", "password": "plaintext" }
```
**Response:**
```json
{ "token": "jwt...", "refresh_token": "opaque-token..." }
```
 
JWT lifetime: **15 minutes**. Refresh token lifetime: **30 days**, stored in the `refresh_tokens` table (token hash, admin_id, expires_at, revoked).
 
---
 
### Protected Admin Endpoints (JWT required via middleware)
 
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/admin/sessions` | List all PENDING sessions |
| POST | `/api/admin/approve/:id` | Run approval workflow (see below) |
| POST | `/api/admin/reject/:id` | Reject a session with an optional note |
| POST | `/api/admin/options` | Add a wheel task |
| DELETE | `/api/admin/options/:id` | Remove a wheel task |
| POST | `/api/admin/create-admin` | Create second admin (blocked if count >= 2) |
 
**POST /api/admin/reject/:id payload:**
```json
{ "note": "Try again after dinner 🎻" }
```
 
---
 
## Core Business Logic
 
### Approval Workflow
 
Runs inside a SQLite transaction to prevent race conditions.
 
```
1. Fetch session by id. Abort if not PENDING.
2. COUNT sessions WHERE child_id = session.child_id
                   AND date = session.date
                   AND status = 'APPROVED'
3. If count == 0:
     SET is_first_of_day = 1
   Else:
     SET is_first_of_day = 0
4. SET status = 'APPROVED'
5. base_points = tasks_completed * 10
   ADD base_points TO user_stats.total_points
   ADD base_points TO user_stats.experience_points
6. If is_first_of_day == 1:
     year_week = format(session.date, "YYYY-[W]WW")  // Go: time.ISOWeek()
     UPSERT weekly_streaks (child_id, year_week)
       SET session_count = session_count + 1
     If session_count >= 3 AND milestone_reached == 0:
       ADD 50 TO user_stats.total_points
       ADD 50 TO user_stats.experience_points
       SET milestone_reached = 1
7. Recalculate level:
     threshold = current_level * 100
     While experience_points >= threshold:
       experience_points -= threshold
       current_level += 1
       threshold = current_level * 100
   UPDATE user_stats
8. Fire async FCM notification to child's CHILD tokens: "⭐ Dad approved your session!"
```
 
### Streak Week Boundary
 
`time.ISOWeek()` in Go returns Monday-start ISO weeks natively. No manual offset required. The `year_week` key format is `"2026-W23"` — zero-padded week number.
 
### 4 PM Practice Reminder Cronjob
 
Uses `robfig/cron/v3`. On startup the backend schedules one job per child, using the child's IANA timezone:
 
```go
for _, child := range children {
    loc, _ := time.LoadLocation(child.Timezone)
    // cron spec in local time using the loaded location
    scheduler.AddFunc("0 16 * * *", func() {
        count := countTodaysSessions(child.ID) // PENDING or APPROVED
        if count == 0 {
            sendFCMToChildDevices(child.ID, "Time for Violin! 🎻 The magic wheel is waiting for you.")
        }
    }, cron.WithLocation(loc))
}
```
 
If a child's timezone is updated, the backend must restart the relevant cron job.
 
---
 
## Frontend Architecture (Svelte + Vite PWA)
 
### Routing
 
Client-side routing with two views:
 
- `/` — Child view (no auth)
- `/admin` — Admin view (login modal, JWT stored in memory + sessionStorage)
The child view URL includes a `?child=1` query param. Each device bookmarks its own URL. No child-side login.
 
### Child View Components
 
**HUD (top bar)**
- XP progress bar: `experience_points / (current_level * 100) * 100%` width
- Level badge
- Three musical note icons for weekly streak (filled = approved day this week)
- Shield count badge (Phase 8+)
**The Wheel (HTML5 Canvas)**
- Draws segments dynamically from `/api/options` on mount
- Segment count and labels are reactive; re-draws on data change
- Spin triggered by button click
- Animation: CSS `transform: rotate()` with `cubic-bezier(0.17, 0.67, 0.12, 0.99)` easing, ~4 second duration
- Landing logic: calculate final rotation modulo 360, map to segment index
- Bonus segment landing: fire `canvas-confetti` with full-screen particle burst
**Session Tracker (bottom drawer)**
- Tracks spin results locally in a Svelte store during the active session
- Shows list of completed tasks
- "Send to Dad 🚀" button activates once spin count >= 1
- On submit: POST `/api/session`, transition to success state (animated checkmark, lock wheel)
- Reset button available after submission to start a new session
### Admin View Components
 
**Login modal**
- Username + password fields
- On success: store JWT in memory, store refresh token in `sessionStorage`
- Auto-refresh: silent token refresh 1 minute before JWT expiry using a `setInterval`
**Pending Dashboard**
- Polls `/api/admin/sessions` on mount and every 30 seconds
- Cards show: child name, date, time submitted, task count
- Approve button → POST `/api/admin/approve/:id` → remove card with animation
- Reject button → opens note input → POST `/api/admin/reject/:id`
**Wheel Content Manager**
- Lists current tasks for selected child
- Inline text input to add new task, toggle bonus flag
- Delete button per task with confirm-on-click
---
 
## Phase Breakdown
 
---
 
### Phase 1 — PoC: The Wheel
 
**Goal:** Child can spin the wheel and tap "Send to Dad." Deployable immediately.
 
**Scope:**
- Svelte + Vite project scaffolded (`npm create vite@latest violin-quest -- --template svelte`)
- Wheel tasks hardcoded as a JS array (8–10 entries)
- HTML5 Canvas wheel drawn with `requestAnimationFrame` on mount
- Spin button triggers CSS rotation animation with `cubic-bezier` easing
- Result label displayed after spin
- Local counter tracks spins this session
- "Send to Dad 🚀" button appears after first spin — for now just shows a success message (`alert` or a reactive success panel)
- No backend, no database, no auth
- Built with `npm run build`, output copied to `/var/www/violin-quest` on the VM, served by Nginx
**Nginx config (minimal):**
```nginx
server {
    listen 80;
    server_name your-domain-or-ip;
    root /var/www/violin-quest;
    index index.html;
    location / { try_files $uri $uri/ /index.html; }
}
```
 
**Deliverable:** Child can open the URL on any device, spin the wheel, and see a fun response.
 
---
 
### Phase 2 — Backend Foundation
 
**Goal:** Real API, real database. Wheel tasks load dynamically.
 
**Scope:**
 
Project structure:
```
violin-quest-api/
  main.go
  db/
    schema.go      // CREATE TABLE statements, run on startup
    db.go          // SQLite connection singleton (mattn/go-sqlite3)
  handlers/
    stats.go
    options.go
    session.go
    fcm.go
  middleware/
    auth.go        // JWT validation
  models/
    models.go      // Go structs matching DB tables
  go.mod
```
 
- `GET /api/stats?child_id=1` — returns `user_stats` joined with `weekly_streaks` for current week
- `GET /api/options?child_id=1` — returns all `wheel_options` for the child
- `POST /api/session` — inserts PENDING session, returns session ID
- SQLite file at `/data/violin-quest.db`, path configurable via env var `DB_PATH`
- Schema auto-migrated on startup (check table existence, create if missing)
- Seed script: inserts one child row and one admin row (bcrypt password hash) on first run if tables are empty
- Frontend updated: remove hardcoded wheel data, call `/api/options` on mount, call `/api/session` on submit
**Dependencies:**
```
github.com/gin-gonic/gin
github.com/mattn/go-sqlite3
golang.org/x/crypto  // bcrypt
```
 
---
 
### Phase 3 — Parent Approval Flow
 
**Goal:** Full core loop working. Child submits → parent approves → XP updates.
 
**Scope:**
 
- `POST /api/auth/login` — bcrypt compare, return signed JWT (HS256, 15min) + refresh token
- `POST /api/auth/refresh` — validate refresh token from DB, issue new JWT
- `refresh_tokens` table: `(id, admin_id, token_hash, expires_at, revoked)`
- JWT middleware: validate `Authorization: Bearer <token>` header on all `/api/admin/*` routes
- `GET /api/admin/sessions` — list PENDING sessions with child name joined
- `POST /api/admin/approve/:id` — full approval algorithm (see Core Business Logic), wrapped in SQLite transaction
- `POST /api/admin/reject/:id` — set status REJECTED, save optional note
Frontend:
- Admin login modal (Svelte component, shown at `/admin` if no valid JWT in memory)
- Pending session cards with Approve / Reject actions
- Child HUD polls `/api/stats` every 10 seconds so XP bar and level update after approval without manual refresh
---
 
### Phase 4 — Streak System & Weekly Goals
 
**Goal:** Habit loop fully operational with visual weekly goal tracking.
 
**Scope:**
 
- `weekly_streaks` upsert logic already included in approval workflow from Phase 3 — this phase makes it *visible*
- `/api/stats` response extended:
```json
{
  "current_level": 3,
  "experience_points": 45,
  "total_points": 345,
  "shield_count": 0,
  "week_session_count": 2,
  "milestone_reached": false
}
```
- Child HUD: three streak icons (e.g. 🎵) — filled/coloured for each approved day this week up to 3, grey for remaining
- Milestone animation: when `week_session_count` reaches 3 for the first time this week, trigger a small celebratory animation (CSS keyframe burst on the streak icons)
- Week boundary: confirm that `time.ISOWeek()` correctly resets the display on Monday — test with a seeded past week in the DB
---
 
### Phase 5 — Push Notifications
 
**Goal:** Parent notified on session submit. Child reminded at 4 PM if no session yet.
 
**Scope:**
 
Firebase setup:
- Create Firebase project, enable Cloud Messaging
- Add `google-services.json` to frontend build (for Web Push via VAPID key)
- Store FCM server key in backend env var `FCM_SERVER_KEY`
Backend:
- `POST /api/fcm/register` — upsert token in `fcm_tokens` (update `updated_at` if token exists)
- FCM send helper function (HTTP v1 API via `googleapis` service account or legacy server key)
- On session submit: goroutine fires notification to all ADMIN tokens: `"🎻 [Child name] just finished practicing! (X tasks)"`
- On session approve: goroutine fires to CHILD tokens for that child: `"⭐ [Admin name] approved your session! +X XP"`
- Cronjob: one scheduled job per child at 16:00 local time, fires to CHILD tokens if no session today
Frontend:
- Request notification permission on child view mount (after first user gesture)
- Register service worker, get FCM token, POST to `/api/fcm/register` with role CHILD and child_id
- On admin view login success: register FCM token with role ADMIN
---
 
### Phase 6 — PWA & Installation
 
**Goal:** App installable as a native-feeling app on Android and Windows.
 
**Scope:**
 
Vite PWA plugin (`vite-plugin-pwa`):
```javascript
// vite.config.js
VitePWA({
  registerType: 'autoUpdate',
  manifest: {
    name: 'Violin Quest',
    short_name: 'ViolinQuest',
    theme_color: '#6C3CE1',
    background_color: '#1A1A2E',
    display: 'standalone',
    orientation: 'portrait',
    icons: [
      { src: '/icons/icon-192.png', sizes: '192x192', type: 'image/png' },
      { src: '/icons/icon-512.png', sizes: '512x512', type: 'image/png' }
    ]
  },
  workbox: {
    globPatterns: ['**/*.{js,css,html,ico,png,svg}'],
    runtimeCaching: [{
      urlPattern: /^https?.*\/api\/options/,
      handler: 'StaleWhileRevalidate'  // wheel tasks load even if offline
    }]
  }
})
```
 
- Create app icons (192×192 and 512×512 PNG)
- Test "Add to Home Screen" on Android Chrome
- Test "Install App" prompt on Windows Chrome/Edge
---
 
### Phase 7 — Wheel Content Management & Confetti
 
**Goal:** Parent can manage wheel tasks via the app. Bonus segments trigger confetti.
 
**Scope:**
 
Backend:
- `POST /api/admin/options` — insert new wheel task for a child
- `DELETE /api/admin/options/:id` — delete wheel task (verify it belongs to an existing child, not a hard constraint but a sanity check)
- `PATCH /api/admin/options/:id` — toggle `is_bonus` flag
Frontend (admin view):
- Wheel content manager: list tasks per child, add/delete/toggle bonus inline
- Changes immediately reflected via reactive Svelte store; re-fetches `/api/options` after any mutation
- Bonus confetti: `import confetti from 'canvas-confetti'` — on wheel land, check if landed segment has `is_bonus = true`, fire:
```javascript
confetti({
  particleCount: 200,
  spread: 120,
  origin: { y: 0.4 },
  colors: ['#FFD700', '#FF69B4', '#00CED1']
})
```
 
---
 
### Phase 8 — In-Game Store: Shields
 
**Goal:** Child can spend points on streak shields. Shields absorb a missed week.
 
**Scope:**
 
Schema already includes `store_items` and `shield_count`. Seed one shield item:
```sql
INSERT INTO store_items (name, type, cost) VALUES ('Streak Shield 🛡️', 'SHIELD', 100);
```
 
Backend:
- `GET /api/store` — returns available store items
- `POST /api/store/buy/:item_id` — deduct cost from `total_points`, apply effect:
  - SHIELD: increment `shield_count`
  - Reject if insufficient points
- Shield consumption: at the Monday cronjob (or on any approval that checks the previous week), if `session_count < 3` for the previous week and `milestone_reached == 0`, check `shield_count > 0` — if so, decrement `shield_count` and set `milestone_reached = 1` with 0 bonus points (streak preserved, no bonus). If no shield, streak is simply missed (no penalty in current design beyond missing the bonus).
Frontend (child view):
- Store drawer/modal accessible from a coin/shop icon in the HUD
- Lists items with cost, current point balance, buy button (disabled if insufficient points)
- Shield count displayed in HUD as a badge
---
 
### Phase 9 — In-Game Store: Themes
 
**Goal:** Child can personalise the look of their view.
 
**Scope:**
 
Add theme items to store:
```sql
INSERT INTO store_items (name, type, cost) VALUES ('Space Theme 🚀', 'THEME', 200);
INSERT INTO store_items (name, type, cost) VALUES ('Ocean Theme 🌊', 'THEME', 200);
```
 
Add `active_theme` column to `user_stats`:
```sql
ALTER TABLE user_stats ADD COLUMN active_theme TEXT DEFAULT 'default';
```
 
Backend:
- `POST /api/store/buy/:item_id` extended: THEME type sets `active_theme` on `user_stats`
- `GET /api/stats` response includes `active_theme`
Frontend:
- CSS custom property themes defined per theme key:
```javascript
const themes = {
  default: { '--bg': '#1A1A2E', '--accent': '#6C3CE1', '--text': '#FFFFFF' },
  space:   { '--bg': '#0D0D1A', '--accent': '#00BFFF', '--text': '#E0E0FF' },
  ocean:   { '--bg': '#001F3F', '--accent': '#00CED1', '--text': '#E0F7FA' }
}
```
- On stats load, apply `active_theme` properties to `:root`
- Theme switcher in store: preview swatch shown per item before purchase
---
 
### Phase 10 — Polish & Hardening
 
**Goal:** App is stable, secure, and ready for daily use indefinitely.
 
**Scope:**
 
Backend:
- Input validation on all endpoints (check required fields, type assertions, max lengths)
- Rate limiting on public endpoints: `POST /api/session` limited to 20 req/min per IP using `gin-contrib/limiter` or a simple in-memory token bucket
- Structured logging with `log/slog` (Go 1.21+): request method, path, status code, duration, errors
- `GET /health` endpoint returns `{ "status": "ok", "db": "ok" }` — checks DB connectivity
- Graceful shutdown: `os.Signal` listener, close DB connection and drain in-flight requests on SIGTERM
Frontend:
- Error boundary: show a friendly "Something went wrong 🎻" screen instead of a blank page on API failure
- Loading skeletons on the wheel and HUD while initial data fetches
- Responsive layout audit: test on 360px wide Android phone, 800px tablet, 1280px laptop
- Rejection UX: when a session is REJECTED, child view shows the parent's note in a gentle modal ("Dad left a message for you 💬")
Infrastructure:
- Nginx rate limiting (`limit_req_zone`) as a second layer
- Certbot auto-renew confirmed (`systemctl status certbot.timer`)
- SQLite WAL mode enabled on startup: `PRAGMA journal_mode=WAL;` — improves concurrent read/write performance
- Daily SQLite backup script: `cp /data/violin-quest.db /data/backups/violin-quest-$(date +%F).db` via cron, keep last 14 days
---
 
## Environment Variables
 
```env
DB_PATH=/data/violin-quest.db
JWT_SECRET=your-random-256-bit-secret
JWT_EXPIRY_MINUTES=15
REFRESH_TOKEN_EXPIRY_DAYS=30
FCM_SERVER_KEY=your-firebase-server-key
PORT=8080
```
 
---
 
## Dependency Summary
 
### Backend (Go)
```
github.com/gin-gonic/gin
github.com/mattn/go-sqlite3
golang.org/x/crypto          // bcrypt
github.com/golang-jwt/jwt/v5
github.com/robfig/cron/v3
firebase.google.com/go/v4    // FCM (or use raw HTTP v1 API)
```
 
### Frontend (Node/Svelte)
```
svelte
vite
vite-plugin-pwa
canvas-confetti
```
 
---
 
## Open Items & Future Considerations
 
- **Background sync for offline session submit** — if the child submits while offline, a service worker background sync job could retry the POST when connectivity is restored. Deferred to post-Phase 10.
- **Multi-child support** — schema is ready (child_id FKs everywhere). Adding a second child requires: inserting a row in `children`, seeding `user_stats` and `wheel_options`, and updating the frontend to route by `?child=` param. No structural migration needed.
- **Second admin** — `POST /api/admin/create-admin` is built in Phase 3. Just call it once from the admin UI or via curl.
- **Streak failure penalty** — currently a missed week simply loses the bonus. A future design could add a visual "broken streak" state to increase loss aversion, but this should be tested carefully to avoid demotivating the child.
 
