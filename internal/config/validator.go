package config

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func Validate(cfg *Config) []ValidationError {
	var errs []ValidationError

	if len(cfg.Databases) == 0 {
		errs = append(errs, ValidationError{Field: "databases", Message: "at least one database must be configured"})
	}

	for i, db := range cfg.Databases {
		prefix := fmt.Sprintf("databases[%d]", i)
		if db.Name == "" {
			errs = append(errs, ValidationError{Field: prefix + ".name", Message: "required"})
		}
		if db.Type != "mysql" && db.Type != "postgres" {
			errs = append(errs, ValidationError{Field: prefix + ".type", Message: "must be 'mysql' or 'postgres'"})
		}
		if db.User == "" {
			errs = append(errs, ValidationError{Field: prefix + ".user", Message: "required"})
		}
		if db.Database == "" {
			errs = append(errs, ValidationError{Field: prefix + ".database", Message: "required"})
		}
		if db.Password == "" && db.PasswordEnv == "" {
			errs = append(errs, ValidationError{Field: prefix + ".password", Message: "password or password_env required"})
		}
	}

	if cfg.Storage.Rclone.Enabled {
		if cfg.Storage.Rclone.Remote == "" {
			errs = append(errs, ValidationError{Field: "storage.rclone.remote", Message: "required when rclone is enabled"})
		}
	}

	if cfg.Notifications.Telegram.Enabled {
		if cfg.Notifications.Telegram.BotToken == "" {
			errs = append(errs, ValidationError{Field: "notifications.telegram.bot_token", Message: "required when telegram is enabled (or set bot_token_env)"})
		}
		if cfg.Notifications.Telegram.ChatID == "" {
			errs = append(errs, ValidationError{Field: "notifications.telegram.chat_id", Message: "required when telegram is enabled"})
		}
	}

	if cfg.Notifications.Email.Enabled {
		if cfg.Notifications.Email.SMTPHost == "" {
			errs = append(errs, ValidationError{Field: "notifications.email.smtp_host", Message: "required when email is enabled"})
		}
		if cfg.Notifications.Email.From == "" {
			errs = append(errs, ValidationError{Field: "notifications.email.from", Message: "required when email is enabled"})
		}
		if len(cfg.Notifications.Email.To) == 0 {
			errs = append(errs, ValidationError{Field: "notifications.email.to", Message: "at least one recipient required when email is enabled"})
		}
	}

	if cfg.Schedule != "" {
		if err := validateCron(cfg.Schedule); err != nil {
			errs = append(errs, ValidationError{Field: "schedule", Message: err.Error()})
		}
	}

	return errs
}

func validateCron(expr string) error {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return fmt.Errorf("must be a 5-field cron expression (minute hour dom month dow)")
	}
	return nil
}
