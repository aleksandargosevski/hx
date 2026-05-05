package cmd

import (
	"encoding/json"
	"fmt"

	tmplpkg "hx/internal/template"
	"github.com/spf13/cobra"
)

// expandResult is the JSON output of `hx expand`.
type expandResult struct {
	Expanded     string             `json:"expanded"`
	Placeholders []expandPlaceholder `json:"placeholders"`
}

type expandPlaceholder struct {
	Index int    `json:"index"`
	Label string `json:"label"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

var expandCmd = &cobra.Command{
	Use:    "expand",
	Short:  "Expand a template string and output placeholder positions as JSON",
	Hidden: true, // Called by the zsh widget
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tmpl := args[0]
		expanded, placeholders := tmplpkg.Expand(tmpl)

		result := expandResult{Expanded: expanded}
		for _, p := range placeholders {
			result.Placeholders = append(result.Placeholders, expandPlaceholder{
				Index: p.Index,
				Label: p.Label,
				Start: p.Start,
				End:   p.End,
			})
		}

		data, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("failed to marshal: %w", err)
		}
		fmt.Print(string(data))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(expandCmd)
}
