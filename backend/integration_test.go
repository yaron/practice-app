package main_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"violin-quest-api/db"
	"violin-quest-api/handlers"
	"violin-quest-api/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	log.SetOutput(io.Discard) // suppress db/seed startup logs
	os.Exit(m.Run())
}

// setupTestRouter creates a fresh in-memory SQLite DB, runs migrations and seed,
// then returns a Gin router wired to all handlers (public + auth + admin).
func setupTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	var err error
	db.DB, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.DB.Close() })

	if _, err := db.DB.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		t.Fatalf("pragma: %v", err)
	}

	db.Migrate()
	db.Seed()

	r := gin.New()
	api := r.Group("/api")
	api.GET("/options", handlers.GetOptions)
	api.GET("/stats", handlers.GetStats)
	api.POST("/session", handlers.SubmitSession)
	api.POST("/auth/login", handlers.Login)
	api.POST("/auth/refresh", handlers.RefreshToken)

	admin := api.Group("/admin")
	admin.Use(middleware.JWTAuth())
	admin.GET("/sessions", handlers.ListPendingSessions)
	admin.GET("/history", handlers.GetHistory)
	admin.POST("/approve/:id", handlers.ApproveSession)
	admin.POST("/reject/:id", handlers.RejectSession)
	admin.GET("/children", handlers.ListChildren)
	admin.PATCH("/children/:id", handlers.RenameChild)
	admin.GET("/admins", handlers.ListAdmins)
	admin.POST("/admins", handlers.CreateAdmin)
	admin.DELETE("/admins/:id", handlers.DeleteAdmin)
	admin.PATCH("/admins/:id", handlers.UpdateAdmin)

	return r
}

// --- helpers ---

func postJSON(t *testing.T, r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func authReq(t *testing.T, r *gin.Engine, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var buf *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewReader(b)
	} else {
		buf = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func loginAsAdmin(t *testing.T, r *gin.Engine) string {
	t.Helper()
	w := postJSON(t, r, "/api/auth/login", map[string]any{
		"username": "admin",
		"password": "changeme",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
	return resp["token"].(string)
}

// --- GET /api/options ---

func TestGetOptions_ReturnsSeededOptions(t *testing.T) {
	r := setupTestRouter(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/options?child_id=1", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var options []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &options); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(options) != 7 {
		t.Errorf("expected 7 options, got %d", len(options))
	}
	if options[0]["text"] != "Speel een liedje op 1 been" {
		t.Errorf("unexpected first option text: %v", options[0]["text"])
	}
	if options[0]["short_text"] != "1 Been 1️⃣" {
		t.Errorf("unexpected first option short_text: %v", options[0]["short_text"])
	}
	for _, o := range options {
		if o["child_id"].(float64) != 1 {
			t.Errorf("option has wrong child_id: %v", o["child_id"])
		}
		if o["short_text"] == "" {
			t.Errorf("option has empty short_text: %v", o)
		}
	}
}

func TestGetOptions_MissingChildID(t *testing.T) {
	r := setupTestRouter(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/options", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetOptions_UnknownChildReturnsEmptyArray(t *testing.T) {
	r := setupTestRouter(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/options?child_id=999", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var options []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &options); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(options) != 0 {
		t.Errorf("expected empty array for unknown child, got %d items", len(options))
	}
}

// --- GET /api/stats ---

func TestGetStats_ReturnsStatsForSeededChild(t *testing.T) {
	r := setupTestRouter(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/stats?child_id=1", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var stats map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, field := range []string{"current_level", "experience_points", "total_points", "shield_count", "week_session_count", "milestone_reached", "year_week"} {
		if _, ok := stats[field]; !ok {
			t.Errorf("missing field %q in stats response", field)
		}
	}
	if stats["current_level"].(float64) != 1 {
		t.Errorf("expected current_level=1, got %v", stats["current_level"])
	}
	if stats["week_session_count"].(float64) != 0 {
		t.Errorf("expected week_session_count=0 for fresh DB, got %v", stats["week_session_count"])
	}
}

func TestGetStats_MissingChildID(t *testing.T) {
	r := setupTestRouter(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/stats", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetStats_UnknownChild(t *testing.T) {
	r := setupTestRouter(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/stats?child_id=999", nil))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- POST /api/session ---

func TestSubmitSession_CreatesSession(t *testing.T) {
	r := setupTestRouter(t)

	w := postJSON(t, r, "/api/session", map[string]any{
		"child_id":        1,
		"tasks_completed": 3,
	})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := resp["id"]; !ok {
		t.Error("expected id in response")
	}
}

func TestSubmitSession_PersistsToDatabase(t *testing.T) {
	r := setupTestRouter(t)

	postJSON(t, r, "/api/session", map[string]any{
		"child_id":        1,
		"tasks_completed": 4,
	})

	var count int
	db.DB.QueryRow(`SELECT COUNT(*) FROM sessions WHERE child_id=1 AND tasks_completed=4 AND status='PENDING'`).Scan(&count) //nolint:errcheck
	if count != 1 {
		t.Errorf("expected 1 pending session in DB, got %d", count)
	}
}

func TestSubmitSession_ZeroTasksRejected(t *testing.T) {
	r := setupTestRouter(t)

	w := postJSON(t, r, "/api/session", map[string]any{
		"child_id":        1,
		"tasks_completed": 0,
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSubmitSession_UnknownChildRejected(t *testing.T) {
	r := setupTestRouter(t)

	w := postJSON(t, r, "/api/session", map[string]any{
		"child_id":        999,
		"tasks_completed": 3,
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- POST /api/auth/login ---

func TestLogin_HappyPath(t *testing.T) {
	r := setupTestRouter(t)

	w := postJSON(t, r, "/api/auth/login", map[string]any{
		"username": "admin",
		"password": "changeme",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
	if _, ok := resp["token"]; !ok {
		t.Error("expected token in response")
	}
	if _, ok := resp["refresh_token"]; !ok {
		t.Error("expected refresh_token in response")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	r := setupTestRouter(t)

	w := postJSON(t, r, "/api/auth/login", map[string]any{
		"username": "admin",
		"password": "wrong",
	})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	r := setupTestRouter(t)

	w := postJSON(t, r, "/api/auth/login", map[string]any{
		"username": "nobody",
		"password": "changeme",
	})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// --- POST /api/auth/refresh ---

func TestRefreshToken_HappyPath(t *testing.T) {
	r := setupTestRouter(t)

	// Login to get a refresh token
	w := postJSON(t, r, "/api/auth/login", map[string]any{
		"username": "admin",
		"password": "changeme",
	})
	var loginResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &loginResp) //nolint:errcheck
	refreshToken := loginResp["refresh_token"].(string)

	// Exchange it for a new JWT
	w2 := postJSON(t, r, "/api/auth/refresh", map[string]any{
		"refresh_token": refreshToken,
	})

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	var refreshResp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &refreshResp) //nolint:errcheck
	if _, ok := refreshResp["token"]; !ok {
		t.Error("expected token in refresh response")
	}
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	r := setupTestRouter(t)

	w := postJSON(t, r, "/api/auth/refresh", map[string]any{
		"refresh_token": "not-a-real-token",
	})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// --- GET /api/admin/sessions ---

func TestListPendingSessions_RequiresAuth(t *testing.T) {
	r := setupTestRouter(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/sessions", nil))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestListPendingSessions_EmptyByDefault(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	w := authReq(t, r, http.MethodGet, "/api/admin/sessions", nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var sessions []any
	json.Unmarshal(w.Body.Bytes(), &sessions) //nolint:errcheck
	if len(sessions) != 0 {
		t.Errorf("expected empty list, got %d", len(sessions))
	}
}

func TestListPendingSessions_ShowsPendingSession(t *testing.T) {
	r := setupTestRouter(t)

	// Submit a session as child
	postJSON(t, r, "/api/session", map[string]any{
		"child_id":        1,
		"tasks_completed": 3,
	})

	token := loginAsAdmin(t, r)
	w := authReq(t, r, http.MethodGet, "/api/admin/sessions", nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var sessions []map[string]any
	json.Unmarshal(w.Body.Bytes(), &sessions) //nolint:errcheck
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0]["child_name"] == nil {
		t.Error("expected child_name to be present")
	}
}

// --- POST /api/admin/approve/:id ---

func TestApproveSession_HappyPath(t *testing.T) {
	r := setupTestRouter(t)

	// Submit session (3 tasks → 30 points)
	sw := postJSON(t, r, "/api/session", map[string]any{
		"child_id":        1,
		"tasks_completed": 3,
	})
	var sessResp map[string]any
	json.Unmarshal(sw.Body.Bytes(), &sessResp) //nolint:errcheck
	sessID := int(sessResp["id"].(float64))

	token := loginAsAdmin(t, r)
	w := authReq(t, r, http.MethodPost, "/api/admin/approve/"+itoa(sessID), nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
	if resp["approved"] != true {
		t.Error("expected approved=true")
	}
	if resp["base_points"].(float64) != 30 {
		t.Errorf("expected 30 base_points, got %v", resp["base_points"])
	}
}

func TestApproveSession_UpdatesStats(t *testing.T) {
	r := setupTestRouter(t)

	sw := postJSON(t, r, "/api/session", map[string]any{
		"child_id":        1,
		"tasks_completed": 5,
	})
	var sessResp map[string]any
	json.Unmarshal(sw.Body.Bytes(), &sessResp) //nolint:errcheck
	sessID := int(sessResp["id"].(float64))

	token := loginAsAdmin(t, r)
	authReq(t, r, http.MethodPost, "/api/admin/approve/"+itoa(sessID), nil, token)

	// Verify stats updated: 5 tasks × 10 = 50 points
	var totalPoints, xp int
	db.DB.QueryRow(`SELECT total_points, experience_points FROM user_stats WHERE child_id = 1`).Scan(&totalPoints, &xp) //nolint:errcheck
	if totalPoints != 50 {
		t.Errorf("expected total_points=50, got %d", totalPoints)
	}
	if xp != 50 {
		t.Errorf("expected experience_points=50, got %d", xp)
	}
}

func TestApproveSession_LevelUp(t *testing.T) {
	r := setupTestRouter(t)

	// 10 tasks × 10 = 100 XP → level up from 1 to 2 (threshold 100 XP)
	sw := postJSON(t, r, "/api/session", map[string]any{
		"child_id":        1,
		"tasks_completed": 10,
	})
	var sessResp map[string]any
	json.Unmarshal(sw.Body.Bytes(), &sessResp) //nolint:errcheck
	sessID := int(sessResp["id"].(float64))

	token := loginAsAdmin(t, r)
	authReq(t, r, http.MethodPost, "/api/admin/approve/"+itoa(sessID), nil, token)

	var level, xp int
	db.DB.QueryRow(`SELECT current_level, experience_points FROM user_stats WHERE child_id = 1`).Scan(&level, &xp) //nolint:errcheck
	if level != 2 {
		t.Errorf("expected level=2 after 100 XP, got %d", level)
	}
	if xp != 0 {
		t.Errorf("expected xp=0 after exact level-up, got %d", xp)
	}
}

func TestApproveSession_NotPending(t *testing.T) {
	r := setupTestRouter(t)

	sw := postJSON(t, r, "/api/session", map[string]any{
		"child_id":        1,
		"tasks_completed": 2,
	})
	var sessResp map[string]any
	json.Unmarshal(sw.Body.Bytes(), &sessResp) //nolint:errcheck
	sessID := int(sessResp["id"].(float64))

	token := loginAsAdmin(t, r)
	authReq(t, r, http.MethodPost, "/api/admin/approve/"+itoa(sessID), nil, token)
	// Second approve must fail
	w := authReq(t, r, http.MethodPost, "/api/admin/approve/"+itoa(sessID), nil, token)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestApproveSession_WeeklyMilestone(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	// Approve 3 different sessions (simulating 3 different days)
	// We insert sessions directly with different dates so weekly_streaks counts them
	for i, date := range []string{"2026-01-05", "2026-01-06", "2026-01-07"} {
		db.DB.Exec( //nolint:errcheck
			`INSERT INTO sessions (child_id, date, tasks_completed, status) VALUES (1, ?, 2, 'PENDING')`,
			date,
		)
		var sessID int
		db.DB.QueryRow(`SELECT MAX(id) FROM sessions`).Scan(&sessID) //nolint:errcheck
		w := authReq(t, r, http.MethodPost, "/api/admin/approve/"+itoa(sessID), nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("approve %d failed: %d %s", i+1, w.Code, w.Body.String())
		}
	}

	// Third approval should have triggered milestone bonus (+50)
	// 3 sessions × 2 tasks × 10 pts = 60 base + 50 milestone = 110 total
	var totalPoints int
	db.DB.QueryRow(`SELECT total_points FROM user_stats WHERE child_id = 1`).Scan(&totalPoints) //nolint:errcheck
	if totalPoints != 110 {
		t.Errorf("expected total_points=110 (3×20 base + 50 milestone), got %d", totalPoints)
	}

	// milestone_reached must be set
	var milestone int
	db.DB.QueryRow(`SELECT milestone_reached FROM weekly_streaks WHERE child_id = 1 AND year_week = '2026-W02'`).Scan(&milestone) //nolint:errcheck
	if milestone != 1 {
		t.Errorf("expected milestone_reached=1, got %d", milestone)
	}
}

// --- POST /api/admin/reject/:id ---

func TestRejectSession_HappyPath(t *testing.T) {
	r := setupTestRouter(t)

	sw := postJSON(t, r, "/api/session", map[string]any{
		"child_id":        1,
		"tasks_completed": 2,
	})
	var sessResp map[string]any
	json.Unmarshal(sw.Body.Bytes(), &sessResp) //nolint:errcheck
	sessID := int(sessResp["id"].(float64))

	token := loginAsAdmin(t, r)
	w := authReq(t, r, http.MethodPost, "/api/admin/reject/"+itoa(sessID), map[string]any{
		"note": "Probeer het morgen nog eens 🎻",
	}, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var status, note string
	db.DB.QueryRow(`SELECT status, rejection_note FROM sessions WHERE id = ?`, sessID).Scan(&status, &note) //nolint:errcheck
	if status != "REJECTED" {
		t.Errorf("expected status=REJECTED, got %s", status)
	}
	if note != "Probeer het morgen nog eens 🎻" {
		t.Errorf("expected note to be set, got %q", note)
	}
}

func TestRejectSession_NotFoundOrAlreadyProcessed(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	w := authReq(t, r, http.MethodPost, "/api/admin/reject/9999", nil, token)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- POST /api/session with tasks[] ---

func TestSubmitSession_StoresTasks(t *testing.T) {
	r := setupTestRouter(t)

	w := postJSON(t, r, "/api/session", map[string]any{
		"child_id": 1,
		"tasks":    []string{"Speel een liedje op 1 been", "Speel een liedje boos"},
	})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
	sessID := int(resp["id"].(float64))

	var count int
	db.DB.QueryRow(`SELECT COUNT(*) FROM session_tasks WHERE session_id = ?`, sessID).Scan(&count) //nolint:errcheck
	if count != 2 {
		t.Errorf("expected 2 session_tasks rows, got %d", count)
	}
}

// --- GET /api/admin/history ---

func TestGetHistory_RequiresAuth(t *testing.T) {
	r := setupTestRouter(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/history", nil))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetHistory_EmptyByDefault(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	w := authReq(t, r, http.MethodGet, "/api/admin/history", nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var sessions []any
	json.Unmarshal(w.Body.Bytes(), &sessions) //nolint:errcheck
	if len(sessions) != 0 {
		t.Errorf("expected empty history, got %d", len(sessions))
	}
}

func TestGetHistory_IncludesTaskNames(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	// Submit a session with named tasks
	sw := postJSON(t, r, "/api/session", map[string]any{
		"child_id": 1,
		"tasks":    []string{"Speel een liedje op 1 been", "Speel een liedje boos", "Speel een liedje blij"},
	})
	var sessResp map[string]any
	json.Unmarshal(sw.Body.Bytes(), &sessResp) //nolint:errcheck
	sessID := int(sessResp["id"].(float64))

	// Approve it so it appears in history
	authReq(t, r, http.MethodPost, "/api/admin/approve/"+itoa(sessID), nil, token)

	w := authReq(t, r, http.MethodGet, "/api/admin/history", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var sessions []map[string]any
	json.Unmarshal(w.Body.Bytes(), &sessions) //nolint:errcheck
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session in history, got %d", len(sessions))
	}

	tasks, ok := sessions[0]["tasks"].([]any)
	if !ok || len(tasks) != 3 {
		t.Errorf("expected 3 tasks in history, got %v", sessions[0]["tasks"])
	}
	if tasks[0].(string) != "Speel een liedje op 1 been" {
		t.Errorf("unexpected first task: %v", tasks[0])
	}
}

func TestGetHistory_ShowsAllStatuses(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	// Submit + approve one session
	sw1 := postJSON(t, r, "/api/session", map[string]any{
		"child_id": 1, "tasks_completed": 2,
	})
	var r1 map[string]any
	json.Unmarshal(sw1.Body.Bytes(), &r1) //nolint:errcheck
	authReq(t, r, http.MethodPost, "/api/admin/approve/"+itoa(int(r1["id"].(float64))), nil, token)

	// Submit + reject one session
	sw2 := postJSON(t, r, "/api/session", map[string]any{
		"child_id": 1, "tasks_completed": 1,
	})
	var r2 map[string]any
	json.Unmarshal(sw2.Body.Bytes(), &r2) //nolint:errcheck
	authReq(t, r, http.MethodPost, "/api/admin/reject/"+itoa(int(r2["id"].(float64))), nil, token)

	// Submit one that stays PENDING
	postJSON(t, r, "/api/session", map[string]any{
		"child_id": 1, "tasks_completed": 1,
	})

	w := authReq(t, r, http.MethodGet, "/api/admin/history", nil, token)
	var sessions []map[string]any
	json.Unmarshal(w.Body.Bytes(), &sessions) //nolint:errcheck

	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions (APPROVED+REJECTED+PENDING) in history, got %d", len(sessions))
	}

	statuses := map[string]int{}
	for _, s := range sessions {
		statuses[s["status"].(string)]++
	}
	if statuses["APPROVED"] != 1 || statuses["REJECTED"] != 1 || statuses["PENDING"] != 1 {
		t.Errorf("unexpected status distribution: %v", statuses)
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

// --- GET /api/admin/children ---

func TestListChildren_RequiresAuth(t *testing.T) {
	r := setupTestRouter(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/children", nil))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestListChildren_ReturnsSeededChild(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	w := authReq(t, r, http.MethodGet, "/api/admin/children", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var children []map[string]any
	json.Unmarshal(w.Body.Bytes(), &children) //nolint:errcheck
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if children[0]["name"] != "Speler" {
		t.Errorf("expected name=Speler, got %v", children[0]["name"])
	}
	if _, ok := children[0]["total_points"]; !ok {
		t.Error("expected total_points field")
	}
}

// --- PATCH /api/admin/children/:id ---

func TestRenameChild_HappyPath(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	w := authReq(t, r, http.MethodPatch, "/api/admin/children/1", map[string]any{"name": "Lena"}, token)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var name string
	db.DB.QueryRow(`SELECT name FROM children WHERE id = 1`).Scan(&name) //nolint:errcheck
	if name != "Lena" {
		t.Errorf("expected name=Lena in DB, got %q", name)
	}
}

func TestRenameChild_NotFound(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	w := authReq(t, r, http.MethodPatch, "/api/admin/children/999", map[string]any{"name": "X"}, token)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- GET /api/admin/admins ---

func TestListAdmins_ReturnsSeededAdmin(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	w := authReq(t, r, http.MethodGet, "/api/admin/admins", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var admins []map[string]any
	json.Unmarshal(w.Body.Bytes(), &admins) //nolint:errcheck
	if len(admins) != 1 {
		t.Fatalf("expected 1 admin, got %d", len(admins))
	}
	if admins[0]["username"] != "admin" {
		t.Errorf("expected username=admin, got %v", admins[0]["username"])
	}
	if _, ok := admins[0]["password_hash"]; ok {
		t.Error("password_hash must not be exposed")
	}
}

// --- POST /api/admin/admins ---

func TestCreateAdmin_HappyPath(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	w := authReq(t, r, http.MethodPost, "/api/admin/admins", map[string]any{
		"username": "jan",
		"password": "geheim123",
	}, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
	if resp["username"] != "jan" {
		t.Errorf("expected username=jan, got %v", resp["username"])
	}
	if _, ok := resp["id"]; !ok {
		t.Error("expected id in response")
	}
}

func TestCreateAdmin_DuplicateUsername(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	authReq(t, r, http.MethodPost, "/api/admin/admins", map[string]any{
		"username": "admin",
		"password": "other",
	}, token)

	w := authReq(t, r, http.MethodPost, "/api/admin/admins", map[string]any{
		"username": "admin",
		"password": "other",
	}, token)
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

// --- DELETE /api/admin/admins/:id ---

func TestDeleteAdmin_HappyPath(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	// Create a second admin to delete
	cw := authReq(t, r, http.MethodPost, "/api/admin/admins", map[string]any{
		"username": "tobedeleted",
		"password": "pass",
	}, token)
	var created map[string]any
	json.Unmarshal(cw.Body.Bytes(), &created) //nolint:errcheck
	newID := itoa(int(created["id"].(float64)))

	w := authReq(t, r, http.MethodDelete, "/api/admin/admins/"+newID, nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var count int
	db.DB.QueryRow(`SELECT COUNT(*) FROM admins WHERE id = ?`, newID).Scan(&count) //nolint:errcheck
	if count != 0 {
		t.Error("expected admin to be deleted from DB")
	}
}

func TestDeleteAdmin_CannotDeleteSelf(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	w := authReq(t, r, http.MethodDelete, "/api/admin/admins/1", nil, token)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestDeleteAdmin_NotFound(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	w := authReq(t, r, http.MethodDelete, "/api/admin/admins/9999", nil, token)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- PATCH /api/admin/admins/:id ---

func TestUpdateAdmin_ChangeUsername(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	// Create a second admin to update
	cw := authReq(t, r, http.MethodPost, "/api/admin/admins", map[string]any{
		"username": "original",
		"password": "pass",
	}, token)
	var created map[string]any
	json.Unmarshal(cw.Body.Bytes(), &created) //nolint:errcheck
	newID := itoa(int(created["id"].(float64)))

	w := authReq(t, r, http.MethodPatch, "/api/admin/admins/"+newID, map[string]any{
		"username": "renamed",
	}, token)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var username string
	db.DB.QueryRow(`SELECT username FROM admins WHERE id = ?`, newID).Scan(&username) //nolint:errcheck
	if username != "renamed" {
		t.Errorf("expected username=renamed, got %q", username)
	}
}

func TestUpdateAdmin_ChangePassword(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	w := authReq(t, r, http.MethodPatch, "/api/admin/admins/1", map[string]any{
		"password": "nieuwwachtwoord",
	}, token)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// New password should work for login
	lw := postJSON(t, r, "/api/auth/login", map[string]any{
		"username": "admin",
		"password": "nieuwwachtwoord",
	})
	if lw.Code != http.StatusOK {
		t.Errorf("expected login with new password to succeed, got %d", lw.Code)
	}
}

func TestUpdateAdmin_NoFields(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	w := authReq(t, r, http.MethodPatch, "/api/admin/admins/1", map[string]any{}, token)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdateAdmin_NotFound(t *testing.T) {
	r := setupTestRouter(t)
	token := loginAsAdmin(t, r)

	w := authReq(t, r, http.MethodPatch, "/api/admin/admins/9999", map[string]any{"username": "x"}, token)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
