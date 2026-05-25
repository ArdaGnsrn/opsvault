package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ArdaGnsrn/opsvault/internal/config"
	"github.com/ArdaGnsrn/opsvault/internal/restore"
	"github.com/ArdaGnsrn/opsvault/internal/ui"
	"github.com/spf13/cobra"
)

var (
	restoreDBName string
	restoreFile   string
	restoreYes    bool
)

var restoreRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Restore a database from a local backup file",
	Long:  "Restores a gzip-compressed SQL dump to the target database.",
	Example: `  opsvault restore run --name myapp --file /var/backups/opsvault/myapp_20240115_020000.sql.gz
  opsvault restore run --name myapp --file ./myapp_backup.sql.gz --yes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		// Find the database config
		var dbCfg *config.DatabaseConfig
		for i, db := range cfg.Databases {
			if db.Name == restoreDBName {
				dbCfg = &cfg.Databases[i]
				break
			}
		}
		if dbCfg == nil {
			return fmt.Errorf("database %q not found in config", restoreDBName)
		}

		// Validate file exists
		fi, err := os.Stat(restoreFile)
		if err != nil {
			return fmt.Errorf("backup file: %w", err)
		}

		r := restore.New(dbCfg.Type)
		if r == nil {
			return fmt.Errorf("unsupported database type: %s", dbCfg.Type)
		}

		fmt.Println()
		fmt.Println(ui.Warn("This will overwrite the target database. All existing data will be replaced."))
		fmt.Println()
		fmt.Printf("  %s  %s\n", ui.Bold.Sprint("Database:"), dbCfg.Name)
		fmt.Printf("  %s  %s (%s @ %s:%d)\n", ui.Bold.Sprint("Target:  "), dbCfg.Database, dbCfg.User, dbCfg.Host, dbCfg.Port)
		fmt.Printf("  %s  %s (%s)\n", ui.Bold.Sprint("File:    "), restoreFile, humanSize(fi.Size()))
		fmt.Println()

		if !restoreYes {
			fmt.Printf("  %s Type %s to confirm: ", ui.Cyan.Sprint("?"), ui.Bold.Sprint("yes"))
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			if strings.TrimSpace(scanner.Text()) != "yes" {
				fmt.Println(ui.Warn("Aborted."))
				return nil
			}
		}

		fmt.Println(ui.Info(fmt.Sprintf("Restoring %s → %s...", restoreFile, dbCfg.Database)))

		if err := r.Restore(cmd.Context(), *dbCfg, restoreFile); err != nil {
			return err
		}

		fmt.Println(ui.OK("Restore complete."))
		return nil
	},
}

func init() {
	restoreRunCmd.Flags().StringVar(&restoreDBName, "name", "", "database name from config (required)")
	restoreRunCmd.Flags().StringVar(&restoreFile, "file", "", "path to .sql.gz backup file (required)")
	restoreRunCmd.Flags().BoolVarP(&restoreYes, "yes", "y", false, "skip confirmation prompt")
	_ = restoreRunCmd.MarkFlagRequired("name")
	_ = restoreRunCmd.MarkFlagRequired("file")
	restoreCmd.AddCommand(restoreRunCmd)
}
