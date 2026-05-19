package cmd

import (
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"hx/internal/config"
	"hx/internal/db"
	"hx/internal/tui"
	"github.com/spf13/cobra"
)

var searchQuery string
var searchCwd string

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Launch the fuzzy history search TUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := db.Open()
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer store.Close()

		entries, err := store.ListHistory(10000)
		if err != nil {
			return fmt.Errorf("failed to load history: %w", err)
		}

		frecencyEntries, err := store.ListHistoryFrecency(time.Now().Unix(), 10000)
		if err != nil {
			return fmt.Errorf("failed to load frecency history: %w", err)
		}

		// Load templates from DB
		templates, err := store.ListTemplates()
		if err != nil {
			return fmt.Errorf("failed to load templates: %w", err)
		}

		// Merge in templates from YAML config file
		configTemplates, _ := config.LoadTemplates(config.TemplatesPath())
		for _, ct := range configTemplates {
			now := time.Now().Unix()
			_ = store.UpsertTemplate(db.Template{
				Name:        ct.Name,
				Command:     ct.Command,
				Description: ct.Description,
				CreatedAt:   now,
				UpdatedAt:   now,
			})
		}
		if len(configTemplates) > 0 {
			// Reload after merge
			templates, _ = store.ListTemplates()
		}

		model := tui.NewSearchModel(entries, frecencyEntries, templates, store, searchQuery, searchCwd)

		// Run TUI on /dev/tty so stdout can be captured by the zsh widget
		ttyFile, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			return fmt.Errorf("failed to open /dev/tty: %w", err)
		}
		defer ttyFile.Close()

		p := tea.NewProgram(model,
			tea.WithInput(ttyFile),
			tea.WithOutput(ttyFile),
		)

		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		// Print selected command to stdout (captured by zsh widget)
		if m, ok := finalModel.(*tui.SearchModel); ok {
			if selected := m.SelectedCommand(); selected != "" {
				fmt.Print(selected)
			}
		}

		return nil
	},
}

func init() {
	searchCmd.Flags().StringVar(&searchQuery, "query", "", "Initial search query")
	searchCmd.Flags().StringVar(&searchCwd, "cwd", "", "Current working directory for directory filter")
}
