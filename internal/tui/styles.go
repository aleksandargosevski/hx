package tui

import (
	"charm.land/lipgloss/v2"
)

var (
	// Colors
	colorPrimary   = lipgloss.Color("#7C3AED") // Violet
	colorSecondary = lipgloss.Color("#6B7280") // Gray
	colorDanger    = lipgloss.Color("#EF4444") // Red
	colorMuted     = lipgloss.Color("#4B5563") // Dark gray
	colorHighlight = lipgloss.Color("#A78BFA") // Light violet
	colorText      = lipgloss.Color("#E5E7EB") // Light gray

	// Styles
	stylePrompt = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	styleInput = lipgloss.NewStyle().
			Foreground(colorText)

	styleSelected = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	styleNormal = lipgloss.NewStyle().
			Foreground(colorText)

	styleMatch = lipgloss.NewStyle().
			Foreground(colorHighlight).
			Bold(true)

	styleMeta = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleStatusBar = lipgloss.NewStyle().
			Foreground(colorSecondary)

	styleHelp = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleHelpKey = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	styleCursor = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	styleTabActive = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Underline(true)

	styleTabInactive = lipgloss.NewStyle().
				Foreground(colorMuted)
)
