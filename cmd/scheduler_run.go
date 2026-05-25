package cmd

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/ArdaGnsrn/opsvault/internal/backup"
	"github.com/ArdaGnsrn/opsvault/internal/config"
	"github.com/ArdaGnsrn/opsvault/internal/retention"
	"github.com/ArdaGnsrn/opsvault/internal/scheduler"
	"github.com/ArdaGnsrn/opsvault/internal/storage"
	"github.com/spf13/cobra"
)

var schedulerRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the backup scheduler daemon",
	Long:  "Start the long-running scheduler daemon. Typically called by systemd via the service unit.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		log := getLogger(cfg.LogLevel, cfg.LogFormat)

		if errs := config.Validate(cfg); len(errs) > 0 {
			for _, e := range errs {
				log.Error("config validation error", "error", e.Error())
			}
			return fmt.Errorf("config validation failed with %d error(s)", len(errs))
		}

		var uploader storage.Uploader
		if cfg.Storage.Rclone.Enabled {
			uploader = storage.NewRcloneUploader(cfg.Storage.Rclone)
		}

		notifier := buildNotifier(cfg)
		mgr := backup.NewManager(cfg, uploader, notifier, log)
		sched := scheduler.New(log)

		if err := sched.Add(cfg.Schedule, func(ctx context.Context) {
			log.Info("scheduled backup triggered", "schedule", cfg.Schedule)
			results := mgr.RunAll(ctx)

			for _, r := range results {
				if r.Err != nil {
					log.Error("backup failed", "database", r.Database, "error", r.Err)
				} else {
					log.Info("backup succeeded", "database", r.Database, "duration", r.Duration)
				}
			}

			if cfg.Retention.Local.Enabled {
				if err := retention.ApplyLocal(cfg.Retention.Local, cfg.BackupDir, cfg.Databases, log); err != nil {
					log.Warn("local retention error", "error", err)
				}
			}

			if cfg.Retention.Remote.Enabled && cfg.Storage.Rclone.Enabled {
				if err := retention.ApplyRemote(ctx, cfg.Retention, cfg.Storage.Rclone, cfg.Databases, log); err != nil {
					log.Warn("remote retention error", "error", err)
				}
			}
		}); err != nil {
			return fmt.Errorf("adding cron job: %w", err)
		}

		log.Info("opsvault daemon started", "schedule", cfg.Schedule, "databases", len(cfg.Databases))

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer stop()

		sched.Run(ctx)
		return nil
	},
}

func init() {
	schedulerCmd.AddCommand(schedulerRunCmd)
}
