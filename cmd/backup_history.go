package cmd

import (
	"fmt"
	"strings"

	"github.com/ArdaGnsrn/opsvault/internal/config"
	"github.com/ArdaGnsrn/opsvault/internal/history"
	"github.com/ArdaGnsrn/opsvault/internal/ui"
	"github.com/spf13/cobra"
)

var (
	historyLimit int
	historyDB    string
)

var backupHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show backup history",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadFile(cfgFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		entries, err := history.Load(cfg.BackupDir)
		if err != nil {
			return fmt.Errorf("reading history: %w", err)
		}

		if len(entries) == 0 {
			fmt.Println(ui.Warn("No backup history found."))
			return nil
		}

		// filter by db name if requested
		if historyDB != "" {
			var filtered []history.Entry
			for _, e := range entries {
				if e.Database == historyDB {
					filtered = append(filtered, e)
				}
			}
			entries = filtered
		}

		// apply limit
		if historyLimit > 0 && len(entries) > historyLimit {
			entries = entries[:historyLimit]
		}

		fmt.Println()
		fmt.Printf("  %s\n\n", ui.Bold.Sprintf("%-20s  %-10s  %-8s  %-8s  %s",
			"DATABASE", "STATUS", "SIZE", "DURATION", "STARTED AT"))
		fmt.Println("  " + strings.Repeat("─", 72))

		for _, e := range entries {
			var statusStr string
			if e.Status == history.StatusSuccess {
				statusStr = ui.Green.Sprint("success")
			} else {
				statusStr = ui.Red.Sprint("failed ")
			}

			dur := fmt.Sprintf("%.1fs", e.Duration)
			size := "-"
			if e.FileSize > 0 {
				size = humanSize(e.FileSize)
			}
			date := ui.Dim.Sprint(e.StartedAt.Local().Format("2006-01-02 15:04:05"))

			fmt.Printf("  %-20s  %s  %8s  %8s  %s\n",
				ui.Bold.Sprint(e.Database), statusStr, size, dur, date)

			if e.Status == history.StatusFailed && e.Error != "" {
				fmt.Printf("  %s\n", ui.Red.Sprintf("  └─ %s", e.Error))
			}
		}
		fmt.Println()
		return nil
	},
}

func init() {
	backupHistoryCmd.Flags().IntVarP(&historyLimit, "limit", "n", 50, "max number of entries to show")
	backupHistoryCmd.Flags().StringVar(&historyDB, "db", "", "filter by database name")
	backupCmd.AddCommand(backupHistoryCmd)
}
