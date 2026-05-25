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
	if err := r.resetSchema(ctx, db); err != nil {
		return err
	}

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

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("psql restore failed: %w\n%s", err, out)
	}
	return nil
}

// resetSchema drops and recreates the public schema inside the target database,
// giving psql a clean slate without touching the database itself or its connections.
func (r *PostgresRestorer) resetSchema(ctx context.Context, db config.DatabaseConfig) error {
	sql := `DROP SCHEMA public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO public;`
	args := []string{
		fmt.Sprintf("--host=%s", db.Host),
		fmt.Sprintf("--port=%d", db.Port),
		fmt.Sprintf("--username=%s", db.User),
		"--no-password",
		"--command=" + sql,
		db.Database,
	}
	cmd := exec.CommandContext(ctx, "psql", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", db.Password))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("resetting schema: %w\n%s", err, out)
	}
	return nil
}
