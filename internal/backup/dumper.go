package backup

import (
	"context"

	"github.com/ArdaGnsrn/opsvault/internal/config"
)

// Dumper writes a database dump to destPath (a .sql.gz file path).
type Dumper interface {
	Dump(ctx context.Context, db config.DatabaseConfig, destPath string) error
}
