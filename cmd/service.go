package cmd

import "github.com/spf13/cobra"

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage the opsvault systemd service",
}

func init() {
	rootCmd.AddCommand(serviceCmd)
}
