package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTemplates(t *testing.T) {
	content := `templates:
  - name: docker-exec
    command: "docker exec -it ${1:container} ${2:command}"
    description: "Execute command in running container"
  - name: git-rebase
    command: "git rebase -i HEAD~${1:count}"
    description: "Interactive rebase last N commits"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "templates.yaml")

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	templates, err := LoadTemplates(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}

	if templates[0].Name != "docker-exec" {
		t.Errorf("expected name %q, got %q", "docker-exec", templates[0].Name)
	}
	if templates[0].Command != "docker exec -it ${1:container} ${2:command}" {
		t.Errorf("expected command %q, got %q", "docker exec -it ${1:container} ${2:command}", templates[0].Command)
	}
	if templates[1].Name != "git-rebase" {
		t.Errorf("expected name %q, got %q", "git-rebase", templates[1].Name)
	}
}

func TestLoadTemplatesFileNotFound(t *testing.T) {
	templates, err := LoadTemplates("/nonexistent/path/templates.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if templates != nil {
		t.Errorf("expected nil for nonexistent file, got %v", templates)
	}
}

func TestSaveAndLoadTemplates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "templates.yaml")

	templates := []TemplateConfig{
		{
			Name:        "test-tmpl",
			Command:     "echo ${1:message}",
			Description: "Test template",
		},
	}

	if err := SaveTemplates(path, templates); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadTemplates(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 template, got %d", len(loaded))
	}
	if loaded[0].Name != "test-tmpl" {
		t.Errorf("expected name %q, got %q", "test-tmpl", loaded[0].Name)
	}
	if loaded[0].Command != "echo ${1:message}" {
		t.Errorf("expected command %q, got %q", "echo ${1:message}", loaded[0].Command)
	}
}
