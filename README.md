# OpsVault

[![GitHub release](https://img.shields.io/github/v/release/ArdaGnsrn/opsvault)](https://github.com/ArdaGnsrn/opsvault/releases)
[![Downloads](https://img.shields.io/github/downloads/ArdaGnsrn/opsvault/total)](https://github.com/ArdaGnsrn/opsvault/releases)
[![Stars](https://img.shields.io/github/stars/ArdaGnsrn/opsvault)](https://github.com/ArdaGnsrn/opsvault/stargazers)
[![Go version](https://img.shields.io/github/go-mod/go-version/ArdaGnsrn/opsvault)](go.mod)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

Lightweight backup and DevOps tool for Linux servers. Backs up MySQL/PostgreSQL databases and any directory or file path, compresses and uploads via rclone, runs as a systemd service, and sends Telegram or email notifications — all from a single YAML file.

## ☕ Buy me a coffee

Whether you use this project, have learned something from it, or just like it, please consider supporting it by buying me a coffee, so I can dedicate more time on open-source projects like this :)

<a href="https://www.buymeacoffee.com/ardagnsrn" target="_blank"><img src="https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png" alt="Buy Me A Coffee" style="height: auto !important;width: auto !important;" ></a>

## Install

```bash
curl -fsSL https://get.opsvault.dev | sudo bash
```

The installer downloads the binary, optionally installs `pg_dump`, `mysqldump`, and `rclone`, creates the config, and sets up the systemd service.

**Manual install:**

```bash
curl -L https://github.com/ArdaGnsrn/opsvault/releases/latest/download/opsvault-linux-amd64 \
  -o /usr/local/bin/opsvault
chmod +x /usr/local/bin/opsvault
```

## Quick start

```bash
opsvault config             # interactive TUI to create or edit the config
opsvault backup run         # test a backup manually
opsvault service install    # install as systemd service
systemctl start opsvault
```

## Configuration

```yaml
version: 1
backup_dir: /var/backups/opsvault
schedule: "0 2 * * *"   # every night at 02:00

databases:
  - name: myapp
    type: postgres       # postgres | mysql
    host: 127.0.0.1
    port: 5432
    user: backup_user
    password_env: DB_PASS
    database: myapp_prod
    excluded_tables:     # tables to skip (optional)
      - logs
      - sessions
    enabled: true

paths:
  - name: app_uploads
    path: /var/www/myapp/uploads
    preset_excludes:     # node_modules, git, build, logs, temp, cache...
      - cache
    enabled: true

storage:
  rclone:
    enabled: true
    remote: "s3backup"
    path: "opsvault/{hostname}/{name}/{date}"
    delete_after_upload: false

retention:
  local:
    enabled: true
    keep_last: 7
    keep_days: 30
  remote:
    enabled: true
    keep_days: 90

notifications:
  on_success: true
  on_failure: true
  telegram:
    enabled: true
    bot_token_env: TELEGRAM_TOKEN
    chat_id: "123456789"
```

Full config reference: [opsvault.dev/docs/configuration](https://opsvault.dev/docs/configuration)

## Commands

| Command | Description |
|---|---|
| `opsvault backup run` | Run all enabled backups immediately |
| `opsvault backup run myapp` | Run a single database backup |
| `opsvault backup list` | List local backup files |
| `opsvault backup history` | Show backup history (success/failure log) |
| `opsvault backup history --db myapp --limit 20` | Filter history by database |
| `opsvault restore` | Interactive wizard to restore from a local or remote backup |
| `opsvault restore run --name myapp --file backup.sql.gz` | Restore non-interactively (scripting) |
| `opsvault config` | Interactive terminal wizard to create or edit the config |
| `opsvault service install` | Install and enable the systemd service |
| `opsvault service uninstall` | Disable and remove the systemd service |
| `opsvault service start` | Start the service |
| `opsvault service stop` | Stop the service |
| `opsvault service status` | Show service status |
| `opsvault service logs` | Tail service logs |
| `opsvault reload` | Validate config and restart the service |
| `opsvault doctor` | Check that required tools are installed |
| `opsvault version` | Print version and build info |

Global flag: `--config` (default: `/etc/opsvault/config.yaml` for root, `~/.config/opsvault/config.yaml` for non-root)

## How it works

```
systemd → opsvault scheduler run
              │
              ▼  (on cron schedule)
        backup.RunAll()
              │
              ├─ mysqldump / pg_dump  →  gzip  →  .sql.gz
              ├─ tar + gzip (paths)   →           .tar.gz
              ├─ rclone copy          →  remote storage
              ├─ retention cleanup    →  local + remote
              └─ Telegram / email notification
```

Passwords are never exposed on the command line. MySQL uses a temporary `~/.my.cnf` (mode 0600); PostgreSQL uses `PGPASSWORD` set only on the subprocess environment.

## Build from source

Requires Go 1.21+.

```bash
git clone https://github.com/ArdaGnsrn/opsvault
cd opsvault
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o opsvault .
```

Or with the PowerShell build script (sets version from git tag):

```powershell
.\build.ps1          # linux amd64
.\build.ps1 linux-arm64
```

Output goes to `dist/`.

## Requirements

- Linux (x86_64 or arm64)
- Root or sudo for service installation
- `mysqldump` — if backing up MySQL
- `pg_dump` — if backing up PostgreSQL
- `rclone` — if uploading to remote storage

Run `opsvault doctor` to check which tools are present.

## License

Apache 2.0 — see [LICENSE](LICENSE) for details.

See [NOTICE](NOTICE) for copyright and attribution information.
