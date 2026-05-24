package cmd

import (
	"fmt"

	"github.com/ArdaGnsrn/opsvault/internal/config"
	"github.com/ArdaGnsrn/opsvault/internal/service"
	"github.com/spf13/cobra"
)

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install opsvault as a systemd service",
	RunE: func(cmd *cobra.Command, args []string) error {
		binaryPath, _ := cmd.Flags().GetString("binary")

		cfg, err := config.LoadFile(cfgFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		return service.Install(cfgFile, cfg.BackupDir, binaryPath)
	},
}

func init() {
	serviceInstallCmd.Flags().String("binary", "", "path to the opsvault binary (default: /usr/local/bin/opsvault)")
	serviceCmd.AddCommand(serviceInstallCmd)
}
