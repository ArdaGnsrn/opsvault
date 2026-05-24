package cmd

import (
	"github.com/ArdaGnsrn/opsvault/internal/service"
	"github.com/spf13/cobra"
)

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the opsvault service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return service.Passthrough("stop", "opsvault.service")
	},
}

func init() {
	serviceCmd.AddCommand(serviceStopCmd)
}
