package retention

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/ArdaGnsrn/opsvault/internal/config"
)

// ApplyRemote removes old remote backups via rclone according to keep_last and keep_days rules.
func ApplyRemote(ctx context.Context, cfg config.RetentionConfig, rcloneCfg config.RcloneConfig, databases []config.DatabaseConfig, log *slog.Logger) error {
	if !cfg.Remote.Enabled {
		return nil
	}

	hostname, _ := os.Hostname()

	for _, db := range databases {
		if !db.Enabled {
			continue
		}
		remotePath := fmt.Sprintf("%s:%s", rcloneCfg.Remote,
			expandPath(rcloneCfg.Path, hostname, db.Name, ""))

		if cfg.Remote.KeepDays > 0 {
			if err := pruneRemoteByAge(ctx, rcloneCfg, remotePath, db.Name, cfg.Remote.KeepDays, log); err != nil {
				log.Warn("failed to prune remote by age", "database", db.Name, "error", err)
			}
		}

		if cfg.Remote.KeepLast > 0 {
			if err := pruneRemoteByCount(ctx, rcloneCfg, remotePath, db.Name, cfg.Remote.KeepLast, log); err != nil {
				log.Warn("failed to prune remote by count", "database", db.Name, "error", err)
			}
		}
	}
	return nil
}

// pruneRemoteByAge uses rclone's --min-age flag to delete files older than keepDays.
func pruneRemoteByAge(ctx context.Context, rcloneCfg config.RcloneConfig, remotePath, dbName string, keepDays int, log *slog.Logger) error {
	log.Info("pruning remote backups by age", "database", dbName, "keep_days", keepDays, "path", remotePath)

	args := []string{"delete", fmt.Sprintf("--min-age=%dd", keepDays), "--include", dbName + "_*.sql.gz"}
	if rcloneCfg.RcloneConfig != "" {
		args = append(args, "--config", rcloneCfg.RcloneConfig)
	}
	args = append(args, remotePath)

	out, err := exec.CommandContext(ctx, "rclone", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("rclone delete --min-age: %w\noutput: %s", err, string(out))
	}
	return nil
}

// pruneRemoteByCount lists remote files and deletes the oldest ones beyond keepLast.
func pruneRemoteByCount(ctx context.Context, rcloneCfg config.RcloneConfig, remotePath, dbName string, keepLast int, log *slog.Logger) error {
	lsArgs := []string{"lsf", "--files-only", "--include", dbName + "_*.sql.gz"}
	if rcloneCfg.RcloneConfig != "" {
		lsArgs = append(lsArgs, "--config", rcloneCfg.RcloneConfig)
	}
	lsArgs = append(lsArgs, remotePath)

	out, err := exec.CommandContext(ctx, "rclone", lsArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("rclone lsf: %w\noutput: %s", err, string(out))
	}

	var files []string
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			files = append(files, l)
		}
	}

	if len(files) <= keepLast {
		return nil
	}

	sort.Strings(files)
	toDelete := files[:len(files)-keepLast]
	log.Info("pruning remote backups by count", "database", dbName, "deleting", len(toDelete))

	for _, f := range toDelete {
		fullPath := remotePath + "/" + f
		delArgs := []string{"deletefile"}
		if rcloneCfg.RcloneConfig != "" {
			delArgs = append(delArgs, "--config", rcloneCfg.RcloneConfig)
		}
		delArgs = append(delArgs, fullPath)

		if out, err := exec.CommandContext(ctx, "rclone", delArgs...).CombinedOutput(); err != nil {
			log.Warn("failed to delete remote file", "path", fullPath, "output", string(out))
		} else {
			log.Info("deleted remote backup", "file", f, "database", dbName)
		}
	}
	return nil
}

func expandPath(tpl, hostname, name, date string) string {
	r := strings.NewReplacer("{hostname}", hostname, "{name}", name, "{date}", date)
	return r.Replace(tpl)
}
