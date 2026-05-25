package retention

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ArdaGnsrn/opsvault/internal/config"
)

// ApplyLocal removes old backup files according to keep_last and keep_days rules.
func ApplyLocal(cfg config.LocalRetentionConfig, backupDir string, databases []config.DatabaseConfig, log *slog.Logger) error {
	return ApplyLocalWithPaths(cfg, backupDir, databases, nil, log)
}

// ApplyLocalWithPaths is like ApplyLocal but also prunes path archives.
func ApplyLocalWithPaths(cfg config.LocalRetentionConfig, backupDir string, databases []config.DatabaseConfig, paths []config.PathConfig, log *slog.Logger) error {
	if !cfg.Enabled {
		return nil
	}
	for _, db := range databases {
		if !db.Enabled {
			continue
		}
		if err := pruneLocal(backupDir, db.Name, ".sql.gz", cfg, log); err != nil {
			return fmt.Errorf("pruning local backups for %s: %w", db.Name, err)
		}
	}
	for _, p := range paths {
		if !p.Enabled {
			continue
		}
		if err := pruneLocal(backupDir, p.Name, ".tar.gz", cfg, log); err != nil {
			return fmt.Errorf("pruning local backups for %s: %w", p.Name, err)
		}
	}
	return nil
}

func pruneLocal(backupDir, name, ext string, cfg config.LocalRetentionConfig, log *slog.Logger) error {
	matches, err := filepath.Glob(filepath.Join(backupDir, name+"_*"+ext))
	if err != nil {
		return err
	}

	var files []string
	for _, m := range matches {
		if strings.HasPrefix(filepath.Base(m), name+"_") {
			files = append(files, m)
		}
	}

	// keep_days: delete files older than N days
	if cfg.KeepDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -cfg.KeepDays)
		var remaining []string
		for _, f := range files {
			info, err := os.Stat(f)
			if err != nil {
				remaining = append(remaining, f)
				continue
			}
			if info.ModTime().Before(cutoff) {
				log.Info("removing old backup (age)", "file", filepath.Base(f), "database", name, "age_days", int(time.Since(info.ModTime()).Hours()/24))
				if err := os.Remove(f); err != nil {
					log.Warn("failed to remove file", "file", f, "error", err)
				}
			} else {
				remaining = append(remaining, f)
			}
		}
		files = remaining
	}

	// keep_last: keep only the N most recent files
	if cfg.KeepLast > 0 && len(files) > cfg.KeepLast {
		sort.Strings(files) // lexicographic = chronological due to date in filename
		for _, f := range files[:len(files)-cfg.KeepLast] {
			log.Info("removing old backup (count)", "file", filepath.Base(f), "database", name)
			if err := os.Remove(f); err != nil {
				log.Warn("failed to remove file", "file", f, "error", err)
			}
		}
	}

	return nil
}
