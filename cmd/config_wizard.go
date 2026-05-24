package cmd

import (
	"github.com/ArdaGnsrn/opsvault/internal/wizard"
	"github.com/spf13/cobra"
)

var configWizardCmd = &cobra.Command{
	Use:   "wizard",
	Short: "Interactive config wizard",
	Long:  "Launch a terminal wizard to create or edit your config file interactively.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return wizard.Run(cfgFile)
	},
}

func init() {
	configCmd.AddCommand(configWizardCmd)
}
