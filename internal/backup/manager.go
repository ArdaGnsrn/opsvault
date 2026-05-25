package backup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/ArdaGnsrn/opsvault/internal/config"
	"github.com/ArdaGnsrn/opsvault/internal/history"
	"github.com/ArdaGnsrn/opsvault/internal/notify"
	"github.com/ArdaGnsrn/opsvault/internal/storage"
)

// Result holds the outcome of a single database backup.
type Result struct {
	Database string
	Path     string
	Duration time.Duration
	Err      error
}

// Manager orchestrates backups for all configured databases.
type Manager struct {
	cfg      *config.Config
	uploader storage.Uploader
	notifier notify.Notifier
	log      *slog.Logger
}

func NewManager(cfg *config.Config, uploader storage.Uploader, notifier notify.Notifier, log *slog.Logger) *Manager {
	return &Manager{
		cfg:      cfg,
		uploader: uploader,
		notifier: notifier,
		log:      log,
	}
}

// RunAll backs up all enabled databases and paths. Never stops on first failure.
func (m *Manager) RunAll(ctx context.Context) []Result {
	var results []Result
	for _, db := range m.cfg.Databases {
		if !db.Enabled {
			m.log.Info("skipping disabled database", "name", db.Name)
			continue
		}
		r := m.runOne(ctx, db)
		results = append(results, r)
		m.sendNotification(ctx, db, r)
	}
	for _, p := range m.cfg.Paths {
		if !p.Enabled {
			m.log.Info("skipping disabled path", "name", p.Name)
			continue
		}
		r := m.runOnePath(ctx, p)
		results = append(results, r)
		m.sendPathNotification(ctx, p, r)
	}
	return results
}

// RunOne backs up a single named database or path.
func (m *Manager) RunOne(ctx context.Context, name string) Result {
	for _, db := range m.cfg.Databases {
		if db.Name == name {
			r := m.runOne(ctx, db)
			m.sendNotification(ctx, db, r)
			return r
		}
	}
	for _, p := range m.cfg.Paths {
		if p.Name == name {
			r := m.runOnePath(ctx, p)
			m.sendPathNotification(ctx, p, r)
			return r
		}
	}
	return Result{Database: name, Err: fmt.Errorf("%q not found in databases or paths", name)}
}

func (m *Manager) runOne(ctx context.Context, db config.DatabaseConfig) Result {
	start := time.Now()
	log := m.log.With("database", db.Name, "type", db.Type)

	log.Info("starting backup")

	if err := os.MkdirAll(m.cfg.BackupDir, 0750); err != nil {
		return Result{Database: db.Name, Err: fmt.Errorf("creating backup dir: %w", err), Duration: time.Since(start)}
	}

	filename := BackupFilename(db, start)
	destPath := filepath.Join(m.cfg.BackupDir, filename)

	var dumper Dumper
	switch db.Type {
	case "mysql":
		dumper = &MySQLDumper{}
	case "postgres":
		dumper = &PostgresDumper{}
	default:
		return Result{Database: db.Name, Err: fmt.Errorf("unsupported database type: %s", db.Type), Duration: time.Since(start)}
	}

	if err := dumper.Dump(ctx, db, destPath); err != nil {
		log.Error("dump failed", "error", err)
		return Result{Database: db.Name, Err: err, Duration: time.Since(start)}
	}

	dur := time.Since(start)
	log.Info("dump complete", "path", destPath, "duration", dur.Round(time.Millisecond))

	result := Result{Database: db.Name, Path: destPath, Duration: dur}

	if m.cfg.Storage.Rclone.Enabled && m.uploader != nil {
		if err := m.upload(ctx, db.Name, destPath, start); err != nil {
			log.Error("upload failed", "error", err)
			result.Err = err
		}
	}

	m.recordHistory(start, result)
	return result
}

func (m *Manager) runOnePath(ctx context.Context, p config.PathConfig) Result {
	start := time.Now()
	log := m.log.With("path", p.Name, "src", p.Path)
	log.Info("starting path backup")

	if err := os.MkdirAll(m.cfg.BackupDir, 0750); err != nil {
		return Result{Database: p.Name, Err: fmt.Errorf("creating backup dir: %w", err), Duration: time.Since(start)}
	}

	filename := PathArchiveFilename(p, start)
	destPath := filepath.Join(m.cfg.BackupDir, filename)

	archiver := &PathArchiver{}
	if err := archiver.Archive(ctx, p, destPath); err != nil {
		log.Error("archive failed", "error", err)
		return Result{Database: p.Name, Err: err, Duration: time.Since(start)}
	}

	dur := time.Since(start)
	log.Info("archive complete", "path", destPath, "duration", dur.Round(time.Millisecond))

	result := Result{Database: p.Name, Path: destPath, Duration: dur}

	if m.cfg.Storage.Rclone.Enabled && m.uploader != nil {
		if err := m.upload(ctx, p.Name, destPath, start); err != nil {
			log.Error("upload failed", "error", err)
			result.Err = err
		}
	}

	m.recordHistory(start, result)
	return result
}

func (m *Manager) sendPathNotification(ctx context.Context, p config.PathConfig, r Result) {
	if m.notifier == nil {
		return
	}
	cfg := m.cfg.Notifications
	if r.Err != nil {
		if !cfg.OnFailure {
			return
		}
		msg := notify.Message{
			Level:        notify.LevelError,
			DatabaseName: p.Name,
			Subject:      fmt.Sprintf("Backup FAILED: %s", p.Name),
			Body:         fmt.Sprintf("Path backup of %q (%s) failed after %s.\n\nError: %v", p.Name, p.Path, r.Duration.Round(time.Second), r.Err),
			Timestamp:    time.Now(),
		}
		if err := m.notifier.Send(ctx, msg); err != nil {
			m.log.Warn("notification failed", "path", p.Name, "error", err)
		}
		return
	}
	if !cfg.OnSuccess {
		return
	}
	msg := notify.Message{
		Level:        notify.LevelInfo,
		DatabaseName: p.Name,
		Subject:      fmt.Sprintf("Backup OK: %s", p.Name),
		Body:         fmt.Sprintf("Path backup of %q (%s) completed in %s.\n\nFile: %s", p.Name, p.Path, r.Duration.Round(time.Second), r.Path),
		Timestamp:    time.Now(),
	}
	if err := m.notifier.Send(ctx, msg); err != nil {
		m.log.Warn("notification failed", "path", p.Name, "error", err)
	}
}

func (m *Manager) recordHistory(start time.Time, r Result) {
	e := history.Entry{
		Database:  r.Database,
		StartedAt: start,
		Duration:  r.Duration.Seconds(),
		FilePath:  r.Path,
	}
	if r.Path != "" {
		if fi, err := os.Stat(r.Path); err == nil {
			e.FileSize = fi.Size()
		}
	}
	if r.Err != nil {
		e.Status = history.StatusFailed
		e.Error = r.Err.Error()
	} else {
		e.Status = history.StatusSuccess
	}
	_ = history.Append(m.cfg.BackupDir, e)
}

func (m *Manager) upload(ctx context.Context, name, localPath string, t time.Time) error {
	hostname, _ := os.Hostname()
	vars := map[string]string{
		"hostname": hostname,
		"name":     name,
		"date":     t.UTC().Format("2006-01-02"),
	}
	if err := m.uploader.Upload(ctx, localPath, vars); err != nil {
		return err
	}
	if m.cfg.Storage.Rclone.DeleteAfterUpload {
		return os.Remove(localPath)
	}
	return nil
}

func (m *Manager) sendNotification(ctx context.Context, db config.DatabaseConfig, r Result) {
	if m.notifier == nil {
		return
	}

	cfg := m.cfg.Notifications
	if r.Err != nil {
		if !cfg.OnFailure {
			return
		}
		msg := notify.Message{
			Level:        notify.LevelError,
			DatabaseName: db.Name,
			Subject:      fmt.Sprintf("Backup FAILED: %s", db.Name),
			Body:         fmt.Sprintf("Backup of database %q failed after %s.\n\nError: %v", db.Name, r.Duration.Round(time.Second), r.Err),
			Timestamp:    time.Now(),
		}
		if err := m.notifier.Send(ctx, msg); err != nil {
			m.log.Warn("notification failed", "database", db.Name, "error", err)
		}
		return
	}

	if !cfg.OnSuccess {
		return
	}
	msg := notify.Message{
		Level:        notify.LevelInfo,
		DatabaseName: db.Name,
		Subject:      fmt.Sprintf("Backup OK: %s", db.Name),
		Body:         fmt.Sprintf("Backup of database %q completed in %s.\n\nFile: %s", db.Name, r.Duration.Round(time.Second), r.Path),
		Timestamp:    time.Now(),
	}
	if err := m.notifier.Send(ctx, msg); err != nil {
		m.log.Warn("notification failed", "database", db.Name, "error", err)
	}
}
