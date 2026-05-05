package db

import (
	"fmt"
	"strings"
)

// HistoryEntry represents a single command in the history.
type HistoryEntry struct {
	ID        int64
	Command   string
	Timestamp int64
	Duration  int
	Directory string
	ExitCode  int
	Deleted   bool
}

// InsertHistory inserts a single history entry.
func (s *Store) InsertHistory(entry HistoryEntry) error {
	_, err := s.db.Exec(
		`INSERT INTO history (command, timestamp, duration, directory, exit_code)
		 VALUES (?, ?, ?, ?, ?)`,
		entry.Command, entry.Timestamp, entry.Duration, entry.Directory, entry.ExitCode,
	)
	return err
}

// BulkInsertHistory inserts multiple entries in a transaction, skipping duplicates.
// Returns the number of entries actually inserted.
func (s *Store) BulkInsertHistory(entries []HistoryEntry) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT INTO history (command, timestamp, duration, directory, exit_code)
		 SELECT ?, ?, ?, ?, ?
		 WHERE NOT EXISTS (
		   SELECT 1 FROM history WHERE command = ? AND timestamp = ?
		 )`,
	)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	count := 0
	for _, e := range entries {
		result, err := stmt.Exec(e.Command, e.Timestamp, e.Duration, e.Directory, e.ExitCode, e.Command, e.Timestamp)
		if err != nil {
			return count, fmt.Errorf("insert entry: %w", err)
		}
		rows, _ := result.RowsAffected()
		if rows > 0 {
			count++
		}
	}

	return count, tx.Commit()
}

// ListHistory returns the most recent non-deleted history entries, deduplicated by command.
// It returns the most recent occurrence of each unique command.
func (s *Store) ListHistory(limit int) ([]HistoryEntry, error) {
	rows, err := s.db.Query(
		`SELECT id, command, timestamp, duration, directory, exit_code
		 FROM history
		 WHERE deleted = 0
		 AND id IN (
		   SELECT MAX(id) FROM history WHERE deleted = 0 GROUP BY command
		 )
		 ORDER BY timestamp DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		if err := rows.Scan(&e.ID, &e.Command, &e.Timestamp, &e.Duration, &e.Directory, &e.ExitCode); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// SearchHistory performs a LIKE search on command text.
func (s *Store) SearchHistory(query string, limit int) ([]HistoryEntry, error) {
	pattern := "%" + strings.ReplaceAll(query, "%", "\\%") + "%"
	rows, err := s.db.Query(
		`SELECT id, command, timestamp, duration, directory, exit_code
		 FROM history
		 WHERE deleted = 0 AND command LIKE ? ESCAPE '\'
		 AND id IN (
		   SELECT MAX(id) FROM history WHERE deleted = 0 GROUP BY command
		 )
		 ORDER BY timestamp DESC
		 LIMIT ?`,
		pattern, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		if err := rows.Scan(&e.ID, &e.Command, &e.Timestamp, &e.Duration, &e.Directory, &e.ExitCode); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ListHistoryFrecency returns history entries scored by frecency (frequency * recency).
// The scoring formula is: count * (1 / (1 + (now - last_used) / 86400))
// This means frequently used recent commands rank highest.
func (s *Store) ListHistoryFrecency(now int64, limit int) ([]HistoryEntry, error) {
	rows, err := s.db.Query(
		`SELECT h.id, h.command, h.timestamp, h.duration, h.directory, h.exit_code
		 FROM history h
		 INNER JOIN (
		   SELECT command,
		          MAX(id) AS max_id,
		          COUNT(*) AS frequency,
		          MAX(timestamp) AS last_used
		   FROM history
		   WHERE deleted = 0
		   GROUP BY command
		 ) stats ON h.id = stats.max_id
		 ORDER BY stats.frequency * (1.0 / (1.0 + (CAST(? AS REAL) - stats.last_used) / 86400.0)) DESC
		 LIMIT ?`,
		now, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		if err := rows.Scan(&e.ID, &e.Command, &e.Timestamp, &e.Duration, &e.Directory, &e.ExitCode); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// SoftDeleteHistory marks a history entry as deleted.
func (s *Store) SoftDeleteHistory(id int64) error {
	_, err := s.db.Exec(`UPDATE history SET deleted = 1 WHERE id = ?`, id)
	return err
}

// SoftDeleteByCommand marks all history entries matching a command as deleted.
func (s *Store) SoftDeleteByCommand(command string) error {
	_, err := s.db.Exec(`UPDATE history SET deleted = 1 WHERE command = ?`, command)
	return err
}

// RestoreByCommand un-deletes all history entries matching a command.
func (s *Store) RestoreByCommand(command string) error {
	_, err := s.db.Exec(`UPDATE history SET deleted = 0 WHERE command = ?`, command)
	return err
}

// UpdateHistoryCommand updates the command text of a history entry.
func (s *Store) UpdateHistoryCommand(id int64, newCommand string) error {
	_, err := s.db.Exec(`UPDATE history SET command = ? WHERE id = ?`, newCommand, id)
	return err
}
