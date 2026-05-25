package cmd

import (
	"github.com/ArdaGnsrn/opsvault/internal/service"
	"github.com/spf13/cobra"
)

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install opsvault as a systemd service",
	RunE: func(cmd *cobra.Command, args []string) error {
		binaryPath, _ := cmd.Flags().GetString("binary")

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		return service.Install(cfgFile, cfg.BackupDir, binaryPath)
	},
}

func init() {
	serviceInstallCmd.Flags().String("binary", "", "path to the opsvault binary (default: /usr/local/bin/opsvault)")
	serviceCmd.AddCommand(serviceInstallCmd)
}
