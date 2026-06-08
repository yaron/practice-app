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
	"testing"

	"violin-quest-api/db"
	"violin-quest-api/handlers"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	log.SetOutput(io.Discard) // suppress db/seed startup logs
	os.Exit(m.Run())
}

// setupTestRouter creates a fresh in-memory SQLite DB, runs migrations and seed,
// then returns a Gin router wired to the real handlers.
// Each test gets an isolated DB; sequential execution is assumed (no t.Parallel).
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

	return r
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
	// Verify options are scoped to child_id=1
	for _, o := range options {
		if o["child_id"].(float64) != 1 {
			t.Errorf("option has wrong child_id: %v", o["child_id"])
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

	requiredFields := []string{"current_level", "experience_points", "total_points", "shield_count", "week_session_count", "milestone_reached", "year_week"}
	for _, field := range requiredFields {
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

func postJSON(t *testing.T, r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

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
	db.DB.QueryRow(`SELECT COUNT(*) FROM sessions WHERE child_id=1 AND tasks_completed=4 AND status='PENDING'`).Scan(&count)
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
