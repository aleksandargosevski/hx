package db

// Template represents a reusable command template with placeholders.
type Template struct {
	ID          int64
	Name        string
	Command     string
	Description string
	CreatedAt   int64
	UpdatedAt   int64
}

// InsertTemplate inserts a new template.
func (s *Store) InsertTemplate(t Template) error {
	_, err := s.db.Exec(
		`INSERT INTO templates (name, command, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		t.Name, t.Command, t.Description, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

// ListTemplates returns all templates sorted by name.
func (s *Store) ListTemplates() ([]Template, error) {
	rows, err := s.db.Query(
		`SELECT id, name, command, description, created_at, updated_at
		 FROM templates ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []Template
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.Name, &t.Command, &t.Description, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

// GetTemplate retrieves a template by name.
func (s *Store) GetTemplate(name string) (*Template, error) {
	var t Template
	err := s.db.QueryRow(
		`SELECT id, name, command, description, created_at, updated_at
		 FROM templates WHERE name = ?`, name,
	).Scan(&t.ID, &t.Name, &t.Command, &t.Description, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// UpdateTemplate updates an existing template.
func (s *Store) UpdateTemplate(t Template) error {
	_, err := s.db.Exec(
		`UPDATE templates SET command = ?, description = ?, updated_at = ?
		 WHERE name = ?`,
		t.Command, t.Description, t.UpdatedAt, t.Name,
	)
	return err
}

// DeleteTemplate removes a template by name.
func (s *Store) DeleteTemplate(name string) error {
	result, err := s.db.Exec(`DELETE FROM templates WHERE name = ?`, name)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// UpsertTemplate inserts or updates a template by name.
func (s *Store) UpsertTemplate(t Template) error {
	_, err := s.db.Exec(
		`INSERT INTO templates (name, command, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(name) DO UPDATE SET
		   command = excluded.command,
		   description = excluded.description,
		   updated_at = excluded.updated_at`,
		t.Name, t.Command, t.Description, t.CreatedAt, t.UpdatedAt,
	)
	return err
}
