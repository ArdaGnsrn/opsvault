package service

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"text/template"

	"github.com/ArdaGnsrn/opsvault/internal/envfile"
)

//go:embed templates/opsvault.service.tmpl
var templatesFS embed.FS

const (
	unitFile   = "/etc/systemd/system/opsvault.service"
	unitName   = "opsvault.service"
	defaultBin = "/usr/local/bin/opsvault"
)

type unitData struct {
	BinaryPath  string
	ConfigPath  string
	BackupDir   string
	EnvFilePath string
}

// Install writes the systemd unit file, reloads the daemon, and enables the service.
func Install(configPath, backupDir, binaryPath string) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("service install requires root privileges (run with sudo)")
	}

	if binaryPath == "" {
		binaryPath = defaultBin
	}

	data := unitData{
		BinaryPath:  binaryPath,
		ConfigPath:  configPath,
		BackupDir:   backupDir,
		EnvFilePath: envfile.PathFor(configPath),
	}

	tmplBytes, err := templatesFS.ReadFile("templates/opsvault.service.tmpl")
	if err != nil {
		return fmt.Errorf("reading service template: %w", err)
	}

	tmpl, err := template.New("service").Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("parsing service template: %w", err)
	}

	f, err := os.OpenFile(unitFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("writing unit file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("rendering unit file: %w", err)
	}

	if err := runSystemctl("daemon-reload"); err != nil {
		return err
	}
	if err := runSystemctl("enable", unitName); err != nil {
		return err
	}

	fmt.Printf("Service installed: %s\n", unitFile)
	fmt.Println("Run: systemctl start opsvault")
	return nil
}

// Uninstall disables and removes the systemd unit file.
func Uninstall() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("service uninstall requires root privileges (run with sudo)")
	}

	_ = runSystemctl("stop", unitName)
	_ = runSystemctl("disable", unitName)

	if err := os.Remove(unitFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing unit file: %w", err)
	}

	_ = runSystemctl("daemon-reload")

	fmt.Println("Service uninstalled.")
	return nil
}

// Passthrough runs a systemctl command with its output forwarded to stdout/stderr.
func Passthrough(args ...string) error {
	cmd := exec.Command("systemctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// JournalPassthrough runs journalctl -u opsvault with forwarded output.
func JournalPassthrough(args ...string) error {
	allArgs := append([]string{"-u", unitName}, args...)
	cmd := exec.Command("journalctl", allArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runSystemctl(args ...string) error {
	out, err := exec.Command("systemctl", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %v: %w\noutput: %s", args, err, string(out))
	}
	return nil
}
