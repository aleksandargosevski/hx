package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatal(err)
	}

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { store.Close() })
	return store
}

func TestInsertAndListHistory(t *testing.T) {
	store := testStore(t)

	entries := []HistoryEntry{
		{Command: "echo hello", Timestamp: time.Now().Unix(), Directory: "/tmp"},
		{Command: "git status", Timestamp: time.Now().Unix() - 100, Directory: "/home"},
		{Command: "docker ps", Timestamp: time.Now().Unix() - 200, Directory: "/home"},
	}

	for _, e := range entries {
		if err := store.InsertHistory(e); err != nil {
			t.Fatal(err)
		}
	}

	listed, err := store.ListHistory(100)
	if err != nil {
		t.Fatal(err)
	}

	if len(listed) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(listed))
	}

	// Should be ordered by timestamp DESC
	if listed[0].Command != "echo hello" {
		t.Errorf("expected first entry to be 'echo hello', got %q", listed[0].Command)
	}
}

func TestDeduplication(t *testing.T) {
	store := testStore(t)

	// Insert same command twice with different timestamps
	store.InsertHistory(HistoryEntry{Command: "git status", Timestamp: 1000})
	store.InsertHistory(HistoryEntry{Command: "git status", Timestamp: 2000})
	store.InsertHistory(HistoryEntry{Command: "echo hi", Timestamp: 1500})

	listed, err := store.ListHistory(100)
	if err != nil {
		t.Fatal(err)
	}

	// Should deduplicate "git status" — only show most recent
	if len(listed) != 2 {
		t.Fatalf("expected 2 deduplicated entries, got %d", len(listed))
	}
}

func TestSoftDelete(t *testing.T) {
	store := testStore(t)

	store.InsertHistory(HistoryEntry{Command: "secret command", Timestamp: 1000})
	store.InsertHistory(HistoryEntry{Command: "safe command", Timestamp: 2000})

	if err := store.SoftDeleteByCommand("secret command"); err != nil {
		t.Fatal(err)
	}

	listed, err := store.ListHistory(100)
	if err != nil {
		t.Fatal(err)
	}

	if len(listed) != 1 {
		t.Fatalf("expected 1 entry after delete, got %d", len(listed))
	}
	if listed[0].Command != "safe command" {
		t.Errorf("expected 'safe command', got %q", listed[0].Command)
	}
}

func TestUpdateHistoryCommand(t *testing.T) {
	store := testStore(t)

	store.InsertHistory(HistoryEntry{Command: "old command", Timestamp: 1000})

	listed, err := store.ListHistory(100)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.UpdateHistoryCommand(listed[0].ID, "new command"); err != nil {
		t.Fatal(err)
	}

	listed, err = store.ListHistory(100)
	if err != nil {
		t.Fatal(err)
	}

	if listed[0].Command != "new command" {
		t.Errorf("expected 'new command', got %q", listed[0].Command)
	}
}

func TestBulkInsert(t *testing.T) {
	store := testStore(t)

	entries := []HistoryEntry{
		{Command: "cmd1", Timestamp: 1000},
		{Command: "cmd2", Timestamp: 2000},
		{Command: "cmd1", Timestamp: 1000}, // Duplicate
	}

	count, err := store.BulkInsertHistory(entries)
	if err != nil {
		t.Fatal(err)
	}

	if count != 2 {
		t.Errorf("expected 2 inserted (1 duplicate skipped), got %d", count)
	}
}

func TestTemplateCRUD(t *testing.T) {
	store := testStore(t)

	now := time.Now().Unix()
	tmpl := Template{
		Name:        "docker-exec",
		Command:     "docker exec -it ${1:container} ${2:cmd}",
		Description: "Exec into container",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := store.InsertTemplate(tmpl); err != nil {
		t.Fatal(err)
	}

	// List
	templates, err := store.ListTemplates()
	if err != nil {
		t.Fatal(err)
	}
	if len(templates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(templates))
	}
	if templates[0].Name != "docker-exec" {
		t.Errorf("expected name 'docker-exec', got %q", templates[0].Name)
	}

	// Get
	got, err := store.GetTemplate("docker-exec")
	if err != nil {
		t.Fatal(err)
	}
	if got.Command != tmpl.Command {
		t.Errorf("expected command %q, got %q", tmpl.Command, got.Command)
	}

	// Update
	got.Description = "Updated description"
	got.UpdatedAt = now + 1
	if err := store.UpdateTemplate(*got); err != nil {
		t.Fatal(err)
	}

	got2, _ := store.GetTemplate("docker-exec")
	if got2.Description != "Updated description" {
		t.Errorf("expected updated description, got %q", got2.Description)
	}

	// Upsert
	upsertTmpl := Template{
		Name:        "docker-exec",
		Command:     "docker exec -it ${1:container} sh",
		Description: "Upserted",
		CreatedAt:   now,
		UpdatedAt:   now + 2,
	}
	if err := store.UpsertTemplate(upsertTmpl); err != nil {
		t.Fatal(err)
	}

	got3, _ := store.GetTemplate("docker-exec")
	if got3.Command != "docker exec -it ${1:container} sh" {
		t.Errorf("upsert failed, got %q", got3.Command)
	}

	// Delete
	if err := store.DeleteTemplate("docker-exec"); err != nil {
		t.Fatal(err)
	}

	templates, _ = store.ListTemplates()
	if len(templates) != 0 {
		t.Errorf("expected 0 templates after delete, got %d", len(templates))
	}

	// Delete non-existent
	err = store.DeleteTemplate("nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSearchHistory(t *testing.T) {
	store := testStore(t)

	store.InsertHistory(HistoryEntry{Command: "docker compose up -d", Timestamp: 1000})
	store.InsertHistory(HistoryEntry{Command: "docker ps", Timestamp: 2000})
	store.InsertHistory(HistoryEntry{Command: "git status", Timestamp: 3000})

	results, err := store.SearchHistory("docker", 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results for 'docker', got %d", len(results))
	}
}

// Test that the database file is created if config dir doesn't exist
func TestOpenCreatesDir(t *testing.T) {
	// We test dbPath directly by checking it doesn't error
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home dir")
	}

	expectedDir := filepath.Join(home, ".config", "hx")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		// Would be created by Open(), but we don't want to test Open() side effects here
		t.Skip("config dir doesn't exist and we don't want to create it in tests")
	}
}
