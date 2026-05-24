package cmd

import (
	"github.com/ArdaGnsrn/opsvault/internal/service"
	"github.com/spf13/cobra"
)

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the opsvault service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return service.Passthrough("start", "opsvault.service")
	},
}

func init() {
	serviceCmd.AddCommand(serviceStartCmd)
}
