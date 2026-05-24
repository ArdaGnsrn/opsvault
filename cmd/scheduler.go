package cmd

import "github.com/spf13/cobra"

var schedulerCmd = &cobra.Command{
	Use:    "scheduler",
	Short:  "Scheduler daemon commands",
	Hidden: true,
}

func init() {
	rootCmd.AddCommand(schedulerCmd)
}
