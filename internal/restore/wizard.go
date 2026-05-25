package restore

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ArdaGnsrn/opsvault/internal/config"
	"github.com/ArdaGnsrn/opsvault/internal/ui"
	"github.com/charmbracelet/huh"
)

type rcloneEntry struct {
	Path    string    `json:"Path"`
	Name    string    `json:"Name"`
	Size    int64     `json:"Size"`
	ModTime time.Time `json:"ModTime"`
	IsDir   bool      `json:"IsDir"`
}

func RunWizard(ctx context.Context, cfg *config.Config) error {
	// Step 1: source
	var source string
	{
		localLabel := fmt.Sprintf("Local   (%s)", cfg.BackupDir)
		remoteLabel := "Remote  (rclone: not enabled)"
		if cfg.Storage.Rclone.Enabled {
			remoteLabel = fmt.Sprintf("Remote  (rclone: %s)", cfg.Storage.Rclone.Remote)
		}
		form := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Restore — Select source").
				Options(
					huh.NewOption(localLabel, "local"),
					huh.NewOption(remoteLabel, "remote"),
				).
				Value(&source),
		))
		if err := form.Run(); err != nil {
			return nil
		}
	}

	var filePath string
	var fileSize int64
	var fileMod time.Time
	var tmpDir string

	switch source {
	case "local":
		p, sz, mod, err := pickLocalFile(cfg.BackupDir)
		if err != nil {
			return err
		}
		if p == "" {
			return nil
		}
		filePath, fileSize, fileMod = p, sz, mod

	case "remote":
		if !cfg.Storage.Rclone.Enabled {
			fmt.Println(ui.Warn("Rclone storage is not enabled in config."))
			return nil
		}
		p, sz, mod, td, err := pickRemoteFile(ctx, cfg)
		if err != nil {
			return err
		}
		if p == "" {
			return nil
		}
		filePath, fileSize, fileMod = p, sz, mod
		tmpDir = td
		if tmpDir != "" {
			defer os.RemoveAll(tmpDir)
		}
	}

	// Step 2: select target database
	db, err := pickDatabase(cfg)
	if err != nil {
		return err
	}
	if db == nil {
		return nil
	}

	// Step 3: confirm — critical, explicit yes/no required
	if !confirmRestore(db, filePath, fileSize, fileMod) {
		fmt.Println(ui.Warn("Restore cancelled."))
		return nil
	}

	r := New(db.Type)
	if r == nil {
		return fmt.Errorf("unsupported database type: %s", db.Type)
	}
	fmt.Println(ui.Info(fmt.Sprintf("Restoring %s → %s...", filepath.Base(filePath), db.Database)))
	if err := r.Restore(ctx, *db, filePath); err != nil {
		return err
	}
	fmt.Println(ui.OK("Restore complete."))
	return nil
}

func pickLocalFile(backupDir string) (string, int64, time.Time, error) {
	matches, _ := filepath.Glob(filepath.Join(backupDir, "*.sql.gz"))
	if len(matches) == 0 {
		fmt.Println(ui.Warn("No local backups found in " + backupDir))
		return "", 0, time.Time{}, nil
	}

	type entry struct {
		path string
		size int64
		mod  time.Time
	}
	entries := make([]entry, 0, len(matches))
	for _, m := range matches {
		fi, err := os.Stat(m)
		if err != nil {
			continue
		}
		entries = append(entries, entry{m, fi.Size(), fi.ModTime()})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].mod.After(entries[j].mod)
	})

	opts := make([]huh.Option[string], len(entries))
	for i, e := range entries {
		label := fmt.Sprintf("%-44s  %8s  %s",
			filepath.Base(e.path), wsize(e.size), e.mod.Format("2006-01-02 15:04"))
		opts[i] = huh.NewOption(label, e.path)
	}

	var selected string
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Select local backup file").
			Description("Directory: "+backupDir).
			Options(opts...).
			Value(&selected),
	))
	if err := form.Run(); err != nil {
		return "", 0, time.Time{}, nil
	}

	for _, e := range entries {
		if e.path == selected {
			return e.path, e.size, e.mod, nil
		}
	}
	return selected, 0, time.Time{}, nil
}

func pickRemoteFile(ctx context.Context, cfg *config.Config) (localPath string, size int64, mod time.Time, tmpDir string, err error) {
	r := cfg.Storage.Rclone
	basePath := rcloneBasePath(r.Path)

	remoteRoot := r.Remote + ":"
	if basePath != "" {
		remoteRoot = r.Remote + ":" + basePath
	}

	fmt.Println(ui.Info("Listing remote backups from " + remoteRoot + " ..."))

	lsArgs := []string{"lsjson", "--recursive"}
	if r.RcloneConfig != "" {
		lsArgs = append(lsArgs, "--config", r.RcloneConfig)
	}
	lsArgs = append(lsArgs, remoteRoot)

	out, lsErr := exec.CommandContext(ctx, "rclone", lsArgs...).Output()
	if lsErr != nil {
		return "", 0, time.Time{}, "", fmt.Errorf("rclone lsjson: %w", lsErr)
	}

	var all []rcloneEntry
	if jsonErr := json.Unmarshal(out, &all); jsonErr != nil {
		return "", 0, time.Time{}, "", fmt.Errorf("parsing rclone output: %w", jsonErr)
	}

	var backups []rcloneEntry
	for _, f := range all {
		if !f.IsDir && strings.HasSuffix(f.Name, ".sql.gz") {
			backups = append(backups, f)
		}
	}
	if len(backups) == 0 {
		fmt.Println(ui.Warn("No remote backup files found."))
		return "", 0, time.Time{}, "", nil
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime.After(backups[j].ModTime)
	})

	opts := make([]huh.Option[string], len(backups))
	for i, f := range backups {
		label := fmt.Sprintf("%-44s  %8s  %s",
			f.Name, wsize(f.Size), f.ModTime.Format("2006-01-02 15:04"))
		opts[i] = huh.NewOption(label, f.Path)
	}

	var selectedPath string
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Select remote backup file").
			Description("Remote: "+remoteRoot).
			Options(opts...).
			Value(&selectedPath),
	))
	if formErr := form.Run(); formErr != nil {
		return "", 0, time.Time{}, "", nil
	}

	var selected rcloneEntry
	for _, f := range backups {
		if f.Path == selectedPath {
			selected = f
			break
		}
	}

	remoteFileDir := path.Dir(selected.Path)
	remoteFileName := path.Base(selected.Path)
	var remoteDownloadDir string
	switch {
	case remoteFileDir == ".":
		remoteDownloadDir = basePath
	case basePath == "":
		remoteDownloadDir = remoteFileDir
	default:
		remoteDownloadDir = basePath + "/" + remoteFileDir
	}

	var remoteFilePath string
	if remoteDownloadDir == "" {
		remoteFilePath = r.Remote + ":" + remoteFileName
	} else {
		remoteFilePath = r.Remote + ":" + remoteDownloadDir + "/" + remoteFileName
	}

	cacheDir := restoreCacheDir()
	_ = os.MkdirAll(cacheDir, 0700)
	cacheFile := filepath.Join(cacheDir, remoteFileName)

	fmt.Println(ui.Info("Checking remote hash..."))
	if remoteHash, hErr := rcloneHashSHA1(ctx, r, remoteFilePath); hErr == nil && remoteHash != "" {
		if localHash, lErr := sha1File(cacheFile); lErr == nil && localHash == remoteHash {
			fmt.Println(ui.OK("Using cached file (SHA1 match)."))
			fi, _ := os.Stat(cacheFile)
			return cacheFile, fi.Size(), selected.ModTime, "", nil
		}
	}

	fmt.Println(ui.Info("Downloading " + remoteFileName + "..."))

	dlArgs := []string{"copy"}
	if r.RcloneConfig != "" {
		dlArgs = append(dlArgs, "--config", r.RcloneConfig)
	}
	dlArgs = append(dlArgs, r.Remote+":"+remoteDownloadDir, cacheDir, "--include", remoteFileName)

	if dlOut, dlErr := exec.CommandContext(ctx, "rclone", dlArgs...).CombinedOutput(); dlErr != nil {
		return "", 0, time.Time{}, "", fmt.Errorf("rclone download failed: %w\n%s", dlErr, dlOut)
	}

	fi, statErr := os.Stat(cacheFile)
	if statErr != nil {
		return "", 0, time.Time{}, "", fmt.Errorf("downloaded file not found: %w", statErr)
	}

	return cacheFile, fi.Size(), selected.ModTime, "", nil
}

func pickDatabase(cfg *config.Config) (*config.DatabaseConfig, error) {
	opts := make([]huh.Option[int], 0, len(cfg.Databases)+1)
	for i, db := range cfg.Databases {
		label := fmt.Sprintf("%-18s  %s  %s:%d/%s",
			db.Name, db.Type, db.Host, db.Port, db.Database)
		opts = append(opts, huh.NewOption(label, i))
	}
	opts = append(opts, huh.NewOption("+ Enter custom database details", -1))

	var idx int
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[int]().
			Title("Select target database").
			Description("All existing data in this database will be replaced.").
			Options(opts...).
			Value(&idx),
	))
	if err := form.Run(); err != nil {
		return nil, nil
	}
	if idx >= 0 {
		return &cfg.Databases[idx], nil
	}
	return enterCustomDatabase()
}

func enterCustomDatabase() (*config.DatabaseConfig, error) {
	db := config.DatabaseConfig{
		Type: "postgres",
		Host: "127.0.0.1",
		Port: 5432,
	}
	portStr := "5432"

	for {
		var choice string
		form := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Custom Database").
				Description("Enter connection details for the target database.").
				Options(
					huh.NewOption(fmt.Sprintf("Type          %s", db.Type), "type"),
					huh.NewOption(fmt.Sprintf("Host          %s", preview(db.Host)), "host"),
					huh.NewOption(fmt.Sprintf("Port          %s", portStr), "port"),
					huh.NewOption(fmt.Sprintf("Database      %s", preview(db.Database)), "database"),
					huh.NewOption(fmt.Sprintf("User          %s", preview(db.User)), "user"),
					huh.NewOption(fmt.Sprintf("Password      %s", maskPassword(db.Password)), "password"),
					huh.NewOption("✓  Use these settings", "confirm"),
					huh.NewOption("← Back", "back"),
				).
				Value(&choice),
		))
		if err := form.Run(); err != nil || choice == "back" {
			return nil, nil
		}

		switch choice {
		case "type":
			f := huh.NewForm(huh.NewGroup(
				huh.NewSelect[string]().Title("Database type").
					Options(huh.NewOption("PostgreSQL", "postgres"), huh.NewOption("MySQL", "mysql")).
					Value(&db.Type),
			))
			if err := ignoreAbort(f.Run()); err == nil {
				if portStr == "5432" && db.Type == "mysql" {
					portStr = "3306"
				} else if portStr == "3306" && db.Type == "postgres" {
					portStr = "5432"
				}
			}
		case "host":
			f := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Host").Value(&db.Host)))
			_ = ignoreAbort(f.Run())
		case "port":
			f := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Port").Value(&portStr)))
			_ = ignoreAbort(f.Run())
		case "database":
			f := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Database name").Value(&db.Database)))
			_ = ignoreAbort(f.Run())
		case "user":
			f := huh.NewForm(huh.NewGroup(huh.NewInput().Title("User").Value(&db.User)))
			_ = ignoreAbort(f.Run())
		case "password":
			f := huh.NewForm(huh.NewGroup(
				huh.NewInput().Title("Password").EchoMode(huh.EchoModePassword).Value(&db.Password),
			))
			_ = ignoreAbort(f.Run())
		case "confirm":
			if db.Database == "" || db.User == "" {
				fmt.Println(ui.Warn("Database name and user are required."))
				continue
			}
			if p, err := strconv.Atoi(strings.TrimSpace(portStr)); err == nil {
				db.Port = p
			}
			db.Name = fmt.Sprintf("custom:%s/%s", db.Host, db.Database)
			return &db, nil
		}
	}
}

func confirmRestore(db *config.DatabaseConfig, filePath string, size int64, mod time.Time) bool {
	desc := fmt.Sprintf(
		"  File:      %s\n  Size:      %s\n  Modified:  %s\n\n  Database:  %s (%s)\n  Host:      %s:%d  /  %s",
		filepath.Base(filePath), wsize(size), mod.Format("2006-01-02 15:04:05"),
		db.Name, db.Type, db.Host, db.Port, db.Database,
	)
	var confirmed bool
	form := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title("WARNING: This will permanently overwrite the target database. Continue?").
			Description(desc).
			Affirmative("Yes, restore now").
			Negative("Cancel").
			Value(&confirmed),
	))
	if err := form.Run(); err != nil {
		return false
	}
	return confirmed
}

func ignoreAbort(err error) error {
	if errors.Is(err, huh.ErrUserAborted) {
		return nil
	}
	return err
}

func preview(s string) string {
	if s == "" {
		return "(not set)"
	}
	if len(s) > 30 {
		return s[:27] + "..."
	}
	return s
}

func maskPassword(s string) string {
	if s == "" {
		return "(not set)"
	}
	return strings.Repeat("*", len(s))
}

func restoreCacheDir() string {
	if d, err := os.UserCacheDir(); err == nil {
		return filepath.Join(d, "opsvault", "restore")
	}
	return filepath.Join(os.TempDir(), "opsvault-restore-cache")
}

func rcloneHashSHA1(ctx context.Context, r config.RcloneConfig, remotePath string) (string, error) {
	args := []string{"hashsum", "SHA1"}
	if r.RcloneConfig != "" {
		args = append(args, "--config", r.RcloneConfig)
	}
	args = append(args, remotePath)
	out, err := exec.CommandContext(ctx, "rclone", args...).Output()
	if err != nil {
		return "", err
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 1 || fields[0] == "UNSUPPORTED" {
		return "", nil
	}
	return fields[0], nil
}

func sha1File(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func rcloneBasePath(tpl string) string {
	idx := strings.Index(tpl, "{")
	if idx < 0 {
		return strings.Trim(tpl, "/")
	}
	return strings.Trim(tpl[:idx], "/")
}

func wsize(b int64) string {
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
