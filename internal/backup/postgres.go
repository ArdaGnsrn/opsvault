package backup

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/ArdaGnsrn/opsvault/internal/config"
)

type PostgresDumper struct{}

func (d *PostgresDumper) Dump(ctx context.Context, db config.DatabaseConfig, destPath string) error {
	args := []string{
		fmt.Sprintf("--host=%s", db.Host),
		fmt.Sprintf("--port=%d", db.Port),
		fmt.Sprintf("--username=%s", db.User),
		"--no-password",
		"--format=plain",
	}

	for _, tbl := range db.ExcludedTables {
		args = append(args, fmt.Sprintf("--exclude-table=%s", tbl))
	}
	if db.ExtraOpts != "" {
		args = append(args, strings.Fields(db.ExtraOpts)...)
	}
	args = append(args, db.Database)

	tmpPath := destPath + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	gz, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("creating gzip writer: %w", err)
	}

	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	// Pass password via environment variable — never via CLI args.
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", db.Password))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		gz.Close()
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("getting stdout pipe: %w", err)
	}

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		gz.Close()
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("starting pg_dump: %w", err)
	}

	if _, err := io.Copy(gz, stdout); err != nil {
		cmd.Wait()
		gz.Close()
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("copying dump output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		gz.Close()
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("pg_dump failed: %w\nstderr: %s", err, stderrBuf.String())
	}

	if err := gz.Close(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("closing gzip: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("closing file: %w", err)
	}

	return os.Rename(tmpPath, destPath)
}
