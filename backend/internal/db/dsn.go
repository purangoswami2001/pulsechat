package db

import "strings"

// FormatDSN wraps or adjusts configuration DSNs if needed.
func FormatDSN(driver, dsn string) string {
	if driver == "postgres" && !strings.HasPrefix(dsn, "postgres://") && !strings.HasPrefix(dsn, "postgresql://") {
		return "postgres://" + dsn
	}
	return dsn
}
