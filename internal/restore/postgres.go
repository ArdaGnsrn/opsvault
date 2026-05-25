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
	if err := r.dropAndRecreate(ctx, db); err != nil {
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

// dropAndRecreate terminates active connections, drops the target database,
// and recreates it — giving psql a clean slate to restore into.
func (r *PostgresRestorer) dropAndRecreate(ctx context.Context, db config.DatabaseConfig) error {
	env := append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", db.Password))
	connArgs := []string{
		fmt.Sprintf("--host=%s", db.Host),
		fmt.Sprintf("--port=%d", db.Port),
		fmt.Sprintf("--username=%s", db.User),
		"--no-password",
		"--dbname=postgres",
	}

	steps := []string{
		fmt.Sprintf(
			`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()`,
			db.Database,
		),
		fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, db.Database),
		fmt.Sprintf(`CREATE DATABASE "%s"`, db.Database),
	}

	for _, sql := range steps {
		args := append(connArgs, "--command="+sql)
		cmd := exec.CommandContext(ctx, "psql", args...)
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("preparing database: %w\n%s", err, out)
		}
	}
	return nil
}
