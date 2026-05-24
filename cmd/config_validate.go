package cmd

import (
	"fmt"
	"os"

	"github.com/ArdaGnsrn/opsvault/internal/config"
	"github.com/ArdaGnsrn/opsvault/internal/ui"
	"github.com/spf13/cobra"
)

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadFile(cfgFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		errs := config.Validate(cfg)
		if len(errs) > 0 {
			fmt.Fprintln(os.Stderr, ui.Fail(ui.Bold.Sprintf("Config invalid (%d error(s))", len(errs))))
			fmt.Fprintln(os.Stderr)
			for _, e := range errs {
				fmt.Fprintln(os.Stderr, "    "+ui.Red.Sprint("• ")+e.Error())
			}
			fmt.Fprintln(os.Stderr)
			os.Exit(1)
		}

		onOff := func(b bool) string {
			if b {
				return ui.Green.Sprint("enabled")
			}
			return ui.Dim.Sprint("disabled")
		}

		fmt.Println()
		fmt.Println(ui.OK(ui.Bold.Sprint("Config valid") + "  " + ui.Dim.Sprint(cfgFile)))
		fmt.Println()
		fmt.Printf("    %-14s %s\n", ui.Dim.Sprint("databases"), ui.Bold.Sprintf("%d", len(cfg.Databases)))
		fmt.Printf("    %-14s %s\n", ui.Dim.Sprint("schedule"), ui.Cyan.Sprint(cfg.Schedule))
		fmt.Printf("    %-14s %s\n", ui.Dim.Sprint("backup dir"), cfg.BackupDir)
		fmt.Printf("    %-14s %s\n", ui.Dim.Sprint("rclone"), onOff(cfg.Storage.Rclone.Enabled))
		fmt.Printf("    %-14s %s\n", ui.Dim.Sprint("telegram"), onOff(cfg.Notifications.Telegram.Enabled))
		fmt.Printf("    %-14s %s\n", ui.Dim.Sprint("email"), onOff(cfg.Notifications.Email.Enabled))
		fmt.Println()
		return nil
	},
}

func init() {
	configCmd.AddCommand(configValidateCmd)
}
