package storage

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ArdaGnsrn/opsvault/internal/config"
)

// RcloneUploader uploads files using the rclone binary.
type RcloneUploader struct {
	cfg config.RcloneConfig
}

func NewRcloneUploader(cfg config.RcloneConfig) *RcloneUploader {
	return &RcloneUploader{cfg: cfg}
}

func (r *RcloneUploader) Upload(ctx context.Context, localPath string, vars map[string]string) error {
	remotePath := ExpandPathTemplate(r.cfg.Path, vars)
	dest := fmt.Sprintf("%s:%s/%s", r.cfg.Remote, remotePath, filepath.Base(localPath))

	args := []string{"copy"}

	if r.cfg.RcloneConfig != "" {
		args = append(args, "--config", r.cfg.RcloneConfig)
	}

	if r.cfg.ExtraArgs != "" {
		args = append(args, strings.Fields(r.cfg.ExtraArgs)...)
	}

	args = append(args, localPath, filepath.Dir(dest))

	// rclone copy takes a source and destination DIRECTORY, not a destination FILE.
	// Rewrite: `rclone copy <localPath> <remote>:<remotePath>/`
	args = []string{"copy"}
	if r.cfg.RcloneConfig != "" {
		args = append(args, "--config", r.cfg.RcloneConfig)
	}
	if r.cfg.ExtraArgs != "" {
		args = append(args, strings.Fields(r.cfg.ExtraArgs)...)
	}
	args = append(args, localPath, fmt.Sprintf("%s:%s", r.cfg.Remote, remotePath))

	cmd := exec.CommandContext(ctx, "rclone", args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rclone copy failed: %w\noutput: %s", err, string(out))
	}
	return nil
}
