package restore

import (
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/ArdaGnsrn/opsvault/internal/config"
)

type PostgresRestorer struct{}

func (r *PostgresRestorer) Restore(ctx context.Context, db config.DatabaseConfig, filePath string) error {
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
		fmt.Sprintf("--host=%s", db.Host),
		fmt.Sprintf("--port=%d", db.Port),
		fmt.Sprintf("--username=%s", db.User),
		"--no-password",
		db.Database,
	}

	cmd := exec.CommandContext(ctx, "psql", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", db.Password))
	cmd.Stdin = gz
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql restore failed: %w", err)
	}
	return nil
}
