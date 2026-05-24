package cmd

import (
	"github.com/ArdaGnsrn/opsvault/internal/service"
	"github.com/spf13/cobra"
)

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Disable and remove the opsvault systemd service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return service.Uninstall()
	},
}

func init() {
	serviceCmd.AddCommand(serviceUninstallCmd)
}
