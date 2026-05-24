package cmd

import "github.com/spf13/cobra"

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Database backup commands",
}

func init() {
	rootCmd.AddCommand(backupCmd)
}
