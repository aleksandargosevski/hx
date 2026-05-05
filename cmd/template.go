package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"hx/internal/db"
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage command templates",
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := db.Open()
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer store.Close()

		templates, err := store.ListTemplates()
		if err != nil {
			return fmt.Errorf("failed to list templates: %w", err)
		}

		if len(templates) == 0 {
			fmt.Fprintln(os.Stderr, "No templates found. Create one with: hx template add")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tCOMMAND\tDESCRIPTION")
		for _, t := range templates {
			fmt.Fprintf(w, "%s\t%s\t%s\n", t.Name, t.Command, t.Description)
		}
		return w.Flush()
	},
}

var (
	templateAddName string
	templateAddCmd  string
	templateAddDesc string
)

var templateAddCommand = &cobra.Command{
	Use:   "add",
	Short: "Add a new template",
	RunE: func(cmd *cobra.Command, args []string) error {
		if templateAddName == "" || templateAddCmd == "" {
			return fmt.Errorf("--name and --command are required")
		}

		store, err := db.Open()
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer store.Close()

		tmpl := db.Template{
			Name:        templateAddName,
			Command:     templateAddCmd,
			Description: templateAddDesc,
			CreatedAt:   time.Now().Unix(),
			UpdatedAt:   time.Now().Unix(),
		}

		if err := store.InsertTemplate(tmpl); err != nil {
			return fmt.Errorf("failed to add template: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Template '%s' added\n", templateAddName)
		return nil
	},
}

var templateRemoveName string

var templateRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a template",
	RunE: func(cmd *cobra.Command, args []string) error {
		if templateRemoveName == "" {
			return fmt.Errorf("--name is required")
		}

		store, err := db.Open()
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer store.Close()

		if err := store.DeleteTemplate(templateRemoveName); err != nil {
			return fmt.Errorf("failed to remove template: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Template '%s' removed\n", templateRemoveName)
		return nil
	},
}

func init() {
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateAddCommand)
	templateCmd.AddCommand(templateRemoveCmd)

	templateAddCommand.Flags().StringVar(&templateAddName, "name", "", "Template name")
	templateAddCommand.Flags().StringVar(&templateAddCmd, "command", "", "Template command with ${N:label} placeholders")
	templateAddCommand.Flags().StringVar(&templateAddDesc, "description", "", "Template description")

	templateRemoveCmd.Flags().StringVar(&templateRemoveName, "name", "", "Template name to remove")
}
