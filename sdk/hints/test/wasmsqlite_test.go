//go:build !js && !sqlite_math_functions

package test

import (
	"database/sql"
	"os"
	"testing"

	"github.com/nbd-wtf/go-nostr/sdk/hints/sqlh"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/require"
)

func TestSQLiteHintsNcruces(t *testing.T) {
	path := "/tmp/tmpsdkhintssqlite"
	os.RemoveAll(path)

	db, err := sql.Open("sqlite3", path)

	require.NoError(t, err, "failed to create sqlitehints db")
	db.SetMaxOpenConns(1)

	sh, err := sqlh.NewSQLHints(db, "sqlite3")
	require.NoError(t, err, "failed to setup sqlitehints db")

	runTestWith(t, sh)
}
