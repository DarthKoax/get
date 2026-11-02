package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/example/go-wasm-sqlite-poc/internal/database"
)

type Handlers struct {
	DB              *sql.DB
	DBPath          string
	DefaultResolver string
}

func NewHandlers(db *sql.DB, dbPath string, defaultResolver string) *Handlers {
	return &Handlers{DB: db, DBPath: dbPath, DefaultResolver: defaultResolver}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("JSON encode error: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// clientError returns a safe, client-facing message for an error and logs the
// real error server-side when it is not a controlled validation/not-found error.
func clientError(w http.ResponseWriter, status int, err error) {
	if status >= 500 {
		log.Printf("internal error: %v", err)
		writeError(w, status, "internal server error")
		return
	}
	if database.IsValidationError(err) {
		writeError(w, status, err.Error())
		return
	}
	// Controlled sentinel errors (not found, duplicate) keep their messages.
	if err == database.ErrNotFound || err == database.ErrDuplicate {
		writeError(w, status, err.Error())
		return
	}
	// Other 4xx errors: avoid leaking raw driver details.
	log.Printf("request error: %v", err)
	writeError(w, status, "bad request")
}

func (h *Handlers) ListShortcuts(w http.ResponseWriter, r *http.Request) {
	shortcuts, err := database.ListShortcuts(h.DB)
	if err != nil {
		clientError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, shortcuts)
}

func (h *Handlers) CreateShortcut(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var body struct {
		Name        string `json:"name"`
		URL         string `json:"url"`
		SearchURL   string `json:"search_url"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	s, err := database.CreateShortcut(h.DB, body.Name, body.URL, body.SearchURL, body.Description)
	if err != nil {
		status := http.StatusBadRequest
		if err == database.ErrDuplicate {
			status = http.StatusConflict
		}
		if err != database.ErrDuplicate && !database.IsValidationError(err) {
			status = http.StatusInternalServerError
		}
		clientError(w, status, err)
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

func (h *Handlers) UpdateShortcut(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var body struct {
		Name        string `json:"name"`
		URL         string `json:"url"`
		SearchURL   string `json:"search_url"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	s, err := database.UpdateShortcut(h.DB, id, body.Name, body.URL, body.SearchURL, body.Description)
	if err != nil {
		status := http.StatusBadRequest
		if err == database.ErrNotFound {
			status = http.StatusNotFound
		} else if err == database.ErrDuplicate {
			status = http.StatusConflict
		}
		if err != database.ErrNotFound && err != database.ErrDuplicate && !database.IsValidationError(err) {
			status = http.StatusInternalServerError
		}
		clientError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *Handlers) DeleteShortcut(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := database.DeleteShortcut(h.DB, id); err != nil {
		status := http.StatusInternalServerError
		if err == database.ErrNotFound {
			status = http.StatusNotFound
		}
		clientError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"id": id})
}

func (h *Handlers) ResolveShortcut(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	u, searchURL, err := database.ResolveShortcut(h.DB, name)
	if err != nil {
		// If shortcut not found and default resolver is configured, try resolving with default resolver
		if err == database.ErrNotFound && h.DefaultResolver != "" {
			defaultName := h.DefaultResolver + " " + name
			u, searchURL, err = database.ResolveShortcut(h.DB, defaultName)
			if err == nil {
				// Successfully resolved with default resolver
				writeJSON(w, http.StatusOK, map[string]string{"url": u, "search_url": searchURL})
				return
			}
			// If default resolver also fails, return the original not found error
			err = database.ErrNotFound
		}
		
		status := http.StatusNotFound
		if database.IsValidationError(err) {
			status = http.StatusBadRequest
		} else if err != database.ErrNotFound {
			status = http.StatusInternalServerError
		}
		clientError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": u, "search_url": searchURL})
}

func (h *Handlers) SearchShortcuts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	shortcuts, err := database.SearchShortcuts(h.DB, q)
	if err != nil {
		clientError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, shortcuts)
}

func (h *Handlers) DownloadDatabase(w http.ResponseWriter, r *http.Request) {
	if h.DBPath == "" {
		writeError(w, http.StatusInternalServerError, "database path not configured")
		return
	}
	f, err := os.Open(h.DBPath)
	if err != nil {
		log.Printf("download open error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		log.Printf("download stat error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\"shortcuts.db\"")
	http.ServeContent(w, r, "shortcuts.db", info.ModTime(), f)
}
