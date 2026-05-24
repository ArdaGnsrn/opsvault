package restore

import (
	"context"

	"github.com/ArdaGnsrn/opsvault/internal/config"
)

// Restorer restores a gzip-compressed SQL dump to a database.
type Restorer interface {
	Restore(ctx context.Context, db config.DatabaseConfig, filePath string) error
}

func New(dbType string) Restorer {
	switch dbType {
	case "mysql":
		return &MySQLRestorer{}
	case "postgres":
		return &PostgresRestorer{}
	default:
		return nil
	}
}
