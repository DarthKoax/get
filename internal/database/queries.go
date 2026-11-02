package database

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var nameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

var (
	ErrNotFound  = errors.New("shortcut not found")
	ErrDuplicate = errors.New("shortcut name already exists")
)

// ValidationError represents a client-facing validation failure.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// IsValidationError reports whether err is a validation error.
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

type Shortcut struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	SearchURL   string `json:"search_url"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

func ValidateName(name string) error {
	if name == "" {
		return &ValidationError{Message: "name is required"}
	}
	if len(name) > 64 {
		return &ValidationError{Message: "name must be at most 64 characters"}
	}
	if !nameRegex.MatchString(name) {
		return &ValidationError{Message: "invalid name format"}
	}
	return nil
}

func ValidateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return &ValidationError{Message: "invalid URL"}
	}
	if u.Scheme == "" {
		return &ValidationError{Message: "invalid URL: scheme is required"}
	}
	// For http/https, ensure host is present
	if u.Scheme == "http" || u.Scheme == "https" {
		if u.Host == "" {
			return &ValidationError{Message: "invalid URL: host is required for http/https"}
		}
	}
	// Allow other schemes (chrome://, mailto:, tel:, etc.) without host requirement
	return nil
}

func validateShortcutFields(name, urlStr, searchURL, desc string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	if err := ValidateURL(urlStr); err != nil {
		return err
	}
	if searchURL != "" {
		if err := ValidateURL(searchURL); err != nil {
			return fmt.Errorf("search_url: %w", err)
		}
	}
	if len(desc) > 256 {
		return &ValidationError{Message: "description must be at most 256 characters"}
	}
	return nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique")
}

func CreateShortcut(db *sql.DB, name, urlStr, searchURL, desc string) (Shortcut, error) {
	name = strings.TrimSpace(name)
	urlStr = strings.TrimSpace(urlStr)
	searchURL = strings.TrimSpace(searchURL)
	desc = strings.TrimSpace(desc)

	if err := validateShortcutFields(name, urlStr, searchURL, desc); err != nil {
		return Shortcut{}, err
	}

	res, err := db.Exec("INSERT INTO shortcuts(name, url, search_url, description) VALUES(?, ?, ?, ?)", name, urlStr, searchURL, desc)
	if err != nil {
		if isUniqueViolation(err) {
			return Shortcut{}, ErrDuplicate
		}
		return Shortcut{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Shortcut{}, err
	}
	var createdAt string
	if err := db.QueryRow("SELECT created_at FROM shortcuts WHERE id = ?", id).Scan(&createdAt); err != nil {
		return Shortcut{}, err
	}
	return Shortcut{
		ID:          int(id),
		Name:        name,
		URL:         urlStr,
		SearchURL:   searchURL,
		Description: desc,
		CreatedAt:   createdAt,
	}, nil
}

func ListShortcuts(db *sql.DB) ([]Shortcut, error) {
	rows, err := db.Query("SELECT id, name, url, search_url, description, created_at FROM shortcuts ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	shortcuts := []Shortcut{}
	for rows.Next() {
		var s Shortcut
		if err := rows.Scan(&s.ID, &s.Name, &s.URL, &s.SearchURL, &s.Description, &s.CreatedAt); err != nil {
			return nil, err
		}
		shortcuts = append(shortcuts, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return shortcuts, nil
}

func GetShortcut(db *sql.DB, id int) (Shortcut, error) {
	var s Shortcut
	err := db.QueryRow("SELECT id, name, url, search_url, description, created_at FROM shortcuts WHERE id = ?", id).
		Scan(&s.ID, &s.Name, &s.URL, &s.SearchURL, &s.Description, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return Shortcut{}, ErrNotFound
	}
	if err != nil {
		return Shortcut{}, err
	}
	return s, nil
}

func UpdateShortcut(db *sql.DB, id int, name, urlStr, searchURL, desc string) (Shortcut, error) {
	name = strings.TrimSpace(name)
	urlStr = strings.TrimSpace(urlStr)
	searchURL = strings.TrimSpace(searchURL)
	desc = strings.TrimSpace(desc)

	if err := validateShortcutFields(name, urlStr, searchURL, desc); err != nil {
		return Shortcut{}, err
	}

	res, err := db.Exec("UPDATE shortcuts SET name = ?, url = ?, search_url = ?, description = ? WHERE id = ?", name, urlStr, searchURL, desc, id)
	if err != nil {
		if isUniqueViolation(err) {
			return Shortcut{}, ErrDuplicate
		}
		return Shortcut{}, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return Shortcut{}, ErrNotFound
	}
	var updatedAt string
	if err := db.QueryRow("SELECT datetime('now')").Scan(&updatedAt); err != nil {
		return Shortcut{}, err
	}
	return Shortcut{
		ID:          id,
		Name:        name,
		URL:         urlStr,
		SearchURL:   searchURL,
		Description: desc,
		CreatedAt:   updatedAt,
	}, nil
}

func DeleteShortcut(db *sql.DB, id int) error {
	res, err := db.Exec("DELETE FROM shortcuts WHERE id = ?", id)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func ResolveShortcut(db *sql.DB, name string) (string, string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", "", &ValidationError{Message: "shortcut name is required"}
	}
	normalized := strings.ToLower(trimmed)
	var u, searchURL string
	err := db.QueryRow("SELECT url, search_url FROM shortcuts WHERE name = ?", normalized).Scan(&u, &searchURL)
	if err == nil {
		return u, searchURL, nil
	}
	if err != sql.ErrNoRows {
		return "", "", err
	}
	parts := strings.SplitN(trimmed, " ", 2)
	if len(parts) == 2 {
		engine := strings.ToLower(parts[0])
		query := parts[1]
		var engineURL, engineSearchURL string
		err := db.QueryRow("SELECT url, search_url FROM shortcuts WHERE name = ?", engine).Scan(&engineURL, &engineSearchURL)
		if err == nil && engineSearchURL != "" {
			resolvedURL := engineSearchURL + url.PathEscape(query)
			return resolvedURL, "", nil
		}
	}
	return "", "", ErrNotFound
}

func SearchShortcuts(db *sql.DB, query string) ([]Shortcut, error) {
	var rows *sql.Rows
	var err error
	if strings.TrimSpace(query) == "" {
		rows, err = db.Query("SELECT id, name, url, search_url, description, created_at FROM shortcuts ORDER BY name")
	} else {
		likeQuery := "%" + query + "%"
		rows, err = db.Query(`SELECT id, name, url, search_url, description, created_at FROM shortcuts 
			WHERE LOWER(name) LIKE LOWER(?) 
			OR LOWER(url) LIKE LOWER(?) 
			OR LOWER(search_url) LIKE LOWER(?) 
			OR LOWER(description) LIKE LOWER(?) 
			ORDER BY name`, likeQuery, likeQuery, likeQuery, likeQuery)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	shortcuts := []Shortcut{}
	for rows.Next() {
		var s Shortcut
		if err := rows.Scan(&s.ID, &s.Name, &s.URL, &s.SearchURL, &s.Description, &s.CreatedAt); err != nil {
			return nil, err
		}
		shortcuts = append(shortcuts, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return shortcuts, nil
}
