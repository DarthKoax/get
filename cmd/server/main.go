package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/ncruces/go-sqlite3/driver"

	"github.com/example/go-wasm-sqlite-poc/internal/api"
	"github.com/example/go-wasm-sqlite-poc/internal/config"
	"github.com/example/go-wasm-sqlite-poc/internal/database"
)

func main() {
	cfgPath := "build.env"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("config load: %v", err)
	}

	// Ensure data directory exists with restrictive permissions
	if dir := filepath.Dir(cfg.DBPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			log.Fatalf("create data dir: %v", err)
		}
	}

	db, err := sql.Open("sqlite3", cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := database.InitSchema(db); err != nil {
		log.Fatalf("init schema: %v", err)
	}
	if err := database.SeedIfEmpty(db); err != nil {
		log.Printf("seed warning: %v", err)
	}

	handlers := api.NewHandlers(db, cfg.DBPath, cfg.DefaultResolver)

	mux := http.NewServeMux()
	prefix := strings.TrimRight(cfg.APIPrefix, "/")
	mux.HandleFunc("GET "+prefix+"/shortcuts", handlers.ListShortcuts)
	mux.HandleFunc("POST "+prefix+"/shortcuts", handlers.CreateShortcut)
	mux.HandleFunc("PUT "+prefix+"/shortcuts/{id}", handlers.UpdateShortcut)
	mux.HandleFunc("DELETE "+prefix+"/shortcuts/{id}", handlers.DeleteShortcut)
	mux.HandleFunc("GET "+prefix+"/resolve/{name}", handlers.ResolveShortcut)
	mux.HandleFunc("GET "+prefix+"/search", handlers.SearchShortcuts)
	mux.HandleFunc("GET "+prefix+"/download", handlers.DownloadDatabase)

	// Serve static frontend
	fs := http.FileServer(http.Dir("web"))
	mux.Handle("/", fs)

	// CORS middleware (configurable via CORS_ALLOWED_ORIGINS env var)
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = os.Getenv("CORS_ORIGINS")
	}
	handler := corsMiddleware(mux, allowedOrigins)

	addr := ":" + cfg.ServerPort
	log.Printf("Server mode listening on %s (db=%s)", addr, cfg.DBPath)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func corsMiddleware(next http.Handler, allowedOrigins string) http.Handler {
	origins := map[string]bool{}
	for _, o := range strings.Split(allowedOrigins, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins[o] = true
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && origins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			if origin == "" || origins[origin] {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.WriteHeader(http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
