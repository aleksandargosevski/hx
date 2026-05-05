package template

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Placeholder represents a template parameter like ${1:container}.
type Placeholder struct {
	Index    int    // Tab-stop order (1-based)
	Label    string // Descriptive name
	Start    int    // Start position in the template string
	End      int    // End position in the template string
	Original string // The full placeholder text e.g. "${1:container}"
}

var placeholderRe = regexp.MustCompile(`\$\{(\d+):([^}]+)\}`)

// Parse extracts all placeholders from a template string.
// Returns them sorted by index.
func Parse(tmpl string) []Placeholder {
	matches := placeholderRe.FindAllStringSubmatchIndex(tmpl, -1)
	var placeholders []Placeholder

	for _, match := range matches {
		// match[0]:match[1] = full match
		// match[2]:match[3] = index group
		// match[4]:match[5] = label group
		idx, _ := strconv.Atoi(tmpl[match[2]:match[3]])
		label := tmpl[match[4]:match[5]]

		placeholders = append(placeholders, Placeholder{
			Index:    idx,
			Label:    label,
			Start:    match[0],
			End:      match[1],
			Original: tmpl[match[0]:match[1]],
		})
	}

	sort.Slice(placeholders, func(i, j int) bool {
		return placeholders[i].Index < placeholders[j].Index
	})

	return placeholders
}

// Fill replaces all placeholders in a template with the provided values.
// values is a map of placeholder index to replacement value.
func Fill(tmpl string, values map[int]string) string {
	result := tmpl
	// Replace in reverse order to preserve positions
	placeholders := Parse(tmpl)
	for i := len(placeholders) - 1; i >= 0; i-- {
		p := placeholders[i]
		if val, ok := values[p.Index]; ok {
			result = result[:p.Start] + val + result[p.End:]
		}
	}
	return result
}

// Preview returns the template with placeholders shown as their labels.
// e.g. "docker exec -it ${1:container} ${2:cmd}" → "docker exec -it <container> <cmd>"
func Preview(tmpl string) string {
	return placeholderRe.ReplaceAllStringFunc(tmpl, func(match string) string {
		sub := placeholderRe.FindStringSubmatch(match)
		if len(sub) >= 3 {
			return "<" + sub[2] + ">"
		}
		return match
	})
}

// ToTemplate converts a command into a template by replacing specified substrings
// with numbered placeholders.
// replacements is a list of (substring, label) pairs.
func ToTemplate(command string, replacements []struct{ Substr, Label string }) string {
	result := command
	for i, r := range replacements {
		placeholder := fmt.Sprintf("${%d:%s}", i+1, r.Label)
		result = strings.Replace(result, r.Substr, placeholder, 1)
	}
	return result
}

// ExpandedPlaceholder represents a placeholder's position in the expanded string.
type ExpandedPlaceholder struct {
	Index int    // Tab-stop order (1-based)
	Label string // Descriptive name (also the default text in the expanded string)
	Start int    // Start position in the expanded string (0-based)
	End   int    // End position in the expanded string (exclusive)
}

// Expand replaces all ${N:label} placeholders with just their label text
// and returns the expanded string along with the positions of each placeholder
// in the expanded string (sorted by Index for tab-stop order).
//
// Example:
//
//	Expand("docker exec -it ${1:container} ${2:cmd}")
//	=> "docker exec -it container cmd", [{1,"container",16,25}, {2,"cmd",26,29}]
func Expand(tmpl string) (string, []ExpandedPlaceholder) {
	// First, find all placeholders in the original string (sorted by position)
	matches := placeholderRe.FindAllStringSubmatchIndex(tmpl, -1)
	if len(matches) == 0 {
		return tmpl, nil
	}

	// Build a list of replacements with their original positions
	type replacement struct {
		origStart int
		origEnd   int
		index     int
		label     string
	}
	var replacements []replacement
	for _, match := range matches {
		idx, _ := strconv.Atoi(tmpl[match[2]:match[3]])
		label := tmpl[match[4]:match[5]]
		replacements = append(replacements, replacement{
			origStart: match[0],
			origEnd:   match[1],
			index:     idx,
			label:     label,
		})
	}

	// Build the expanded string and track new positions
	var expanded strings.Builder
	var placeholders []ExpandedPlaceholder
	prev := 0

	for _, r := range replacements {
		// Copy text before this placeholder
		expanded.WriteString(tmpl[prev:r.origStart])
		start := expanded.Len()
		expanded.WriteString(r.label)
		end := expanded.Len()

		placeholders = append(placeholders, ExpandedPlaceholder{
			Index: r.index,
			Label: r.label,
			Start: start,
			End:   end,
		})

		prev = r.origEnd
	}
	// Copy remaining text after last placeholder
	expanded.WriteString(tmpl[prev:])

	// Sort by Index for tab-stop order
	sort.Slice(placeholders, func(i, j int) bool {
		return placeholders[i].Index < placeholders[j].Index
	})

	return expanded.String(), placeholders
}
