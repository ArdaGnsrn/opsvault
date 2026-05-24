package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ArdaGnsrn/opsvault/assets"
	"github.com/spf13/cobra"
)

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Write a default config file",
	Long:  "Write a commented default configuration to the config file path (default: /etc/opsvault/config.yaml).",
	RunE: func(cmd *cobra.Command, args []string) error {
		dest := cfgFile

		if _, err := os.Stat(dest); err == nil {
			force, _ := cmd.Flags().GetBool("force")
			if !force {
				return fmt.Errorf("config file already exists at %s (use --force to overwrite)", dest)
			}
		}

		dir := filepath.Dir(dest)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating config directory %s: %w", dir, err)
		}

		if err := os.WriteFile(dest, assets.DefaultConfig, 0640); err != nil {
			return fmt.Errorf("writing config file: %w", err)
		}

		fmt.Printf("Config written to %s\n", dest)
		fmt.Println("Edit the file to configure your databases, storage, and notifications.")
		return nil
	},
}

func init() {
	configInitCmd.Flags().Bool("force", false, "overwrite existing config file")
	configCmd.AddCommand(configInitCmd)
}
