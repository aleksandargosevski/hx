package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"hx/internal/db"
	"github.com/sahilm/fuzzy"
)

// Mode represents the current TUI mode.
type Mode int

const (
	ModeSearch Mode = iota
	ModeTemplates
	ModeEdit
	ModeTemplateCreate
)

// SearchModel is the main bubbletea model for the search TUI.
type SearchModel struct {
	// Data
	entries   []db.HistoryEntry
	templates []db.Template
	store     *db.Store

	// Search state
	query       string
	cursor      int // Index in filtered results
	matches     []fuzzy.Match
	filtered    []db.HistoryEntry
	matchIdxs   [][]int // Matched character indices per result
	successOnly bool    // Filter to exit code 0 only
	dirFilter   bool    // Filter to current directory only
	cwd         string  // Current working directory (passed from shell)
	allEntries  []db.HistoryEntry // Full entries before dir filter

	// Template search state
	templateQuery    string
	templateCursor   int
	templateMatches  []fuzzy.Match
	templateFiltered []db.Template

	// Mode
	mode Mode

	// Edit state
	editBuffer string
	editCursor int

	// Template create state
	createSource     string // Original command
	createTemplate   string // Template with placeholders
	createName       string
	createDesc       string
	createField      int // 0=template, 1=name, 2=desc
	createCursorPos  int

	// Undo state
	lastDeleted []db.HistoryEntry // Stack of deleted entries for undo

	// Terminal size
	width  int
	height int

	// Result
	selected string
}

// historySource implements fuzzy.Source for history entries.
type historySource struct {
	entries []db.HistoryEntry
}

func (s historySource) String(i int) string { return s.entries[i].Command }
func (s historySource) Len() int            { return len(s.entries) }

// templateSource implements fuzzy.Source for templates.
type templateSource struct {
	templates []db.Template
}

func (s templateSource) String(i int) string {
	return s.templates[i].Name + " " + s.templates[i].Command
}
func (s templateSource) Len() int { return len(s.templates) }

// NewSearchModel creates a new search model.
func NewSearchModel(entries []db.HistoryEntry, templates []db.Template, store *db.Store, initialQuery string, cwd string) *SearchModel {
	m := &SearchModel{
		entries:    entries,
		allEntries: entries,
		templates:  templates,
		store:      store,
		query:      initialQuery,
		cwd:        cwd,
	}
	m.filterHistory()
	m.filterTemplates()
	return m
}

// SelectedCommand returns the command selected by the user, or empty string if cancelled.
func (m *SearchModel) SelectedCommand() string {
	return m.selected
}

func (m *SearchModel) Init() tea.Cmd {
	return nil
}

func (m *SearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch m.mode {
		case ModeSearch:
			return m.updateSearch(msg)
		case ModeTemplates:
			return m.updateTemplates(msg)
		case ModeEdit:
			return m.updateEdit(msg)
		case ModeTemplateCreate:
			return m.updateTemplateCreate(msg)
		}
	}
	return m, nil
}

func (m *SearchModel) View() tea.View {
	if m.width == 0 {
		v := tea.NewView("Loading...")
		v.AltScreen = true
		return v
	}

	var s strings.Builder

	switch m.mode {
	case ModeSearch:
		m.viewSearch(&s)
	case ModeTemplates:
		m.viewTemplates(&s)
	case ModeEdit:
		m.viewEdit(&s)
	case ModeTemplateCreate:
		m.viewTemplateCreate(&s)
	}

	v := tea.NewView(s.String())
	v.AltScreen = true
	return v
}

// --- Search mode ---

func (m *SearchModel) updateSearch(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		return m, tea.Quit

	case "enter":
		if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			m.selected = m.filtered[m.cursor].Command
		}
		return m, tea.Quit

	case "up", "ctrl+p":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "ctrl+n":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}

	case "ctrl+d":
		if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			entry := m.filtered[m.cursor]
			if m.store != nil {
				_ = m.store.SoftDeleteByCommand(entry.Command)
			}
			m.lastDeleted = append(m.lastDeleted, entry)
			m.removeEntry(entry.Command)
			m.filterHistory()
			if m.cursor >= len(m.filtered) && m.cursor > 0 {
				m.cursor--
			}
		}

	case "ctrl+z":
		if len(m.lastDeleted) > 0 {
			entry := m.lastDeleted[len(m.lastDeleted)-1]
			m.lastDeleted = m.lastDeleted[:len(m.lastDeleted)-1]
			if m.store != nil {
				_ = m.store.RestoreByCommand(entry.Command)
			}
			m.entries = append([]db.HistoryEntry{entry}, m.entries...)
			if m.dirFilter {
				m.allEntries = append([]db.HistoryEntry{entry}, m.allEntries...)
			}
			m.filterHistory()
			m.cursor = 0
		}

	case "ctrl+e":
		if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			entry := m.filtered[m.cursor]
			m.editBuffer = entry.Command
			m.editCursor = len(entry.Command)
			m.mode = ModeEdit
		}

	case "ctrl+t":
		if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			entry := m.filtered[m.cursor]
			m.createSource = entry.Command
			m.createTemplate = entry.Command
			m.createName = ""
			m.createDesc = ""
			m.createField = 0
			m.createCursorPos = len(entry.Command)
			m.mode = ModeTemplateCreate
		}

	case "ctrl+f":
		m.successOnly = !m.successOnly
		m.filterHistory()
		m.cursor = 0

	case "ctrl+g":
		if m.cwd != "" {
			m.dirFilter = !m.dirFilter
			if m.dirFilter {
				var dirEntries []db.HistoryEntry
				for _, e := range m.allEntries {
					if e.Directory == m.cwd {
						dirEntries = append(dirEntries, e)
					}
				}
				m.entries = dirEntries
			} else {
				m.entries = m.allEntries
			}
			m.filterHistory()
			m.cursor = 0
		}

	case "tab":
		m.mode = ModeTemplates
		m.templateCursor = 0

	case "backspace":
		if len(m.query) > 0 {
			m.query = m.query[:len(m.query)-1]
			m.filterHistory()
			m.cursor = 0
		}

	default:
		if c := keyChar(msg); c != "" {
			m.query += c
			m.filterHistory()
			m.cursor = 0
		}
	}
	return m, nil
}

func (m *SearchModel) padLine(line string) string {
	w := lipgloss.Width(line)
	if w >= m.width {
		return truncateStr(line, m.width)
	}
	return line + strings.Repeat(" ", m.width-w)
}

func (m *SearchModel) viewSearch(s *strings.Builder) {
	// Build exactly m.height lines, each padded to m.width
	lines := make([]string, 0, m.height)

	// Input line (top)
	prompt := stylePrompt.Render("> ")
	input := styleInput.Render(m.query)
	lines = append(lines, m.padLine(prompt+input+"█"))

	// Status bar
	statusText := fmt.Sprintf("  %d/%d", len(m.filtered), len(m.entries))
	if m.successOnly {
		statusText += " (exit:0)"
	}
	if m.dirFilter {
		statusText += " (cwd)"
	}
	status := styleStatusBar.Render(statusText)
	tabs := m.renderTabs()
	statusLine := status + strings.Repeat(" ", max(0, m.width-lipgloss.Width(status)-lipgloss.Width(tabs))) + tabs
	lines = append(lines, m.padLine(statusLine))

	// Reserve: input(1) + status(1) + separator(1) + help(1) = 4 lines
	availHeight := max(0, m.height-4)

	// Compute visible results
	visible := m.filtered
	matchIdxs := m.matchIdxs
	if len(visible) > availHeight {
		visible = visible[:availHeight]
		matchIdxs = matchIdxs[:availHeight]
	}

	// Results
	for i := 0; i < len(visible); i++ {
		entry := visible[i]
		isSelected := i == m.cursor
		cmd := m.renderCommand(entry.Command, matchIdxs[i], isSelected)
		meta := m.renderMeta(entry)
		cmdWidth := lipgloss.Width(cmd)
		metaWidth := lipgloss.Width(meta)

		prefix := "  "
		if isSelected {
			prefix = styleCursor.Render("> ")
		}

		// Truncate command if it + meta won't fit
		maxCmdWidth := m.width - metaWidth - 4 // 2 prefix + 1 space + 1 margin
		if maxCmdWidth < 10 {
			maxCmdWidth = 10
		}
		if cmdWidth > maxCmdWidth {
			cmd = truncateStr(cmd, maxCmdWidth)
			cmdWidth = lipgloss.Width(cmd)
		}

		padding := m.width - cmdWidth - metaWidth - 2 // 2 for prefix
		if padding < 1 {
			padding = 1
		}
		line := prefix + cmd + strings.Repeat(" ", padding) + meta
		lines = append(lines, m.padLine(line))
	}

	// Pad with empty lines to push separator + help to the bottom
	for len(lines) < m.height-2 {
		lines = append(lines, strings.Repeat(" ", m.width))
	}

	// Separator + help (always last 2 lines)
	lines = append(lines, styleMeta.Render(strings.Repeat("─", m.width)))
	lines = append(lines, m.padLine(m.renderHelp()))

	s.WriteString(strings.Join(lines, "\n"))
}

func (m *SearchModel) filterHistory() {
	if m.query == "" {
		m.filtered = m.entries
		m.matchIdxs = make([][]int, len(m.entries))
	} else {
		source := historySource{entries: m.entries}
		results := fuzzy.FindFrom(m.query, source)

		m.filtered = make([]db.HistoryEntry, len(results))
		m.matchIdxs = make([][]int, len(results))
		m.matches = results
		for i, r := range results {
			m.filtered[i] = m.entries[r.Index]
			m.matchIdxs[i] = r.MatchedIndexes
		}
	}

	// Apply exit code filter
	if m.successOnly {
		var filteredEntries []db.HistoryEntry
		var filteredIdxs [][]int
		for i, e := range m.filtered {
			if e.ExitCode == 0 {
				filteredEntries = append(filteredEntries, e)
				if i < len(m.matchIdxs) {
					filteredIdxs = append(filteredIdxs, m.matchIdxs[i])
				}
			}
		}
		m.filtered = filteredEntries
		m.matchIdxs = filteredIdxs
	}
}

func (m *SearchModel) renderCommand(cmd string, matchIdxs []int, isSelected bool) string {
	// Replace newlines with spaces to keep each entry on a single line
	cmd = strings.ReplaceAll(cmd, "\n", " ")

	if len(matchIdxs) == 0 {
		if isSelected {
			return styleSelected.Render(cmd)
		}
		return styleNormal.Render(cmd)
	}

	// Build a set of matched positions for O(1) lookup
	matchSet := make(map[int]bool, len(matchIdxs))
	for _, idx := range matchIdxs {
		matchSet[idx] = true
	}

	var result strings.Builder
	for i, ch := range cmd {
		c := string(ch)
		if matchSet[i] {
			result.WriteString(styleMatch.Render(c))
		} else if isSelected {
			result.WriteString(styleSelected.Render(c))
		} else {
			result.WriteString(styleNormal.Render(c))
		}
	}
	return result.String()
}

func (m *SearchModel) renderMeta(entry db.HistoryEntry) string {
	parts := []string{}

	if entry.Duration > 0 {
		parts = append(parts, formatDuration(entry.Duration))
	}

	if entry.Timestamp > 0 {
		parts = append(parts, timeAgo(entry.Timestamp))
	}

	if entry.Directory != "" {
		dir := entry.Directory
		// Shorten home directory
		if home := homeDir(); home != "" && strings.HasPrefix(dir, home) {
			dir = "~" + dir[len(home):]
		}
		// Shorten long paths
		if len(dir) > 20 {
			parts2 := strings.Split(dir, "/")
			if len(parts2) > 3 {
				dir = ".../" + strings.Join(parts2[len(parts2)-2:], "/")
			}
		}
		parts = append(parts, dir)
	}

	if entry.ExitCode != 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(colorDanger).Render(fmt.Sprintf("E%d", entry.ExitCode)))
	}

	return styleMeta.Render(strings.Join(parts, "  "))
}

func (m *SearchModel) renderHelp() string {
	filterDesc := "ok"
	if m.successOnly {
		filterDesc = "all"
	}
	dirDesc := "cwd"
	if m.dirFilter {
		dirDesc = "all"
	}
	keys := []struct{ key, desc string }{
		{"^d", "del"},
		{"^z", "undo"},
		{"^e", "edit"},
		{"^f", filterDesc},
		{"^g", dirDesc},
		{"^t", "tmpl"},
		{"tab", "templates"},
		{"esc", "cancel"},
	}
	var parts []string
	for _, k := range keys {
		parts = append(parts, styleHelpKey.Render(k.key)+styleHelp.Render(":"+k.desc))
	}
	return "  " + strings.Join(parts, "  ")
}

func (m *SearchModel) renderTemplateHelp() string {
	keys := []struct{ key, desc string }{
		{"enter", "select"},
		{"tab", "history"},
		{"esc", "cancel"},
	}
	var parts []string
	for _, k := range keys {
		parts = append(parts, styleHelpKey.Render(k.key)+styleHelp.Render(":"+k.desc))
	}
	return "  " + strings.Join(parts, "  ")
}

func (m *SearchModel) renderTabs() string {
	search := styleTabActive.Render("history")
	templates := styleTabInactive.Render("templates")
	if m.mode == ModeTemplates {
		search = styleTabInactive.Render("history")
		templates = styleTabActive.Render("templates")
	}
	return search + styleMeta.Render(" | ") + templates
}

// --- Templates mode ---

func (m *SearchModel) updateTemplates(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		return m, tea.Quit

	case "tab":
		m.mode = ModeSearch
		m.cursor = 0

	case "enter":
		if len(m.templateFiltered) > 0 && m.templateCursor < len(m.templateFiltered) {
			tmpl := m.templateFiltered[m.templateCursor]
			// Output raw template with prefix — the zsh widget handles expansion
			m.selected = "__hx_template__:" + tmpl.Command
			return m, tea.Quit
		}

	case "up", "ctrl+p":
		if m.templateCursor > 0 {
			m.templateCursor--
		}

	case "down", "ctrl+n":
		if m.templateCursor < len(m.templateFiltered)-1 {
			m.templateCursor++
		}

	case "backspace":
		if len(m.templateQuery) > 0 {
			m.templateQuery = m.templateQuery[:len(m.templateQuery)-1]
			m.filterTemplates()
			m.templateCursor = 0
		}

	default:
		if c := keyChar(msg); c != "" {
			m.templateQuery += c
			m.filterTemplates()
			m.templateCursor = 0
		}
	}
	return m, nil
}

func (m *SearchModel) viewTemplates(s *strings.Builder) {
	// Build exactly m.height lines, each padded to m.width
	lines := make([]string, 0, m.height)

	// Input line (top)
	prompt := stylePrompt.Render("tmpl> ")
	input := styleInput.Render(m.templateQuery)
	lines = append(lines, m.padLine(prompt+input+"█"))

	// Status bar
	status := styleStatusBar.Render(fmt.Sprintf("  %d/%d templates", len(m.templateFiltered), len(m.templates)))
	tabs := m.renderTabs()
	statusLine := status + strings.Repeat(" ", max(0, m.width-lipgloss.Width(status)-lipgloss.Width(tabs))) + tabs
	lines = append(lines, m.padLine(statusLine))

	// Reserve: input(1) + status(1) + separator(1) + help(1) = 4 lines
	availHeight := max(0, m.height-4)

	visible := m.templateFiltered
	if len(visible) > availHeight {
		visible = visible[:availHeight]
	}

	// Results
	for i := 0; i < len(visible); i++ {
		tmpl := visible[i]
		isSelected := i == m.templateCursor

		prefix := "  "
		if isSelected {
			prefix = styleCursor.Render("> ")
		}

		name := styleHelpKey.Render(tmpl.Name)
		cmd := styleMeta.Render("  " + tmpl.Command)
		desc := ""
		if tmpl.Description != "" {
			desc = styleMeta.Render("  — " + tmpl.Description)
		}

		line := prefix + name + cmd + desc
		lines = append(lines, m.padLine(line))
	}

	// Pad with empty lines to push separator + help to the bottom
	for len(lines) < m.height-2 {
		lines = append(lines, strings.Repeat(" ", m.width))
	}

	// Separator + help (always last 2 lines)
	lines = append(lines, styleMeta.Render(strings.Repeat("─", m.width)))
	lines = append(lines, m.padLine(m.renderTemplateHelp()))

	s.WriteString(strings.Join(lines, "\n"))
}

func (m *SearchModel) filterTemplates() {
	if m.templateQuery == "" {
		m.templateFiltered = m.templates
		return
	}

	source := templateSource{templates: m.templates}
	results := fuzzy.FindFrom(m.templateQuery, source)

	m.templateFiltered = make([]db.Template, len(results))
	m.templateMatches = results
	for i, r := range results {
		m.templateFiltered[i] = m.templates[r.Index]
	}
}

func (m *SearchModel) removeEntry(command string) {
	var newEntries []db.HistoryEntry
	for _, e := range m.entries {
		if e.Command != command {
			newEntries = append(newEntries, e)
		}
	}
	m.entries = newEntries
}

// --- Edit mode ---

func (m *SearchModel) updateEdit(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.cursor < len(m.filtered) {
			entry := m.filtered[m.cursor]
			if m.store != nil && m.editBuffer != entry.Command {
				_ = m.store.UpdateHistoryCommand(entry.ID, m.editBuffer)
				// Update in-memory
				for i := range m.entries {
					if m.entries[i].ID == entry.ID {
						m.entries[i].Command = m.editBuffer
						break
					}
				}
				m.filterHistory()
			}
		}
		m.mode = ModeSearch

	case "esc", "ctrl+c":
		m.mode = ModeSearch

	case "backspace":
		if m.editCursor > 0 {
			m.editBuffer = m.editBuffer[:m.editCursor-1] + m.editBuffer[m.editCursor:]
			m.editCursor--
		}

	case "left":
		if m.editCursor > 0 {
			m.editCursor--
		}

	case "right":
		if m.editCursor < len(m.editBuffer) {
			m.editCursor++
		}

	case "ctrl+a":
		m.editCursor = 0

	case "ctrl+e":
		m.editCursor = len(m.editBuffer)

	case "ctrl+k":
		m.editBuffer = m.editBuffer[:m.editCursor]

	case "ctrl+u":
		m.editBuffer = m.editBuffer[m.editCursor:]
		m.editCursor = 0

	default:
		if c := keyChar(msg); c != "" {
			m.editBuffer = m.editBuffer[:m.editCursor] + c + m.editBuffer[m.editCursor:]
			m.editCursor++
		}
	}
	return m, nil
}

func (m *SearchModel) viewEdit(s *strings.Builder) {
	availHeight := m.height - 4

	for i := 0; i < availHeight; i++ {
		s.WriteString("\n")
	}

	title := stylePrompt.Render("Edit command")
	s.WriteString("  " + title + "\n")

	// Render edit buffer with cursor
	before := m.editBuffer[:m.editCursor]
	after := m.editBuffer[m.editCursor:]
	cursorChar := "█"
	if len(after) > 0 {
		cursorChar = string(after[0])
		after = after[1:]
	}

	line := "  " + styleNormal.Render(before) + lipgloss.NewStyle().Reverse(true).Render(cursorChar) + styleNormal.Render(after)
	s.WriteString(line + "\n")

	help := styleHelp.Render("  enter:save  esc:cancel  ctrl+a:home  ctrl+e:end  ctrl+k:kill-right  ctrl+u:kill-left")
	s.WriteString(help)
}

// --- Template create mode ---

func (m *SearchModel) updateTemplateCreate(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = ModeSearch

	case "ctrl+s":
		// Save the template
		if m.createName != "" && m.createTemplate != "" && m.store != nil {
			now := time.Now().Unix()
			_ = m.store.InsertTemplate(db.Template{
				Name:        m.createName,
				Command:     m.createTemplate,
				Description: m.createDesc,
				CreatedAt:   now,
				UpdatedAt:   now,
			})
			// Reload templates
			if templates, err := m.store.ListTemplates(); err == nil {
				m.templates = templates
				m.filterTemplates()
			}
		}
		m.mode = ModeSearch

	case "tab":
		m.createField = (m.createField + 1) % 3

	case "shift+tab":
		m.createField = (m.createField + 2) % 3

	case "backspace":
		switch m.createField {
		case 0:
			if m.createCursorPos > 0 {
				m.createTemplate = m.createTemplate[:m.createCursorPos-1] + m.createTemplate[m.createCursorPos:]
				m.createCursorPos--
			}
		case 1:
			if len(m.createName) > 0 {
				m.createName = m.createName[:len(m.createName)-1]
			}
		case 2:
			if len(m.createDesc) > 0 {
				m.createDesc = m.createDesc[:len(m.createDesc)-1]
			}
		}

	case "left":
		if m.createField == 0 && m.createCursorPos > 0 {
			m.createCursorPos--
		}

	case "right":
		if m.createField == 0 && m.createCursorPos < len(m.createTemplate) {
			m.createCursorPos++
		}

	default:
		if c := keyChar(msg); c != "" {
			switch m.createField {
			case 0:
				m.createTemplate = m.createTemplate[:m.createCursorPos] + c + m.createTemplate[m.createCursorPos:]
				m.createCursorPos++
			case 1:
				m.createName += c
			case 2:
				m.createDesc += c
			}
		}
	}
	return m, nil
}

func (m *SearchModel) viewTemplateCreate(s *strings.Builder) {
	availHeight := m.height - 10

	for i := 0; i < availHeight; i++ {
		s.WriteString("\n")
	}

	title := stylePrompt.Render("Create template from command")
	s.WriteString("  " + title + "\n\n")

	// Original command
	s.WriteString("  " + styleMeta.Render("Original: ") + styleNormal.Render(m.createSource) + "\n\n")

	// Template field
	fieldLabel := "  Template: "
	if m.createField == 0 {
		fieldLabel = styleCursor.Render("> ") + stylePrompt.Render("Template: ")
	} else {
		fieldLabel = "  " + styleMeta.Render("Template: ")
	}
	if m.createField == 0 {
		before := m.createTemplate[:m.createCursorPos]
		after := m.createTemplate[m.createCursorPos:]
		s.WriteString(fieldLabel + styleNormal.Render(before) + "█" + styleNormal.Render(after) + "\n")
	} else {
		s.WriteString(fieldLabel + styleNormal.Render(m.createTemplate) + "\n")
	}

	// Name field
	nameLabel := "  "
	if m.createField == 1 {
		nameLabel = styleCursor.Render("> ") + stylePrompt.Render("Name:     ")
	} else {
		nameLabel = "  " + styleMeta.Render("Name:     ")
	}
	nameVal := m.createName
	if m.createField == 1 {
		nameVal += "█"
	}
	s.WriteString(nameLabel + styleInput.Render(nameVal) + "\n")

	// Desc field
	descLabel := "  "
	if m.createField == 2 {
		descLabel = styleCursor.Render("> ") + stylePrompt.Render("Desc:     ")
	} else {
		descLabel = "  " + styleMeta.Render("Desc:     ")
	}
	descVal := m.createDesc
	if m.createField == 2 {
		descVal += "█"
	}
	s.WriteString(descLabel + styleInput.Render(descVal) + "\n\n")

	hint := styleMeta.Render("  Tip: use ${1:label} syntax for placeholders, e.g. docker exec -it ${1:container} ${2:cmd}")
	s.WriteString(hint + "\n")

	help := styleHelp.Render("  ctrl+s:save  tab:next field  esc:cancel")
	s.WriteString(help)
}

// --- Helpers ---

func formatDuration(seconds int) string {
	switch {
	case seconds < 60:
		return fmt.Sprintf("%ds", seconds)
	case seconds < 3600:
		m := seconds / 60
		s := seconds % 60
		if s == 0 {
			return fmt.Sprintf("%dm", m)
		}
		return fmt.Sprintf("%dm%ds", m, s)
	default:
		h := seconds / 3600
		m := (seconds % 3600) / 60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh%dm", h, m)
	}
}

func timeAgo(timestamp int64) string {
	now := time.Now().Unix()
	diff := now - timestamp

	switch {
	case diff < 60:
		return "just now"
	case diff < 3600:
		m := diff / 60
		return fmt.Sprintf("%dm ago", m)
	case diff < 86400:
		h := diff / 3600
		return fmt.Sprintf("%dh ago", h)
	case diff < 604800:
		d := diff / 86400
		return fmt.Sprintf("%dd ago", d)
	case diff < 2592000:
		w := diff / 604800
		return fmt.Sprintf("%dw ago", w)
	case diff < 31536000:
		mo := diff / 2592000
		return fmt.Sprintf("%dmo ago", mo)
	default:
		y := diff / 31536000
		return fmt.Sprintf("%dy ago", y)
	}
}

func homeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

func truncateStr(s string, maxWidth int) string {
	return ansi.Truncate(s, maxWidth, "…")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// keyChar returns the printable character for a key press, or "" if not printable.
// Handles bubbletea v2's named keys like "space" -> " ".
func keyChar(msg tea.KeyPressMsg) string {
	s := msg.String()
	if s == "space" {
		return " "
	}
	if len(s) == 1 {
		return s
	}
	return ""
}
