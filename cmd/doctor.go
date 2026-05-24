package cmd

import (
	"fmt"
	"os/exec"

	"github.com/ArdaGnsrn/opsvault/internal/ui"
	"github.com/spf13/cobra"
)

type tool struct {
	name    string
	binary  string
	install string
}

var tools = []tool{
	{
		name:    "pg_dump",
		binary:  "pg_dump",
		install: "apt-get install -y postgresql-client",
	},
	{
		name:    "mysqldump",
		binary:  "mysqldump",
		install: "apt-get install -y default-mysql-client",
	},
	{
		name:    "rclone",
		binary:  "rclone",
		install: "curl https://rclone.org/install.sh | sudo bash",
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check that required external tools are installed",
	Run: func(cmd *cobra.Command, args []string) {
		ui.Bold.Println("Checking dependencies...\n")

		allOK := true
		for _, t := range tools {
			path, err := exec.LookPath(t.binary)
			if err != nil {
				fmt.Println(ui.Fail(ui.Bold.Sprint(t.binary)))
				fmt.Printf("       %s %s\n\n", ui.Dim.Sprint("install:"), t.install)
				allOK = false
			} else {
				fmt.Println(ui.OK(ui.Bold.Sprint(t.binary) + "  " + ui.Dim.Sprint(path)))
			}
		}

		fmt.Println()
		if allOK {
			ui.Green.Println("  All tools found. OpsVault is ready.")
		} else {
			ui.Yellow.Println("  Some tools are missing. Run the install commands above.")
		}
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
