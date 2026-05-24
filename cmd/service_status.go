package cmd

import (
	"github.com/ArdaGnsrn/opsvault/internal/service"
	"github.com/spf13/cobra"
)

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of the opsvault service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return service.Passthrough("status", "opsvault.service")
	},
}

func init() {
	serviceCmd.AddCommand(serviceStatusCmd)
}
