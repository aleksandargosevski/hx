package history

import (
	"os"
	"testing"
)

func TestParseExtendedHistory(t *testing.T) {
	content := `: 1700000000:0;echo hello
: 1700000001:5;docker compose up -d
: 1700000002:0;git status
: 1700000003:2;ssh user@host.example.com -p 2222
`
	tmpFile, err := os.CreateTemp("", "zsh_history_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	entries, err := ParseFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	tests := []struct {
		idx       int
		command   string
		timestamp int64
		duration  int
	}{
		{0, "echo hello", 1700000000, 0},
		{1, "docker compose up -d", 1700000001, 5},
		{2, "git status", 1700000002, 0},
		{3, "ssh user@host.example.com -p 2222", 1700000003, 2},
	}

	for _, tt := range tests {
		e := entries[tt.idx]
		if e.Command != tt.command {
			t.Errorf("entry %d: expected command %q, got %q", tt.idx, tt.command, e.Command)
		}
		if e.Timestamp != tt.timestamp {
			t.Errorf("entry %d: expected timestamp %d, got %d", tt.idx, tt.timestamp, e.Timestamp)
		}
		if e.Duration != tt.duration {
			t.Errorf("entry %d: expected duration %d, got %d", tt.idx, tt.duration, e.Duration)
		}
	}
}

func TestParsePlainHistory(t *testing.T) {
	content := `echo hello
docker compose up -d
git status
`
	tmpFile, err := os.CreateTemp("", "zsh_history_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	entries, err := ParseFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	if entries[0].Command != "echo hello" {
		t.Errorf("expected %q, got %q", "echo hello", entries[0].Command)
	}
	if entries[0].Timestamp != 0 {
		t.Errorf("expected timestamp 0 for plain format, got %d", entries[0].Timestamp)
	}
}

func TestParseMultilineCommand(t *testing.T) {
	content := `: 1700000000:0;echo hello \
world
: 1700000001:0;git status
`
	tmpFile, err := os.CreateTemp("", "zsh_history_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	entries, err := ParseFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].Command != "echo hello \\\nworld" {
		t.Errorf("expected multiline command, got %q", entries[0].Command)
	}
}
