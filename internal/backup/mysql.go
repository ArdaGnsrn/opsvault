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

type MySQLDumper struct{}

func (d *MySQLDumper) Dump(ctx context.Context, db config.DatabaseConfig, destPath string) error {
	// Write a temporary .my.cnf to avoid passing password on the command line.
	tmpCnf, err := writeMyCnf(db.User, db.Password)
	if err != nil {
		return fmt.Errorf("creating .my.cnf: %w", err)
	}
	defer os.Remove(tmpCnf)

	args := []string{
		fmt.Sprintf("--defaults-extra-file=%s", tmpCnf),
		fmt.Sprintf("--host=%s", db.Host),
		fmt.Sprintf("--port=%d", db.Port),
		"--single-transaction",
		"--routines",
		"--triggers",
	}

	for _, tbl := range db.ExcludedTables {
		args = append(args, fmt.Sprintf("--ignore-table=%s.%s", db.Database, tbl))
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

	cmd := exec.CommandContext(ctx, "mysqldump", args...)
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
		return fmt.Errorf("starting mysqldump: %w", err)
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
		return fmt.Errorf("mysqldump failed: %w\nstderr: %s", err, stderrBuf.String())
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
