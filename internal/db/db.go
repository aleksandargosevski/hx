package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Store wraps the SQLite database connection.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) the hx database in ~/.config/hx/hx.db.
func Open() (*Store, error) {
	dbPath, err := dbPath()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return store, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying sql.DB for advanced usage.
func (s *Store) DB() *sql.DB {
	return s.db
}

func dbPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}

	dir := filepath.Join(home, ".config", "hx")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}

	return filepath.Join(dir, "hx.db"), nil
}

func (s *Store) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS history (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			command   TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			duration  INTEGER DEFAULT 0,
			directory TEXT DEFAULT '',
			exit_code INTEGER DEFAULT 0,
			deleted   INTEGER DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_history_ts ON history(timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_history_cmd ON history(command)`,
		`CREATE INDEX IF NOT EXISTS idx_history_dir ON history(directory, deleted)`,
		`CREATE TABLE IF NOT EXISTS templates (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT NOT NULL UNIQUE,
			command     TEXT NOT NULL,
			description TEXT DEFAULT '',
			created_at  INTEGER NOT NULL,
			updated_at  INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER NOT NULL
		)`,
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, m := range migrations {
		if _, err := tx.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %s: %w", m[:50], err)
		}
	}

	return tx.Commit()
}
