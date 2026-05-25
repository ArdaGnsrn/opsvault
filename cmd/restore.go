package cmd

import (
	"github.com/ArdaGnsrn/opsvault/internal/restore"
	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore a database from a backup file",
	Long: "Interactive wizard to restore a database from a local or remote backup.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		return restore.RunWizard(cmd.Context(), cfg)
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)
}
