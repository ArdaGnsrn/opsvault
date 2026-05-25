package wizard

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ArdaGnsrn/opsvault/internal/config"
	"github.com/ArdaGnsrn/opsvault/internal/envfile"
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

	envPath := envfile.PathFor(cfgPath)
	envVars := map[string]string{}
	_ = envfile.Load(envPath)

	for {
		action, err := mainMenu(cfgPath)
		if err != nil {
			return nil
		}
		switch action {
		case "general":
			_ = editGeneral(cfg)
		case "databases":
			_ = editDatabases(cfg, envVars)
		case "storage":
			_ = editStorage(cfg)
		case "retention":
			_ = editRetention(cfg)
		case "notifications":
			_ = editNotifications(cfg, envVars)
		case "save":
			if err := config.WriteFile(cfgPath, cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
			if err := envfile.Write(envPath, envVars); err != nil {
				return fmt.Errorf("saving env file: %w", err)
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
				Description(cfgPath+"\nYou can also edit this file directly with any text editor.\nUse ↑↓ to navigate · Enter to select").
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
	for {
		var choice string
		form := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("General Settings").
				Description("Select a field to edit · Esc to go back").
				Options(
					huh.NewOption(fmt.Sprintf("Backup directory      %s", preview(cfg.BackupDir)), "backup_dir"),
					huh.NewOption(fmt.Sprintf("Cron schedule         %s", preview(cfg.Schedule)), "schedule"),
					huh.NewOption(fmt.Sprintf("Log level             %s", cfg.LogLevel), "log_level"),
					huh.NewOption(fmt.Sprintf("Log format            %s", cfg.LogFormat), "log_format"),
					huh.NewOption("← Back", "back"),
				).
				Value(&choice),
		))
		if err := form.Run(); err != nil || choice == "back" {
			return nil
		}

		switch choice {
		case "backup_dir":
			f := huh.NewForm(huh.NewGroup(
				huh.NewInput().Title("Backup directory").Value(&cfg.BackupDir),
			))
			_ = ignoreAbort(f.Run())
		case "schedule":
			f := huh.NewForm(huh.NewGroup(
				huh.NewInput().
					Title("Cron schedule").
					Description("5-field cron expression, e.g. '0 2 * * *' for daily at 02:00").
					Value(&cfg.Schedule),
			))
			_ = ignoreAbort(f.Run())
		case "log_level":
			f := huh.NewForm(huh.NewGroup(
				huh.NewSelect[string]().
					Title("Log level").
					Options(
						huh.NewOption("debug", "debug"),
						huh.NewOption("info", "info"),
						huh.NewOption("warn", "warn"),
						huh.NewOption("error", "error"),
					).
					Value(&cfg.LogLevel),
			))
			_ = ignoreAbort(f.Run())
		case "log_format":
			f := huh.NewForm(huh.NewGroup(
				huh.NewSelect[string]().
					Title("Log format").
					Options(
						huh.NewOption("json (for journald / log aggregators)", "json"),
						huh.NewOption("text (human-readable)", "text"),
					).
					Value(&cfg.LogFormat),
			))
			_ = ignoreAbort(f.Run())
		}
	}
}

// ── Databases ────────────────────────────────────────────────────────────────

func editDatabases(cfg *config.Config, envVars map[string]string) error {
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
			deleted, err := editOneDatabase(&db, true, envVars)
			if err != nil || deleted {
				continue
			}
			cfg.Databases = append(cfg.Databases, db)
		} else {
			db := cfg.Databases[idx]
			deleted, err := editOneDatabase(&db, false, envVars)
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

func editOneDatabase(db *config.DatabaseConfig, isNew bool, envVars map[string]string) (deleted bool, err error) {
	portStr := strconv.Itoa(db.Port)
	if portStr == "0" {
		portStr = ""
	}

	title := "Edit database"
	if isNew {
		title = "Add database"
	}

	for {
		var choice string
		opts := []huh.Option[string]{
			huh.NewOption(fmt.Sprintf("Name             %s", preview(db.Name)), "name"),
			huh.NewOption(fmt.Sprintf("Type             %s", db.Type), "type"),
			huh.NewOption(fmt.Sprintf("Host             %s", preview(db.Host)), "host"),
			huh.NewOption(fmt.Sprintf("Port             %s", portStr), "port"),
			huh.NewOption(fmt.Sprintf("User             %s", preview(db.User)), "user"),
			huh.NewOption(fmt.Sprintf("Database         %s", preview(db.Database)), "database"),
			huh.NewOption(fmt.Sprintf("Password env     %s", preview(db.PasswordEnv)), "password_env"),
			huh.NewOption(fmt.Sprintf("Excluded tables  %s", preview(strings.Join(db.ExcludedTables, ", "))), "excluded_tables"),
			huh.NewOption(fmt.Sprintf("Extra opts       %s", preview(db.ExtraOpts)), "extra_opts"),
			huh.NewOption(fmt.Sprintf("Enabled          %s", yesNo(db.Enabled)), "enabled"),
		}
		if !isNew {
			opts = append(opts, huh.NewOption("Delete this database", "delete"))
		}
		opts = append(opts, huh.NewOption("← Back", "back"))

		form := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title(title).
				Description("Select a field to edit").
				Options(opts...).
				Value(&choice),
		))
		if err := form.Run(); err != nil || choice == "back" {
			return false, nil
		}

		switch choice {
		case "name":
			f := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Name (unique identifier, e.g. myapp_prod)").Value(&db.Name)))
			_ = ignoreAbort(f.Run())
		case "type":
			f := huh.NewForm(huh.NewGroup(
				huh.NewSelect[string]().Title("Database type").
					Options(huh.NewOption("PostgreSQL", "postgres"), huh.NewOption("MySQL", "mysql")).
					Value(&db.Type),
			))
			_ = ignoreAbort(f.Run())
		case "host":
			f := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Host").Value(&db.Host)))
			_ = ignoreAbort(f.Run())
		case "port":
			f := huh.NewForm(huh.NewGroup(
				huh.NewInput().Title("Port (leave blank for default: 5432 / 3306)").Value(&portStr),
			))
			_ = ignoreAbort(f.Run())
			if p := strings.TrimSpace(portStr); p != "" {
				if v, e := strconv.Atoi(p); e == nil {
					db.Port = v
				}
			} else {
				if db.Type == "mysql" {
					db.Port = 3306
				} else {
					db.Port = 5432
				}
			}
		case "user":
			f := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Database user").Value(&db.User)))
			_ = ignoreAbort(f.Run())
		case "database":
			f := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Database name").Value(&db.Database)))
			_ = ignoreAbort(f.Run())
		case "password_env":
			f := huh.NewForm(huh.NewGroup(
				huh.NewInput().
					Title("Password env var (e.g. DB_PASS)").
					Description("The env var that holds the database password").
					Value(&db.PasswordEnv),
			))
			_ = ignoreAbort(f.Run())
			askEnvValue(db.PasswordEnv, envVars)
		case "excluded_tables":
			current := strings.Join(db.ExcludedTables, ", ")
			f := huh.NewForm(huh.NewGroup(
				huh.NewInput().
					Title("Excluded tables").
					Description("Comma-separated table names to skip during backup (e.g. logs, sessions, cache)").
					Value(&current),
			))
			_ = ignoreAbort(f.Run())
			db.ExcludedTables = splitTrimmed(current)
		case "extra_opts":
			f := huh.NewForm(huh.NewGroup(
				huh.NewInput().
					Title("Extra dump options (optional)").
					Description("Extra flags passed to mysqldump / pg_dump").
					Value(&db.ExtraOpts),
			))
			_ = ignoreAbort(f.Run())
		case "enabled":
			f := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("Enable this database?").Value(&db.Enabled)))
			_ = ignoreAbort(f.Run())
		case "delete":
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
	}
}

// ── Storage ──────────────────────────────────────────────────────────────────

func editStorage(cfg *config.Config) error {
	r := &cfg.Storage.Rclone
	for {
		var choice string
		form := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Storage (Rclone)").
				Description("Select a field to edit · Esc to go back").
				Options(
					huh.NewOption(fmt.Sprintf("Enabled               %s", yesNo(r.Enabled)), "enabled"),
					huh.NewOption(fmt.Sprintf("Remote name           %s", preview(r.Remote)), "remote"),
					huh.NewOption(fmt.Sprintf("Remote path           %s", preview(r.Path)), "path"),
					huh.NewOption(fmt.Sprintf("Rclone config file    %s", preview(r.RcloneConfig)), "rclone_config"),
					huh.NewOption(fmt.Sprintf("Extra args            %s", preview(r.ExtraArgs)), "extra_args"),
					huh.NewOption(fmt.Sprintf("Delete after upload   %s", yesNo(r.DeleteAfterUpload)), "delete_after_upload"),
					huh.NewOption("← Back", "back"),
				).
				Value(&choice),
		))
		if err := form.Run(); err != nil || choice == "back" {
			return nil
		}

		switch choice {
		case "enabled":
			f := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("Enable rclone upload?").Value(&r.Enabled)))
			_ = ignoreAbort(f.Run())
		case "remote":
			f := huh.NewForm(huh.NewGroup(
				huh.NewInput().
					Title("Remote name").
					Description("As configured in rclone, e.g. 's3backup' or 'gdrive'").
					Value(&r.Remote),
			))
			_ = ignoreAbort(f.Run())
		case "path":
			f := huh.NewForm(huh.NewGroup(
				huh.NewInput().
					Title("Remote path template").
					Description("Placeholders: {hostname} {name} {date}").
					Value(&r.Path),
			))
			_ = ignoreAbort(f.Run())
		case "rclone_config":
			f := huh.NewForm(huh.NewGroup(
				huh.NewInput().Title("rclone config file path (leave blank for default)").Value(&r.RcloneConfig),
			))
			_ = ignoreAbort(f.Run())
		case "extra_args":
			f := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Extra rclone args (optional)").Value(&r.ExtraArgs)))
			_ = ignoreAbort(f.Run())
		case "delete_after_upload":
			f := huh.NewForm(huh.NewGroup(
				huh.NewConfirm().Title("Delete local backup after successful upload?").Value(&r.DeleteAfterUpload),
			))
			_ = ignoreAbort(f.Run())
		}
	}
}

// ── Retention ────────────────────────────────────────────────────────────────

func editRetention(cfg *config.Config) error {
	for {
		var section string
		form := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Retention").
				Options(
					huh.NewOption(fmt.Sprintf("Local   (enabled: %s, keep last: %d, days: %d)",
						yesNo(cfg.Retention.Local.Enabled), cfg.Retention.Local.KeepLast, cfg.Retention.Local.KeepDays), "local"),
					huh.NewOption(fmt.Sprintf("Remote  (enabled: %s, keep last: %d, days: %d)",
						yesNo(cfg.Retention.Remote.Enabled), cfg.Retention.Remote.KeepLast, cfg.Retention.Remote.KeepDays), "remote"),
					huh.NewOption("← Back", "back"),
				).
				Value(&section),
		))
		if err := form.Run(); err != nil || section == "back" {
			return nil
		}
		switch section {
		case "local":
			_ = editRetentionSection("Local Retention", &cfg.Retention.Local.Enabled, &cfg.Retention.Local.KeepLast, &cfg.Retention.Local.KeepDays)
		case "remote":
			_ = editRetentionSection("Remote Retention", &cfg.Retention.Remote.Enabled, &cfg.Retention.Remote.KeepLast, &cfg.Retention.Remote.KeepDays)
		}
	}
}

func editRetentionSection(title string, enabled *bool, keepLast, keepDays *int) error {
	keepLastStr := intStr(*keepLast)
	keepDaysStr := intStr(*keepDays)
	for {
		var choice string
		form := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title(title).
				Description("Select a field to edit · Esc to go back").
				Options(
					huh.NewOption(fmt.Sprintf("Enabled       %s", yesNo(*enabled)), "enabled"),
					huh.NewOption(fmt.Sprintf("Keep last     %s", keepLastStr), "keep_last"),
					huh.NewOption(fmt.Sprintf("Keep days     %s", keepDaysStr), "keep_days"),
					huh.NewOption("← Back", "back"),
				).
				Value(&choice),
		))
		if err := form.Run(); err != nil || choice == "back" {
			*keepLast = parseIntOr(keepLastStr, *keepLast)
			*keepDays = parseIntOr(keepDaysStr, *keepDays)
			return nil
		}
		switch choice {
		case "enabled":
			f := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("Enable retention?").Value(enabled)))
			_ = ignoreAbort(f.Run())
		case "keep_last":
			f := huh.NewForm(huh.NewGroup(
				huh.NewInput().Title("Keep last N backups (0 = off)").Value(&keepLastStr),
			))
			_ = ignoreAbort(f.Run())
		case "keep_days":
			f := huh.NewForm(huh.NewGroup(
				huh.NewInput().Title("Delete backups older than N days (0 = off)").Value(&keepDaysStr),
			))
			_ = ignoreAbort(f.Run())
		}
	}
}

// ── Notifications ────────────────────────────────────────────────────────────

func editNotifications(cfg *config.Config, envVars map[string]string) error {
	n := &cfg.Notifications
	for {
		var section string
		form := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Notifications").
				Options(
					huh.NewOption(fmt.Sprintf("General   (on success: %s, on failure: %s)",
						yesNo(n.OnSuccess), yesNo(n.OnFailure)), "general"),
					huh.NewOption(fmt.Sprintf("Telegram  (enabled: %s)", yesNo(n.Telegram.Enabled)), "telegram"),
					huh.NewOption(fmt.Sprintf("Email     (enabled: %s)", yesNo(n.Email.Enabled)), "email"),
					huh.NewOption("← Back", "back"),
				).
				Value(&section),
		))
		if err := form.Run(); err != nil || section == "back" {
			return nil
		}
		switch section {
		case "general":
			_ = editNotifGeneral(n)
		case "telegram":
			_ = editNotifTelegram(n, envVars)
		case "email":
			_ = editNotifEmail(n, envVars)
		}
	}
}

func editNotifGeneral(n *config.NotificationConfig) error {
	for {
		var choice string
		form := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Notifications — General").
				Options(
					huh.NewOption(fmt.Sprintf("Notify on success   %s", yesNo(n.OnSuccess)), "on_success"),
					huh.NewOption(fmt.Sprintf("Notify on failure   %s", yesNo(n.OnFailure)), "on_failure"),
					huh.NewOption("← Back", "back"),
				).
				Value(&choice),
		))
		if err := form.Run(); err != nil || choice == "back" {
			return nil
		}
		switch choice {
		case "on_success":
			f := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("Notify on success?").Value(&n.OnSuccess)))
			_ = ignoreAbort(f.Run())
		case "on_failure":
			f := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("Notify on failure?").Value(&n.OnFailure)))
			_ = ignoreAbort(f.Run())
		}
	}
}

func editNotifTelegram(n *config.NotificationConfig, envVars map[string]string) error {
	t := &n.Telegram
	for {
		var choice string
		form := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Notifications — Telegram").
				Options(
					huh.NewOption(fmt.Sprintf("Enabled         %s", yesNo(t.Enabled)), "enabled"),
					huh.NewOption(fmt.Sprintf("Bot token env   %s", preview(t.BotTokenEnv)), "bot_token_env"),
					huh.NewOption(fmt.Sprintf("Chat ID         %s", preview(t.ChatID)), "chat_id"),
					huh.NewOption("← Back", "back"),
				).
				Value(&choice),
		))
		if err := form.Run(); err != nil || choice == "back" {
			return nil
		}
		switch choice {
		case "enabled":
			f := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("Enable Telegram?").Value(&t.Enabled)))
			_ = ignoreAbort(f.Run())
		case "bot_token_env":
			f := huh.NewForm(huh.NewGroup(
				huh.NewInput().Title("Bot token env var (e.g. TELEGRAM_TOKEN)").Value(&t.BotTokenEnv),
			))
			_ = ignoreAbort(f.Run())
			askEnvValue(t.BotTokenEnv, envVars)
		case "chat_id":
			f := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Chat ID").Value(&t.ChatID)))
			_ = ignoreAbort(f.Run())
		}
	}
}

func editNotifEmail(n *config.NotificationConfig, envVars map[string]string) error {
	e := &n.Email
	toStr := strings.Join(e.To, ", ")
	smtpPortStr := intStr(e.SMTPPort)
	if smtpPortStr == "0" {
		smtpPortStr = "587"
	}

	for {
		var choice string
		form := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Notifications — Email").
				Options(
					huh.NewOption(fmt.Sprintf("Enabled       %s", yesNo(e.Enabled)), "enabled"),
					huh.NewOption(fmt.Sprintf("SMTP host     %s", preview(e.SMTPHost)), "smtp_host"),
					huh.NewOption(fmt.Sprintf("SMTP port     %s", smtpPortStr), "smtp_port"),
					huh.NewOption(fmt.Sprintf("TLS           %s", yesNo(e.SMTPTLS)), "smtp_tls"),
					huh.NewOption(fmt.Sprintf("From          %s", preview(e.From)), "from"),
					huh.NewOption(fmt.Sprintf("To            %s", preview(toStr)), "to"),
					huh.NewOption(fmt.Sprintf("Username      %s", preview(e.Username)), "username"),
					huh.NewOption(fmt.Sprintf("Password env  %s", preview(e.PasswordEnv)), "password_env"),
					huh.NewOption("← Back", "back"),
				).
				Value(&choice),
		))
		if err := form.Run(); err != nil || choice == "back" {
			e.SMTPPort = parseIntOr(smtpPortStr, e.SMTPPort)
			e.To = splitTrimmed(toStr)
			return nil
		}
		switch choice {
		case "enabled":
			f := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("Enable Email?").Value(&e.Enabled)))
			_ = ignoreAbort(f.Run())
		case "smtp_host":
			f := huh.NewForm(huh.NewGroup(huh.NewInput().Title("SMTP host").Value(&e.SMTPHost)))
			_ = ignoreAbort(f.Run())
		case "smtp_port":
			f := huh.NewForm(huh.NewGroup(huh.NewInput().Title("SMTP port (e.g. 587)").Value(&smtpPortStr)))
			_ = ignoreAbort(f.Run())
		case "smtp_tls":
			f := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("Use TLS / STARTTLS?").Value(&e.SMTPTLS)))
			_ = ignoreAbort(f.Run())
		case "from":
			f := huh.NewForm(huh.NewGroup(huh.NewInput().Title("From address").Value(&e.From)))
			_ = ignoreAbort(f.Run())
		case "to":
			f := huh.NewForm(huh.NewGroup(
				huh.NewInput().Title("To addresses (comma separated)").Value(&toStr),
			))
			_ = ignoreAbort(f.Run())
		case "username":
			f := huh.NewForm(huh.NewGroup(huh.NewInput().Title("SMTP username").Value(&e.Username)))
			_ = ignoreAbort(f.Run())
		case "password_env":
			f := huh.NewForm(huh.NewGroup(
				huh.NewInput().Title("SMTP password env var (e.g. SMTP_PASS)").Value(&e.PasswordEnv),
			))
			_ = ignoreAbort(f.Run())
			askEnvValue(e.PasswordEnv, envVars)
		}
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func askEnvValue(name string, envVars map[string]string) {
	if name == "" {
		return
	}
	existing := os.Getenv(name)
	desc := fmt.Sprintf("Value will be saved to the env file alongside your config.")
	if existing != "" {
		desc = "Already set in environment. Leave blank to keep the existing value."
	}
	val := ""
	f := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title(fmt.Sprintf("Value for %s", name)).
			Description(desc).
			EchoMode(huh.EchoModePassword).
			Value(&val),
	))
	_ = ignoreAbort(f.Run())
	if val != "" {
		envVars[name] = val
	}
}

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

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
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

func splitTrimmed(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
