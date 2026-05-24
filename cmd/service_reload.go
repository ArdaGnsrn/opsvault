package cmd

import (
	"fmt"
	"os"

	"github.com/ArdaGnsrn/opsvault/internal/config"
	"github.com/ArdaGnsrn/opsvault/internal/service"
	"github.com/ArdaGnsrn/opsvault/internal/ui"
	"github.com/spf13/cobra"
)

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Validate config and restart the service",
	Long:  "Validates the config file, then restarts the opsvault systemd service if valid.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadFile(cfgFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		errs := config.Validate(cfg)
		if len(errs) > 0 {
			fmt.Fprintln(os.Stderr, ui.Fail(ui.Bold.Sprintf("Config invalid (%d error(s)) — service not restarted", len(errs))))
			fmt.Fprintln(os.Stderr)
			for _, e := range errs {
				fmt.Fprintln(os.Stderr, "    "+ui.Red.Sprint("• ")+e.Error())
			}
			fmt.Fprintln(os.Stderr)
			os.Exit(1)
		}

		fmt.Println(ui.OK("Config valid"))

		fmt.Println(ui.Info("Restarting service..."))
		if err := service.Passthrough("restart", "opsvault.service"); err != nil {
			return fmt.Errorf("restart failed: %w", err)
		}

		fmt.Println(ui.OK("Service restarted"))
		fmt.Println()
		return service.Passthrough("status", "opsvault.service")
	},
}

func init() {
	rootCmd.AddCommand(reloadCmd)
}
