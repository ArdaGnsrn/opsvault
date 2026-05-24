package backup

import (
	"fmt"
	"time"

	"github.com/ArdaGnsrn/opsvault/internal/config"
)

// BackupFilename returns the filename for a backup file.
// Format: {name}_{date}_{time}.sql.gz
func BackupFilename(db config.DatabaseConfig, t time.Time) string {
	return fmt.Sprintf("%s_%s.sql.gz", db.Name, t.UTC().Format("20060102_150405"))
}
