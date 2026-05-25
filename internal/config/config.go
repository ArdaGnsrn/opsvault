package config

// Config is the root configuration structure.
type Config struct {
	Version   int              `yaml:"version"`
	BackupDir string           `yaml:"backup_dir"`
	Schedule  string           `yaml:"schedule"`
	LogLevel  string           `yaml:"log_level"`
	LogFormat string           `yaml:"log_format"`
	Databases []DatabaseConfig `yaml:"databases"`
	Storage   StorageConfig    `yaml:"storage"`
	Retention RetentionConfig  `yaml:"retention"`
	Notifications NotificationConfig `yaml:"notifications"`
}

type DatabaseConfig struct {
	Name           string   `yaml:"name"`
	Type           string   `yaml:"type"` // mysql | postgres
	Host           string   `yaml:"host"`
	Port           int      `yaml:"port"`
	User           string   `yaml:"user"`
	Password       string   `yaml:"password"`
	PasswordEnv    string   `yaml:"password_env"`
	Database       string   `yaml:"database"`
	ExcludedTables []string `yaml:"excluded_tables,omitempty"`
	ExtraOpts      string   `yaml:"extra_opts"`
	Enabled        bool     `yaml:"enabled"`
}

type StorageConfig struct {
	Rclone RcloneConfig `yaml:"rclone"`
}

type RcloneConfig struct {
	Enabled           bool   `yaml:"enabled"`
	Remote            string `yaml:"remote"`
	Path              string `yaml:"path"`
	RcloneConfig      string `yaml:"rclone_config"`
	ExtraArgs         string `yaml:"extra_args"`
	DeleteAfterUpload bool   `yaml:"delete_after_upload"`
}

type RetentionConfig struct {
	Local  LocalRetentionConfig  `yaml:"local"`
	Remote RemoteRetentionConfig `yaml:"remote"`
}

type LocalRetentionConfig struct {
	Enabled  bool `yaml:"enabled"`
	KeepLast int  `yaml:"keep_last"` // keep N most recent files (0 = disabled)
	KeepDays int  `yaml:"keep_days"` // delete files older than N days (0 = disabled)
}

type RemoteRetentionConfig struct {
	Enabled  bool `yaml:"enabled"`
	KeepLast int  `yaml:"keep_last"` // keep N most recent files (0 = disabled)
	KeepDays int  `yaml:"keep_days"` // delete files older than N days (0 = disabled)
}

type NotificationConfig struct {
	OnSuccess bool           `yaml:"on_success"`
	OnFailure bool           `yaml:"on_failure"`
	Telegram  TelegramConfig `yaml:"telegram"`
	Email     EmailConfig    `yaml:"email"`
}

type TelegramConfig struct {
	Enabled      bool   `yaml:"enabled"`
	BotToken     string `yaml:"bot_token"`
	BotTokenEnv  string `yaml:"bot_token_env"`
	ChatID       string `yaml:"chat_id"`
}

type EmailConfig struct {
	Enabled     bool     `yaml:"enabled"`
	SMTPHost    string   `yaml:"smtp_host"`
	SMTPPort    int      `yaml:"smtp_port"`
	SMTPTLS     bool     `yaml:"smtp_tls"`
	From        string   `yaml:"from"`
	To          []string `yaml:"to"`
	Username    string   `yaml:"username"`
	PasswordEnv string   `yaml:"password_env"`
}
