package template

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		labels   []string
	}{
		{
			name:     "single placeholder",
			input:    "git rebase -i HEAD~${1:count}",
			expected: 1,
			labels:   []string{"count"},
		},
		{
			name:     "two placeholders",
			input:    "docker exec -it ${1:container} ${2:command}",
			expected: 2,
			labels:   []string{"container", "command"},
		},
		{
			name:     "no placeholders",
			input:    "git status",
			expected: 0,
			labels:   nil,
		},
		{
			name:     "three placeholders",
			input:    "ssh ${1:user}@${2:host} -p ${3:port}",
			expected: 3,
			labels:   []string{"user", "host", "port"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.input)
			if len(result) != tt.expected {
				t.Errorf("expected %d placeholders, got %d", tt.expected, len(result))
			}
			for i, p := range result {
				if i < len(tt.labels) && p.Label != tt.labels[i] {
					t.Errorf("placeholder %d: expected label %q, got %q", i, tt.labels[i], p.Label)
				}
				if p.Index != i+1 {
					t.Errorf("placeholder %d: expected index %d, got %d", i, i+1, p.Index)
				}
			}
		})
	}
}

func TestFill(t *testing.T) {
	tests := []struct {
		name     string
		tmpl     string
		values   map[int]string
		expected string
	}{
		{
			name:     "single replacement",
			tmpl:     "git rebase -i HEAD~${1:count}",
			values:   map[int]string{1: "5"},
			expected: "git rebase -i HEAD~5",
		},
		{
			name:     "multiple replacements",
			tmpl:     "docker exec -it ${1:container} ${2:command}",
			values:   map[int]string{1: "api-prod", 2: "bash"},
			expected: "docker exec -it api-prod bash",
		},
		{
			name:     "partial replacement",
			tmpl:     "ssh ${1:user}@${2:host} -p ${3:port}",
			values:   map[int]string{1: "root", 2: "example.com"},
			expected: "ssh root@example.com -p ${3:port}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Fill(tt.tmpl, tt.values)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestPreview(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "docker exec -it ${1:container} ${2:cmd}",
			expected: "docker exec -it <container> <cmd>",
		},
		{
			input:    "git status",
			expected: "git status",
		},

	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Preview(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestToTemplate(t *testing.T) {
	replacements := []struct{ Substr, Label string }{
		{"api-prod", "container"},
		{"bash", "command"},
	}
	result := ToTemplate("docker exec -it api-prod bash", replacements)
	expected := "docker exec -it ${1:container} ${2:command}"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExpand(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedStr    string
		expectedCount  int
		expectedLabels []string
		expectedStarts []int
		expectedEnds   []int
	}{
		{
			name:           "two placeholders",
			input:          "docker exec -it ${1:container} ${2:cmd}",
			expectedStr:    "docker exec -it container cmd",
			expectedCount:  2,
			expectedLabels: []string{"container", "cmd"},
			expectedStarts: []int{16, 26},
			expectedEnds:   []int{25, 29},
		},
		{
			name:          "no placeholders",
			input:         "git status",
			expectedStr:   "git status",
			expectedCount: 0,
		},
		{
			name:           "three placeholders",
			input:          "ssh ${1:user}@${2:host} -p ${3:port}",
			expectedStr:    "ssh user@host -p port",
			expectedCount:  3,
			expectedLabels: []string{"user", "host", "port"},
			expectedStarts: []int{4, 9, 17},
			expectedEnds:   []int{8, 13, 21},
		},
		{
			name:           "placeholders out of order in string",
			input:          "${2:second} then ${1:first}",
			expectedStr:    "second then first",
			expectedCount:  2,
			expectedLabels: []string{"first", "second"}, // sorted by Index
			expectedStarts: []int{12, 0},                // first is at 12, second is at 0
			expectedEnds:   []int{17, 6},
		},

	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expanded, placeholders := Expand(tt.input)
			if expanded != tt.expectedStr {
				t.Errorf("expanded string: expected %q, got %q", tt.expectedStr, expanded)
			}
			if len(placeholders) != tt.expectedCount {
				t.Fatalf("placeholder count: expected %d, got %d", tt.expectedCount, len(placeholders))
			}
			for i, p := range placeholders {
				if i < len(tt.expectedLabels) && p.Label != tt.expectedLabels[i] {
					t.Errorf("placeholder %d label: expected %q, got %q", i, tt.expectedLabels[i], p.Label)
				}
				if i < len(tt.expectedStarts) && p.Start != tt.expectedStarts[i] {
					t.Errorf("placeholder %d start: expected %d, got %d", i, tt.expectedStarts[i], p.Start)
				}
				if i < len(tt.expectedEnds) && p.End != tt.expectedEnds[i] {
					t.Errorf("placeholder %d end: expected %d, got %d", i, tt.expectedEnds[i], p.End)
				}
			}
		})
	}
}
