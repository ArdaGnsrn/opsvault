package cmd

import (
	"fmt"

	"github.com/ArdaGnsrn/opsvault/internal/buildinfo"
	"github.com/ArdaGnsrn/opsvault/internal/ui"
	"github.com/ArdaGnsrn/opsvault/internal/updater"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("opsvault %s (commit: %s, built: %s)\n",
			buildinfo.Version, buildinfo.Commit, buildinfo.BuildDate)
		fmt.Println(ui.Dim.Sprint("  https://github.com/ArdaGnsrn/opsvault"))

		fmt.Print(ui.Info("Checking for updates...") + "\r")
		latest := updater.LatestVersion()
		// clear the checking line
		fmt.Print("                              \r")

		if updater.IsNewer(buildinfo.Version, latest) {
			fmt.Println(ui.Warn(fmt.Sprintf("New version available: %s (you have %s)", latest, buildinfo.Version)))
			fmt.Println(ui.Info("Update: curl -fsSL https://get.opsvault.dev | sudo bash"))
		} else if latest != "" {
			fmt.Println(ui.OK("You are on the latest version."))
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
