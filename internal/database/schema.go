package database

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed shortcuts.json
var shortcutsConfigFS embed.FS

const CreateSchemaSQL = `CREATE TABLE IF NOT EXISTS shortcuts (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	name        TEXT    NOT NULL UNIQUE,
	url         TEXT    NOT NULL,
	search_url  TEXT    NOT NULL DEFAULT '',
	description TEXT    NOT NULL DEFAULT '',
	created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
)`

// ShortcutConfig represents a single shortcut entry in shortcuts.json.
type ShortcutConfig struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	SearchURL   string `json:"search_url,omitempty"`
	Description string `json:"description"`
}

// Config represents the top-level structure of shortcuts.json.
type Config struct {
	Shortcuts       []ShortcutConfig `json:"shortcuts"`
	DefaultResolver string          `json:"default_resolver,omitempty"`
}

// InitSchema creates the shortcuts table if it does not exist.
func InitSchema(db *sql.DB) error {
	_, err := db.Exec(CreateSchemaSQL)
	return err
}

// SeedDB reads the embedded shortcuts.json and inserts all entries into the
// shortcuts table. It is called once during initialization.
func SeedDB(db *sql.DB) error {
	data, err := shortcutsConfigFS.ReadFile("shortcuts.json")
	if err != nil {
		return fmt.Errorf("read embedded config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	for _, s := range cfg.Shortcuts {
		if err := ValidateName(s.Name); err != nil {
			return fmt.Errorf("seed shortcut %s: %w", s.Name, err)
		}
		if err := ValidateURL(s.URL); err != nil {
			return fmt.Errorf("seed shortcut %s: %w", s.Name, err)
		}
		if s.SearchURL != "" {
			if err := ValidateURL(s.SearchURL); err != nil {
				return fmt.Errorf("seed shortcut %s search_url: %w", s.Name, err)
			}
		}
		if _, err := db.Exec(
			"INSERT INTO shortcuts(name, url, search_url, description) VALUES(?, ?, ?, ?)",
			s.Name, s.URL, s.SearchURL, s.Description,
		); err != nil {
			return fmt.Errorf("seed shortcut %s: %w", s.Name, err)
		}
	}
	return nil
}

// GetDefaultResolver reads the default_resolver from the embedded shortcuts.json.
func GetDefaultResolver() (string, error) {
	data, err := shortcutsConfigFS.ReadFile("shortcuts.json")
	if err != nil {
		return "", fmt.Errorf("read embedded config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("parse config: %w", err)
	}
	return cfg.DefaultResolver, nil
}

// SeedIfEmpty seeds the database only when it contains no shortcuts.
func SeedIfEmpty(db *sql.DB) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM shortcuts").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return SeedDB(db)
}
