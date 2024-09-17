//go:build !sqlite_math_functions

package test

import (
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/nbd-wtf/go-nostr/sdk/hints/sqlite"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/require"
)

func TestSQLiteHintsNcruces(t *testing.T) {
	path := "/tmp/tmpsdkhintssqlite"
	os.RemoveAll(path)

	db, err := sqlx.Connect("sqlite3", path)

	require.NoError(t, err, "failed to create sqlitehints db")
	db.SetMaxOpenConns(1)

	sh, err := sqlite.NewSQLiteHints(db)
	require.NoError(t, err, "failed to setup sqlitehints db")

	runTestWith(t, sh)
}
