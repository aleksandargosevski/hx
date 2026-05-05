package cmd

import (
	"fmt"
	"time"

	"hx/internal/db"
	"github.com/spf13/cobra"
)

var (
	recordCommand  string
	recordDir      string
	recordExitCode int
	recordDuration int
)

var recordCmd = &cobra.Command{
	Use:    "record",
	Short:  "Record a command to the history database",
	Hidden: true, // Called by the zsh preexec/precmd hooks
	RunE: func(cmd *cobra.Command, args []string) error {
		if recordCommand == "" {
			return nil
		}

		store, err := db.Open()
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer store.Close()

		entry := db.HistoryEntry{
			Command:   recordCommand,
			Timestamp: time.Now().Unix(),
			Duration:  recordDuration,
			Directory: recordDir,
			ExitCode:  recordExitCode,
		}

		return store.InsertHistory(entry)
	},
}

func init() {
	recordCmd.Flags().StringVar(&recordCommand, "command", "", "Command to record")
	recordCmd.Flags().StringVar(&recordDir, "dir", "", "Working directory")
	recordCmd.Flags().IntVar(&recordExitCode, "exit-code", 0, "Exit code")
	recordCmd.Flags().IntVar(&recordDuration, "duration", 0, "Duration in seconds")
}
