package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ArdaGnsrn/opsvault/internal/buildinfo"
	"github.com/ArdaGnsrn/opsvault/internal/config"
	"github.com/ArdaGnsrn/opsvault/internal/envfile"
	"github.com/ArdaGnsrn/opsvault/internal/ui"
	"github.com/ArdaGnsrn/opsvault/internal/updater"
	"github.com/fatih/color"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:          "opsvault",
	Short:        "Lightweight database backup tool with rclone storage",
	Long:         ui.Cyan.Sprint("OpsVault") + " — automated database backups with rclone storage and systemd integration.\n  " + ui.Dim.Sprint("https://github.com/ArdaGnsrn/opsvault"),
	SilenceUsage: true,
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		// Skip update notice for the version command (it shows its own)
		// and for non-TTY environments (scripts, systemd)
		if cmd.Name() == "version" {
			return
		}
		isTTY := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
		if !isTTY {
			return
		}
		latest := updater.LatestVersion()
		if updater.IsNewer(buildinfo.Version, latest) {
			fmt.Println()
			fmt.Println(ui.Warn(fmt.Sprintf("New version available: %s → run: curl -fsSL https://get.opsvault.dev | sudo bash", latest)))
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func defaultConfigPath() string {
	if os.Getuid() == 0 {
		return "/etc/opsvault/config.yaml"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "/etc/opsvault/config.yaml"
	}
	return filepath.Join(home, ".config", "opsvault", "config.yaml")
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", defaultConfigPath(), "path to config file")
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	setHelpTemplate()
}

func setHelpTemplate() {
	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	cobra.AddTemplateFunc("boldText", bold)
	cobra.AddTemplateFunc("cyanText", cyan)
	cobra.AddTemplateFunc("dimText", dim)
	cobra.AddTemplateFunc("cmdPad", func(name string, pad int) string {
		return cyan(name) + strings.Repeat(" ", pad-len(name))
	})

	rootCmd.SetHelpTemplate(`{{with .Long}}{{.}}

{{end}}` + bold("Usage:") + `
  {{.UseLine}}{{if .HasAvailableSubCommands}} [command]{{end}}
{{if .HasAvailableSubCommands}}
` + bold("Commands:") + `
{{range .Commands}}{{if .IsAvailableCommand}}  {{cmdPad .Name .NamePadding}}  {{dimText .Short}}
{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}
` + bold("Flags:") + `
{{.LocalFlags.FlagUsages | trimRightSpace}}
{{end}}{{if .HasAvailableInheritedFlags}}` + bold("Global Flags:") + `
{{.InheritedFlags.FlagUsages | trimRightSpace}}
{{end}}{{if .HasAvailableSubCommands}}{{dimText (printf "Use \"%s [command] --help\" for more information." .CommandPath)}}
{{end}}`)
}

func loadConfig() (*config.Config, error) {
	if _, err := os.Stat(cfgFile); errors.Is(err, fs.ErrNotExist) {
		if mkErr := os.MkdirAll(filepath.Dir(cfgFile), 0755); mkErr != nil {
			return nil, fmt.Errorf("creating config directory: %w", mkErr)
		}
		if mkErr := config.WriteFile(cfgFile, config.Defaults()); mkErr != nil {
			return nil, fmt.Errorf("creating default config: %w", mkErr)
		}
		fmt.Println(ui.Info("Created default config at " + cfgFile))
	}
	_ = envfile.Load(envfile.PathFor(cfgFile))
	cfg, err := config.LoadFile(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return cfg, nil
}

func getLogger(level, format string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	isTTY := isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())

	if format == "json" || !isTTY {
		return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
	}

	return slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      lvl,
		TimeFormat: "15:04:05",
	}))
}
