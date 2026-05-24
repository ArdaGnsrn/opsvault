package cmd

import (
	"fmt"

	"github.com/ArdaGnsrn/opsvault/internal/buildinfo"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("opsvault %s (commit: %s, built: %s)\n",
			buildinfo.Version, buildinfo.Commit, buildinfo.BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
