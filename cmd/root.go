package cmd

import (
	"github.com/spf13/cobra"
)

var appVersion = "dev"

func SetVersion(v string) {
	appVersion = v
}

var rootCmd = &cobra.Command{
	Use:   "hx",
	Short: "History Extended — a smarter shell history manager",
	Long: `hx is a fast, fuzzy-searchable shell history manager with template support.

It replaces Ctrl+R with a powerful TUI that lets you search, edit, delete,
and templatize your command history.`,
	// Default command is search
	RunE: func(cmd *cobra.Command, args []string) error {
		return searchCmd.RunE(cmd, args)
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(recordCmd)
	rootCmd.AddCommand(templateCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of hx",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("hx version", appVersion)
	},
}
