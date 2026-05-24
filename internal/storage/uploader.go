package storage

import "context"

// Uploader copies a local file to remote storage.
// vars contains template variables for the remote path (hostname, name, date).
type Uploader interface {
	Upload(ctx context.Context, localPath string, vars map[string]string) error
}
