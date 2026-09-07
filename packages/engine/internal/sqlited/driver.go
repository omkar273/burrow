// Package sqlited registers a cgo-free SQLite driver under the name
// "sqlite3".
//
// ent's SQLite dialect expects a driver registered as "sqlite3", which is
// the name mattn/go-sqlite3 uses. modernc.org/sqlite registers as
// "sqlite". We want modernc because it is pure Go: a cgo driver would
// forfeit cross-compiling a single binary to a NAS.
//
// No Open wrapper is needed to force PRAGMA foreign_keys — the DSN
// carries it (see internal/repository/ent.FileDSN).
package sqlited

import (
	"database/sql"

	sqlite "modernc.org/sqlite"
)

const DriverName = "sqlite3"

func init() {
	sql.Register(DriverName, &sqlite.Driver{})
}
