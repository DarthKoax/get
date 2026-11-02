package database

import (
	"database/sql"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := InitSchema(db); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestCreateAndList(t *testing.T) {
	db := openTestDB(t)
	s, err := CreateShortcut(db, "gh", "https://github.com", "", "GitHub")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "gh" {
		t.Fatalf("unexpected name %s", s.Name)
	}
	list, err := ListShortcuts(db)
	if err != nil || len(list) != 1 {
		t.Fatalf("list err=%v len=%d", err, len(list))
	}
}

func TestDuplicate(t *testing.T) {
	db := openTestDB(t)
	_, _ = CreateShortcut(db, "gh", "https://github.com", "", "")
	if _, err := CreateShortcut(db, "gh", "https://github.com", "", ""); err != ErrDuplicate {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestUpdateDelete(t *testing.T) {
	db := openTestDB(t)
	s, _ := CreateShortcut(db, "gh", "https://github.com", "", "")
	upd, err := UpdateShortcut(db, s.ID, "gh", "https://github.com/x", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if upd.URL != "https://github.com/x" {
		t.Fatalf("update failed")
	}
	if err := DeleteShortcut(db, s.ID); err != nil {
		t.Fatal(err)
	}
	if err := DeleteShortcut(db, s.ID); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestResolveSearchFallback(t *testing.T) {
	db := openTestDB(t)
	_, _ = CreateShortcut(db, "google", "https://www.google.com", "https://www.google.com/search?q=", "Google")
	u, su, err := ResolveShortcut(db, "google maps")
	if err != nil {
		t.Fatal(err)
	}
	if u != "https://www.google.com/search?q=maps" || su != "" {
		t.Fatalf("unexpected resolve %s %s", u, su)
	}
}

func TestSearch(t *testing.T) {
	db := openTestDB(t)
	_, _ = CreateShortcut(db, "gh", "https://github.com", "", "Code hosting platform")
	_, _ = CreateShortcut(db, "gitlab", "https://gitlab.com", "", "GitLab CI/CD")
	
	// Test search by name
	res, err := SearchShortcuts(db, "git")
	if err != nil || len(res) != 2 {
		t.Fatalf("search by name err=%v len=%d", err, len(res))
	}
	
	// Test search by description
	res, err = SearchShortcuts(db, "CI/CD")
	if err != nil || len(res) != 1 {
		t.Fatalf("search by description err=%v len=%d", err, len(res))
	}
	if len(res) == 1 && res[0].Name != "gitlab" {
		t.Fatalf("expected gitlab, got %s", res[0].Name)
	}
	
	// Test search by URL
	res, err = SearchShortcuts(db, "github.com")
	if err != nil || len(res) != 1 {
		t.Fatalf("search by URL err=%v len=%d", err, len(res))
	}
	if len(res) == 1 && res[0].Name != "gh" {
		t.Fatalf("expected gh, got %s", res[0].Name)
	}
}
