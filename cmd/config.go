package cmd

import (
	"github.com/ArdaGnsrn/opsvault/internal/wizard"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Edit configuration interactively",
	Long:  "Launch the interactive TUI wizard to create or edit your config file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return wizard.Run(cfgFile)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
