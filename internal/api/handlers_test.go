package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"

	"github.com/example/go-wasm-sqlite-poc/internal/database"
)

func newTestHandlers(t *testing.T) *Handlers {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.InitSchema(db); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return NewHandlers(db, "")
}

func TestAPIListAndCreate(t *testing.T) {
	h := newTestHandlers(t)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/shortcuts", h.ListShortcuts)
	mux.HandleFunc("POST /api/shortcuts", h.CreateShortcut)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/shortcuts", strings.NewReader(`{"name":"gh","url":"https://github.com"}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status %d: %s", rr.Code, rr.Body.String())
	}

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/shortcuts", nil)
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("list status %d", rr2.Code)
	}
	var list []database.Shortcut
	if err := json.Unmarshal(rr2.Body.Bytes(), &list); err != nil || len(list) != 1 {
		t.Fatalf("list parse err=%v len=%d", err, len(list))
	}
}

func TestAPIResolveAndSearch(t *testing.T) {
	h := newTestHandlers(t)
	_, _ = database.CreateShortcut(h.DB, "google", "https://www.google.com", "https://www.google.com/search?q=", "Google")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/resolve/{name}", h.ResolveShortcut)
	mux.HandleFunc("GET /api/search", h.SearchShortcuts)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/resolve/google%20maps", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("resolve status %d: %s", rr.Code, rr.Body.String())
	}

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/search?q=goog", nil)
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("search status %d", rr2.Code)
	}
}

func TestAPIDeleteNotFound(t *testing.T) {
	h := newTestHandlers(t)
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/shortcuts/{id}", h.DeleteShortcut)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/shortcuts/999", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}
