package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	applyDefaults(&cfg)
	applyEnvOverrides(&cfg)

	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.BackupDir == "" {
		cfg.BackupDir = "/var/backups/opsvault"
	}
	if cfg.Schedule == "" {
		cfg.Schedule = "0 2 * * *"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	if cfg.LogFormat == "" {
		cfg.LogFormat = "json"
	}
	if cfg.Retention.Local.KeepLast == 0 {
		cfg.Retention.Local.KeepLast = 7
	}
	if cfg.Retention.Remote.KeepLast == 0 {
		cfg.Retention.Remote.KeepLast = 30
	}
	if cfg.Storage.Rclone.Path == "" {
		cfg.Storage.Rclone.Path = "opsvault/{hostname}/{name}/{date}"
	}
	for i := range cfg.Databases {
		if cfg.Databases[i].Type == "mysql" && cfg.Databases[i].Port == 0 {
			cfg.Databases[i].Port = 3306
		}
		if cfg.Databases[i].Type == "postgres" && cfg.Databases[i].Port == 0 {
			cfg.Databases[i].Port = 5432
		}
		if cfg.Databases[i].Host == "" {
			cfg.Databases[i].Host = "127.0.0.1"
		}
	}
}

// LoadRaw loads config without applying env overrides (used by the wizard).
func LoadRaw(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	applyDefaults(&cfg)
	return &cfg, nil
}

// Defaults returns a new Config populated with default values.
func Defaults() *Config {
	cfg := &Config{Version: 1}
	applyDefaults(cfg)
	return cfg
}

// WriteFile marshals cfg to YAML and writes it to path (mode 0640).
func WriteFile(path string, cfg *Config) error {
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(path, data, 0640)
}

func applyEnvOverrides(cfg *Config) {
	for i := range cfg.Databases {
		if cfg.Databases[i].PasswordEnv != "" {
			if val := os.Getenv(cfg.Databases[i].PasswordEnv); val != "" {
				cfg.Databases[i].Password = val
			}
		}
	}
	if cfg.Notifications.Telegram.BotTokenEnv != "" {
		if val := os.Getenv(cfg.Notifications.Telegram.BotTokenEnv); val != "" {
			cfg.Notifications.Telegram.BotToken = val
		}
	}
	if cfg.Notifications.Email.PasswordEnv != "" {
		if val := os.Getenv(cfg.Notifications.Email.PasswordEnv); val != "" {
			cfg.Notifications.Email.Username = val
		}
	}
}
