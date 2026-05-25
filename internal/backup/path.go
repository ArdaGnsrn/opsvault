package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ArdaGnsrn/opsvault/internal/config"
)

type PathArchiver struct{}

func (a *PathArchiver) Archive(ctx context.Context, cfg config.PathConfig, destPath string) error {
	src := filepath.Clean(cfg.Path)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("source path: %w", err)
	}

	patterns := ResolveExcludes(cfg)

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

	tw := tar.NewWriter(gz)

	walkErr := filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		if shouldExcludePath(rel, patterns) {
			if fi.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip non-regular files and non-directories (symlinks, devices, etc.)
		if !fi.Mode().IsRegular() && !fi.IsDir() {
			return nil
		}

		hdr, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return fmt.Errorf("creating tar header for %s: %w", rel, err)
		}
		// Use forward slashes and relative path in archive
		hdr.Name = filepath.ToSlash(rel)
		if fi.IsDir() {
			hdr.Name += "/"
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("writing tar header: %w", err)
		}

		if fi.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening %s: %w", rel, err)
		}
		defer file.Close()

		if _, err := io.Copy(tw, file); err != nil {
			return fmt.Errorf("writing %s: %w", rel, err)
		}
		return nil
	})

	if walkErr != nil {
		tw.Close()
		gz.Close()
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("archiving: %w", walkErr)
	}

	if err := tw.Close(); err != nil {
		gz.Close()
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("closing tar: %w", err)
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

// shouldExcludePath returns true if relPath matches any exclusion pattern.
// Patterns without "/" are matched against the base name only (any depth).
// Patterns with "/" are matched against the full relative path.
func shouldExcludePath(relPath string, patterns []string) bool {
	base := filepath.Base(relPath)
	normalized := filepath.ToSlash(relPath)
	for _, pat := range patterns {
		if strings.Contains(pat, "/") {
			matched, _ := filepath.Match(filepath.FromSlash(pat), filepath.FromSlash(normalized))
			if matched {
				return true
			}
		} else {
			matched, _ := filepath.Match(pat, base)
			if matched {
				return true
			}
		}
	}
	return false
}
