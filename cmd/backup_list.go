package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ArdaGnsrn/opsvault/internal/ui"
	"github.com/spf13/cobra"
)

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent local backup files",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		var matches []string
		for _, pat := range []string{"*.sql.gz", "*.tar.gz"} {
			m, err := filepath.Glob(filepath.Join(cfg.BackupDir, pat))
			if err != nil {
				return err
			}
			matches = append(matches, m...)
		}

		if len(matches) == 0 {
			fmt.Println(ui.Warn("No backups found in " + cfg.BackupDir))
			return nil
		}

		sort.Sort(sort.Reverse(sort.StringSlice(matches)))

		fmt.Println()
		fmt.Printf("  %s\n\n", ui.Bold.Sprintf("%-44s  %8s  %-5s  %s", "FILE", "SIZE", "TYPE", "MODIFIED"))
		fmt.Println("  " + strings.Repeat("─", 75))

		for _, m := range matches {
			info, err := os.Stat(m)
			if err != nil {
				continue
			}
			age := time.Since(info.ModTime())
			modStr := ui.Dim.Sprint(info.ModTime().Format("2006-01-02 15:04"))
			sizeStr := ui.Cyan.Sprintf("%8s", humanSize(info.Size()))

			base := filepath.Base(m)
			typeStr := "db"
			if strings.HasSuffix(base, ".tar.gz") {
				typeStr = "path"
			}

			nameDisplay := base
			if age < 24*time.Hour {
				nameDisplay = ui.Green.Sprint(base)
			} else {
				nameDisplay = ui.White.Sprint(base)
			}

			fmt.Printf("  %-53s  %s  %-5s  %s\n", nameDisplay, sizeStr, ui.Dim.Sprint(typeStr), modStr)
		}
		fmt.Println()
		return nil
	},
}

func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func init() {
	backupCmd.AddCommand(backupListCmd)
}
