package cmd

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"hx/internal/db"
	"github.com/spf13/cobra"
)

var statsLimit int

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show history analytics and usage patterns",
	RunE:  runStats,
}

func init() {
	statsCmd.Flags().IntVarP(&statsLimit, "limit", "n", 10, "Number of entries per section")
}

func runStats(cmd *cobra.Command, args []string) error {
	store, err := db.Open()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	// Colors
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true)
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	value := lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Bold(true)
	bar := lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5563"))
	danger := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))

	// Overview
	overview, err := store.StatsOverview()
	if err != nil {
		return fmt.Errorf("failed to get overview: %w", err)
	}

	fmt.Println()
	fmt.Println(title.Render("  ⚡ History Overview"))
	fmt.Println()
	fmt.Printf("  %s %s\n", label.Render("Total commands:"), value.Render(fmt.Sprintf("%d", overview.TotalCommands)))
	fmt.Printf("  %s %s\n", label.Render("Unique commands:"), value.Render(fmt.Sprintf("%d", overview.UniqueCommands)))
	fmt.Printf("  %s %s\n", label.Render("Directories:"), value.Render(fmt.Sprintf("%d", overview.TotalDirs)))
	fmt.Printf("  %s %s\n", label.Render("Avg per day:"), value.Render(fmt.Sprintf("%.1f", overview.AvgPerDay)))

	// Top commands
	topCmds, err := store.TopCommands(statsLimit)
	if err != nil {
		return fmt.Errorf("failed to get top commands: %w", err)
	}

	if len(topCmds) > 0 {
		fmt.Println()
		fmt.Println(title.Render("  🏆 Most Used Commands"))
		fmt.Println()
		maxCount := topCmds[0].Count
		for i, c := range topCmds {
			barWidth := scaleBar(c.Count, maxCount, 20)
			cmdStr := truncate(c.Command, 50)
			fmt.Printf("  %s %s %s %s\n",
				dim.Render(fmt.Sprintf("%2d.", i+1)),
				bar.Render(strings.Repeat("█", barWidth)+strings.Repeat("░", 20-barWidth)),
				value.Render(fmt.Sprintf("%4d", c.Count)),
				label.Render(cmdStr),
			)
		}
	}

	// Top directories
	topDirs, err := store.TopDirectories(statsLimit)
	if err != nil {
		return fmt.Errorf("failed to get top directories: %w", err)
	}

	if len(topDirs) > 0 {
		fmt.Println()
		fmt.Println(title.Render("  📁 Most Active Directories"))
		fmt.Println()
		maxCount := topDirs[0].Count
		for i, d := range topDirs {
			barWidth := scaleBar(d.Count, maxCount, 20)
			dirStr := shortenDir(d.Directory, 50)
			fmt.Printf("  %s %s %s %s\n",
				dim.Render(fmt.Sprintf("%2d.", i+1)),
				bar.Render(strings.Repeat("█", barWidth)+strings.Repeat("░", 20-barWidth)),
				value.Render(fmt.Sprintf("%4d", d.Count)),
				label.Render(dirStr),
			)
		}
	}

	// Activity by hour
	hourStats, err := store.CommandsByHour()
	if err != nil {
		return fmt.Errorf("failed to get hourly stats: %w", err)
	}

	if len(hourStats) > 0 {
		fmt.Println()
		fmt.Println(title.Render("  🕐 Activity by Hour"))
		fmt.Println()

		// Fill in all 24 hours
		hourMap := make(map[int]int)
		maxHour := 0
		for _, h := range hourStats {
			hourMap[h.Hour] = h.Count
			if h.Count > maxHour {
				maxHour = h.Count
			}
		}

		// Render as a horizontal heatmap — 2 rows of 12 hours
		blocks := []string{"░", "▒", "▓", "█"}
		for _, startHour := range []int{0, 12} {
			hourLine := "  "
			countLine := "  "
			for h := startHour; h < startHour+12; h++ {
				count := hourMap[h]
				level := 0
				if maxHour > 0 {
					level = count * 3 / maxHour
				}
				hourLine += dim.Render(fmt.Sprintf("%02d ", h))
				countLine += bar.Render(fmt.Sprintf(" %s ", blocks[level]))
			}
			fmt.Println(hourLine)
			fmt.Println(countLine)
		}
	}

	// Most failing commands
	failing, err := store.MostFailing(statsLimit)
	if err != nil {
		return fmt.Errorf("failed to get failing commands: %w", err)
	}

	if len(failing) > 0 {
		fmt.Println()
		fmt.Println(title.Render("  💥 Most Failing Commands"))
		fmt.Println()
		for i, f := range failing {
			cmdStr := truncate(f.Command, 45)
			fmt.Printf("  %s %s %s %s\n",
				dim.Render(fmt.Sprintf("%2d.", i+1)),
				danger.Render(fmt.Sprintf("%5.1f%%", f.FailRate)),
				dim.Render(fmt.Sprintf("(%d/%d)", f.Failures, f.Total)),
				label.Render(cmdStr),
			)
		}
	}

	// Slowest commands
	slowest, err := store.SlowestCommands(statsLimit)
	if err != nil {
		return fmt.Errorf("failed to get slowest commands: %w", err)
	}

	if len(slowest) > 0 {
		fmt.Println()
		fmt.Println(title.Render("  🐌 Slowest Commands"))
		fmt.Println()
		for i, d := range slowest {
			cmdStr := truncate(d.Command, 45)
			fmt.Printf("  %s %s %s %s\n",
				dim.Render(fmt.Sprintf("%2d.", i+1)),
				value.Render(fmt.Sprintf("%6.1fs avg", d.AvgSeconds)),
				dim.Render(fmt.Sprintf("(max %ds, %dx)", d.MaxSeconds, d.Count)),
				label.Render(cmdStr),
			)
		}
	}

	fmt.Println()
	return nil
}

func scaleBar(count, maxCount, maxWidth int) int {
	if maxCount == 0 {
		return 0
	}
	w := count * maxWidth / maxCount
	if w == 0 && count > 0 {
		w = 1
	}
	return w
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

func shortenDir(dir string, maxLen int) string {
	home, _ := os.UserHomeDir()
	if home != "" && strings.HasPrefix(dir, home) {
		dir = "~" + dir[len(home):]
	}
	if len(dir) <= maxLen {
		return dir
	}
	parts := strings.Split(dir, "/")
	if len(parts) > 3 {
		dir = "…/" + strings.Join(parts[len(parts)-2:], "/")
	}
	return dir
}
