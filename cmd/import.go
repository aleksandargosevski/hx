package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"hx/internal/db"
	"hx/internal/history"
	"github.com/spf13/cobra"
)

var importFile string

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import history from zsh history file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if importFile == "" {
			home, _ := os.UserHomeDir()
			importFile = filepath.Join(home, ".zsh_history")
		}

		store, err := db.Open()
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer store.Close()

		entries, err := history.ParseFile(importFile)
		if err != nil {
			return fmt.Errorf("failed to parse history file: %w", err)
		}

		count, err := store.BulkInsertHistory(entries)
		if err != nil {
			return fmt.Errorf("failed to import history: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Imported %d entries from %s\n", count, importFile)
		return nil
	},
}

func init() {
	importCmd.Flags().StringVarP(&importFile, "file", "f", "", "Path to zsh history file (default: ~/.zsh_history)")
}
