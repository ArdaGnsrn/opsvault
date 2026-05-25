package cmd

import (
	"fmt"
	"os"

	"github.com/ArdaGnsrn/opsvault/internal/backup"
	"github.com/ArdaGnsrn/opsvault/internal/config"
	"github.com/ArdaGnsrn/opsvault/internal/notify"
	"github.com/ArdaGnsrn/opsvault/internal/retention"
	"github.com/ArdaGnsrn/opsvault/internal/storage"
	"github.com/ArdaGnsrn/opsvault/internal/ui"
	"github.com/spf13/cobra"
)

var backupRunCmd = &cobra.Command{
	Use:   "run [database-name]",
	Short: "Run a backup immediately",
	Long:  "Run a backup for all enabled databases, or a specific named database.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		log := getLogger(cfg.LogLevel, cfg.LogFormat)

		var uploader storage.Uploader
		if cfg.Storage.Rclone.Enabled {
			uploader = storage.NewRcloneUploader(cfg.Storage.Rclone)
		}

		notifier := buildNotifier(cfg)
		mgr := backup.NewManager(cfg, uploader, notifier, log)

		ctx := cmd.Context()
		var results []backup.Result

		if len(args) > 0 {
			results = []backup.Result{mgr.RunOne(ctx, args[0])}
		} else {
			results = mgr.RunAll(ctx)
		}

		fmt.Println()
		hasErr := false
		for _, r := range results {
			dur := ui.Dim.Sprintf("(%s)", r.Duration.Round(1e6))
			if r.Err != nil {
				fmt.Fprintln(os.Stderr, ui.Fail(ui.Bold.Sprint(r.Database)+"  "+dur))
				fmt.Fprintln(os.Stderr, "       "+ui.Red.Sprint(r.Err.Error()))
				hasErr = true
			} else {
				fmt.Println(ui.OK(ui.Bold.Sprint(r.Database) + "  " + dur))
				fmt.Println("       " + ui.Dim.Sprint(r.Path))
			}
		}
		fmt.Println()

		if cfg.Retention.Local.Enabled {
			if err := retention.ApplyLocalWithPaths(cfg.Retention.Local, cfg.BackupDir, cfg.Databases, cfg.Paths, log); err != nil {
				fmt.Fprintln(os.Stderr, ui.Warn("retention: "+err.Error()))
			}
		}

		if cfg.Retention.Remote.Enabled && cfg.Storage.Rclone.Enabled {
			if err := retention.ApplyRemoteWithPaths(ctx, cfg.Retention, cfg.Storage.Rclone, cfg.Databases, cfg.Paths, log); err != nil {
				fmt.Fprintln(os.Stderr, ui.Warn("remote retention: "+err.Error()))
			}
		}

		if hasErr {
			os.Exit(1)
		}
		return nil
	},
}

func buildNotifier(cfg *config.Config) notify.Notifier {
	var notifiers []notify.Notifier
	if cfg.Notifications.Telegram.Enabled {
		notifiers = append(notifiers, notify.NewTelegramNotifier(cfg.Notifications.Telegram))
	}
	if cfg.Notifications.Email.Enabled {
		notifiers = append(notifiers, notify.NewEmailNotifier(cfg.Notifications.Email))
	}
	if len(notifiers) == 0 {
		return &notify.NoopNotifier{}
	}
	return notify.NewMultiNotifier(notifiers...)
}

func init() {
	backupCmd.AddCommand(backupRunCmd)
}
