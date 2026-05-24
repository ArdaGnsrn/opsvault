package restore

import (
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/ArdaGnsrn/opsvault/internal/config"
)

type MySQLRestorer struct{}

func (r *MySQLRestorer) Restore(ctx context.Context, db config.DatabaseConfig, filePath string) error {
	tmpCnf, err := writeMyCnf(db.User, db.Password)
	if err != nil {
		return fmt.Errorf("creating .my.cnf: %w", err)
	}
	defer os.Remove(tmpCnf)

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening backup file: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("opening gzip: %w", err)
	}
	defer gz.Close()

	args := []string{
		fmt.Sprintf("--defaults-extra-file=%s", tmpCnf),
		fmt.Sprintf("--host=%s", db.Host),
		fmt.Sprintf("--port=%d", db.Port),
		db.Database,
	}

	cmd := exec.CommandContext(ctx, "mysql", args...)
	cmd.Stdin = gz
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysql restore failed: %w", err)
	}
	return nil
}

func writeMyCnf(user, password string) (string, error) {
	f, err := os.CreateTemp("", "opsvault-mycnf-*.cnf")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := os.Chmod(f.Name(), 0600); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	_, err = fmt.Fprintf(f, "[client]\nuser=%s\npassword=%s\n", user, password)
	if err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}
