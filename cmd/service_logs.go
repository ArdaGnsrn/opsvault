package cmd

import (
	"github.com/ArdaGnsrn/opsvault/internal/service"
	"github.com/spf13/cobra"
)

var serviceLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Stream opsvault service logs via journalctl",
	RunE: func(cmd *cobra.Command, args []string) error {
		follow, _ := cmd.Flags().GetBool("follow")
		if follow {
			return service.JournalPassthrough("-f")
		}
		return service.JournalPassthrough()
	},
}

func init() {
	serviceLogsCmd.Flags().BoolP("follow", "f", false, "follow log output")
	serviceCmd.AddCommand(serviceLogsCmd)
}
