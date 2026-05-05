package history

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"hx/internal/db"
)

// ParseFile reads a zsh history file and returns parsed entries.
// Supports both EXTENDED_HISTORY format (: timestamp:duration;command)
// and plain format (command per line).
func ParseFile(path string) ([]db.HistoryEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []db.HistoryEntry
	scanner := bufio.NewScanner(f)

	// Increase buffer size for very long commands
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	var pendingLine string

	for scanner.Scan() {
		line := scanner.Text()

		// Handle multi-line commands (lines ending with \)
		if pendingLine != "" {
			pendingLine += "\n" + line
			if !strings.HasSuffix(line, "\\") {
				entry := parseLine(pendingLine)
				if entry != nil {
					entries = append(entries, *entry)
				}
				pendingLine = ""
			}
			continue
		}

		if strings.HasSuffix(line, "\\") {
			pendingLine = line
			continue
		}

		entry := parseLine(line)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}

	// Handle any remaining pending line
	if pendingLine != "" {
		entry := parseLine(pendingLine)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}

	return entries, scanner.Err()
}

// parseLine parses a single line from a zsh history file.
// EXTENDED_HISTORY format: ": timestamp:duration;command"
// Plain format: "command"
func parseLine(line string) *db.HistoryEntry {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	// Try EXTENDED_HISTORY format: ": 1234567890:0;command"
	if strings.HasPrefix(line, ": ") {
		return parseExtended(line)
	}

	// Plain format
	return &db.HistoryEntry{
		Command: line,
	}
}

func parseExtended(line string) *db.HistoryEntry {
	// Format: ": timestamp:duration;command"
	// Remove ": " prefix
	rest := line[2:]

	// Find the semicolon that separates metadata from command
	semiIdx := strings.Index(rest, ";")
	if semiIdx < 0 {
		return &db.HistoryEntry{Command: line}
	}

	meta := rest[:semiIdx]
	command := rest[semiIdx+1:]

	if strings.TrimSpace(command) == "" {
		return nil
	}

	// Parse "timestamp:duration"
	colonIdx := strings.Index(meta, ":")
	if colonIdx < 0 {
		return &db.HistoryEntry{Command: command}
	}

	timestamp, err := strconv.ParseInt(strings.TrimSpace(meta[:colonIdx]), 10, 64)
	if err != nil {
		return &db.HistoryEntry{Command: command}
	}

	duration, _ := strconv.Atoi(strings.TrimSpace(meta[colonIdx+1:]))

	return &db.HistoryEntry{
		Command:   command,
		Timestamp: timestamp,
		Duration:  duration,
	}
}
