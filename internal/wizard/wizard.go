package wizard

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ArdaGnsrn/opsvault/internal/config"
	"github.com/charmbracelet/huh"
)

// Run launches the interactive config wizard for the given config file path.
func Run(cfgPath string) error {
	cfg, isNew, err := loadOrNew(cfgPath)
	if err != nil {
		return err
	}
	if isNew {
		fmt.Println("No config found — starting with defaults. Use 'Save & Exit' to write the file.")
		fmt.Println()
	}

	for {
		action, err := mainMenu(cfgPath)
		if err != nil {
			return nil
		}
		switch action {
		case "general":
			_ = editGeneral(cfg)
		case "databases":
			_ = editDatabases(cfg)
		case "storage":
			_ = editStorage(cfg)
		case "retention":
			_ = editRetention(cfg)
		case "notifications":
			_ = editNotifications(cfg)
		case "save":
			if err := config.WriteFile(cfgPath, cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
			fmt.Printf("\nConfig saved → %s\n", cfgPath)
			return nil
		case "exit":
			return nil
		}
	}
}

func loadOrNew(cfgPath string) (*config.Config, bool, error) {
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return config.Defaults(), true, nil
	}
	cfg, err := config.LoadRaw(cfgPath)
	if err != nil {
		return nil, false, err
	}
	return cfg, false, nil
}

func mainMenu(cfgPath string) (string, error) {
	var choice string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("OpsVault Config Wizard").
				Description("Config: "+cfgPath+"\nUse ↑↓ to navigate · Enter to select").
				Options(
					huh.NewOption("1  General Settings   (backup dir, schedule, log)", "general"),
					huh.NewOption("2  Databases          (add / edit / remove)", "databases"),
					huh.NewOption("3  Storage            (rclone remote)", "storage"),
					huh.NewOption("4  Retention          (local & remote cleanup)", "retention"),
					huh.NewOption("5  Notifications      (Telegram / Email)", "notifications"),
					huh.NewOption("6  Save & Exit", "save"),
					huh.NewOption("7  Exit without saving", "exit"),
				).
				Value(&choice),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	return choice, nil
}

// ── General ──────────────────────────────────────────────────────────────────

func editGeneral(cfg *config.Config) error {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Backup directory").
				Value(&cfg.BackupDir),
			huh.NewInput().
				Title("Cron schedule").
				Description("5-field cron expression, e.g. '0 2 * * *' for daily at 02:00").
				Value(&cfg.Schedule),
			huh.NewSelect[string]().
				Title("Log level").
				Options(
					huh.NewOption("debug", "debug"),
					huh.NewOption("info", "info"),
					huh.NewOption("warn", "warn"),
					huh.NewOption("error", "error"),
				).
				Value(&cfg.LogLevel),
			huh.NewSelect[string]().
				Title("Log format").
				Options(
					huh.NewOption("json (for journald / log aggregators)", "json"),
					huh.NewOption("text (human-readable)", "text"),
				).
				Value(&cfg.LogFormat),
		),
	)
	return ignoreAbort(form.Run())
}

// ── Databases ────────────────────────────────────────────────────────────────

func editDatabases(cfg *config.Config) error {
	for {
		opts := make([]huh.Option[int], 0, len(cfg.Databases)+2)
		for i, db := range cfg.Databases {
			status := "enabled"
			if !db.Enabled {
				status = "disabled"
			}
			label := fmt.Sprintf("%-18s  %s  %s:%d/%s  (%s)",
				db.Name, db.Type, db.Host, db.Port, db.Database, status)
			opts = append(opts, huh.NewOption(label, i))
		}
		opts = append(opts, huh.NewOption("+ Add new database", -1))
		opts = append(opts, huh.NewOption("← Back to main menu", -2))

		var idx int
		form := huh.NewForm(huh.NewGroup(
			huh.NewSelect[int]().
				Title("Databases").
				Description("Select a database to edit, or add a new one").
				Options(opts...).
				Value(&idx),
		))
		if err := form.Run(); err != nil || idx == -2 {
			return nil
		}

		if idx == -1 {
			db := config.DatabaseConfig{
				Enabled: true,
				Type:    "postgres",
				Host:    "127.0.0.1",
				Port:    5432,
			}
			deleted, err := editOneDatabase(&db, true)
			if err != nil || deleted {
				continue
			}
			cfg.Databases = append(cfg.Databases, db)
		} else {
			db := cfg.Databases[idx]
			deleted, err := editOneDatabase(&db, false)
			if err != nil {
				continue
			}
			if deleted {
				cfg.Databases = append(cfg.Databases[:idx], cfg.Databases[idx+1:]...)
			} else {
				cfg.Databases[idx] = db
			}
		}
	}
}

func editOneDatabase(db *config.DatabaseConfig, isNew bool) (deleted bool, err error) {
	portStr := strconv.Itoa(db.Port)
	if portStr == "0" {
		portStr = ""
	}
	dbType := db.Type

	title := "Edit database"
	if isNew {
		title = "Add database"
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Name (unique identifier, e.g. myapp_prod)").
				Value(&db.Name),
			huh.NewSelect[string]().
				Title("Database type").
				Options(
					huh.NewOption("PostgreSQL", "postgres"),
					huh.NewOption("MySQL", "mysql"),
				).
				Value(&dbType),
			huh.NewInput().Title("Host").Value(&db.Host),
			huh.NewInput().
				Title("Port (leave blank for default: 5432 / 3306)").
				Value(&portStr),
			huh.NewInput().Title("Database user").Value(&db.User),
			huh.NewInput().Title("Database name").Value(&db.Database),
		).Title(title),
		huh.NewGroup(
			huh.NewInput().
				Title("Password env var (recommended, e.g. DB_PASS)").
				Description("The env var that holds the database password").
				Value(&db.PasswordEnv),
			huh.NewInput().
				Title("Extra dump options (optional)").
				Description("Extra flags passed to mysqldump / pg_dump").
				Value(&db.ExtraOpts),
			huh.NewConfirm().
				Title("Enable this database?").
				Value(&db.Enabled),
		),
	)

	if err := form.Run(); err != nil {
		return false, nil
	}

	db.Type = dbType

	if p := strings.TrimSpace(portStr); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			db.Port = v
		}
	} else {
		if dbType == "mysql" {
			db.Port = 3306
		} else {
			db.Port = 5432
		}
	}

	if !isNew {
		var del bool
		delForm := huh.NewForm(huh.NewGroup(
			huh.NewConfirm().
				Title("Delete this database entry?").
				Description("Removes it from the config (does not drop the database)").
				Value(&del),
		))
		if err := delForm.Run(); err == nil && del {
			return true, nil
		}
	}

	return false, nil
}

// ── Storage ──────────────────────────────────────────────────────────────────

func editStorage(cfg *config.Config) error {
	r := &cfg.Storage.Rclone
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable rclone upload?").
				Value(&r.Enabled),
			huh.NewInput().
				Title("Remote name").
				Description("As configured in rclone, e.g. 's3backup' or 'gdrive'").
				Value(&r.Remote),
			huh.NewInput().
				Title("Remote path template").
				Description("Placeholders: {hostname} {name} {date}").
				Value(&r.Path),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("rclone config file path (leave blank for default)").
				Value(&r.RcloneConfig),
			huh.NewInput().
				Title("Extra rclone args (optional)").
				Value(&r.ExtraArgs),
			huh.NewConfirm().
				Title("Delete local backup after successful upload?").
				Value(&r.DeleteAfterUpload),
		),
	)
	return ignoreAbort(form.Run())
}

// ── Retention ────────────────────────────────────────────────────────────────

func editRetention(cfg *config.Config) error {
	l := &cfg.Retention.Local
	r := &cfg.Retention.Remote

	lKeepLast := intStr(l.KeepLast)
	lKeepDays := intStr(l.KeepDays)
	rKeepLast := intStr(r.KeepLast)
	rKeepDays := intStr(r.KeepDays)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().Title("Enable local retention?").Value(&l.Enabled),
			huh.NewInput().
				Title("Keep last N local backups (0 = off)").
				Value(&lKeepLast),
			huh.NewInput().
				Title("Delete local backups older than N days (0 = off)").
				Value(&lKeepDays),
		).Title("Local Retention"),
		huh.NewGroup(
			huh.NewConfirm().Title("Enable remote retention?").Value(&r.Enabled),
			huh.NewInput().
				Title("Keep last N remote backups (0 = off)").
				Value(&rKeepLast),
			huh.NewInput().
				Title("Delete remote backups older than N days (0 = off)").
				Value(&rKeepDays),
		).Title("Remote Retention"),
	)

	if err := ignoreAbort(form.Run()); err != nil {
		return err
	}

	l.KeepLast = parseIntOr(lKeepLast, l.KeepLast)
	l.KeepDays = parseIntOr(lKeepDays, l.KeepDays)
	r.KeepLast = parseIntOr(rKeepLast, r.KeepLast)
	r.KeepDays = parseIntOr(rKeepDays, r.KeepDays)
	return nil
}

// ── Notifications ────────────────────────────────────────────────────────────

func editNotifications(cfg *config.Config) error {
	n := &cfg.Notifications
	toStr := strings.Join(n.Email.To, ", ")
	smtpPortStr := intStr(n.Email.SMTPPort)
	if smtpPortStr == "0" {
		smtpPortStr = "587"
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().Title("Notify on success?").Value(&n.OnSuccess),
			huh.NewConfirm().Title("Notify on failure?").Value(&n.OnFailure),
		).Title("General"),
		huh.NewGroup(
			huh.NewConfirm().Title("Enable Telegram?").Value(&n.Telegram.Enabled),
			huh.NewInput().
				Title("Bot token env var (e.g. TELEGRAM_TOKEN)").
				Value(&n.Telegram.BotTokenEnv),
			huh.NewInput().
				Title("Chat ID").
				Value(&n.Telegram.ChatID),
		).Title("Telegram"),
		huh.NewGroup(
			huh.NewConfirm().Title("Enable Email?").Value(&n.Email.Enabled),
			huh.NewInput().Title("SMTP host").Value(&n.Email.SMTPHost),
			huh.NewInput().Title("SMTP port (e.g. 587)").Value(&smtpPortStr),
			huh.NewConfirm().Title("Use TLS / STARTTLS?").Value(&n.Email.SMTPTLS),
			huh.NewInput().Title("From address").Value(&n.Email.From),
			huh.NewInput().
				Title("To addresses (comma separated)").
				Value(&toStr),
			huh.NewInput().Title("SMTP username").Value(&n.Email.Username),
			huh.NewInput().
				Title("SMTP password env var (e.g. SMTP_PASS)").
				Value(&n.Email.PasswordEnv),
		).Title("Email"),
	)

	if err := ignoreAbort(form.Run()); err != nil {
		return err
	}

	n.Email.SMTPPort = parseIntOr(smtpPortStr, n.Email.SMTPPort)

	n.Email.To = nil
	for _, p := range strings.Split(toStr, ",") {
		if s := strings.TrimSpace(p); s != "" {
			n.Email.To = append(n.Email.To, s)
		}
	}

	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func ignoreAbort(err error) error {
	if errors.Is(err, huh.ErrUserAborted) {
		return nil
	}
	return err
}

func intStr(n int) string {
	return strconv.Itoa(n)
}

func parseIntOr(s string, fallback int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return v
	}
	return fallback
}
