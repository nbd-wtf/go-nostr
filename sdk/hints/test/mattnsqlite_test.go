//go:build sqlite_math_functions

package test

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nbd-wtf/go-nostr/sdk/hints/sqlh"
	"github.com/stretchr/testify/require"
)

func TestSQLiteHintsMattn(t *testing.T) {
	path := "/tmp/tmpsdkhintssqlite"
	os.RemoveAll(path)

	db, err := sql.Open("sqlite3", path)

	require.NoError(t, err, "failed to create sqlitehints db")
	db.SetMaxOpenConns(1)

	sh, err := sqlh.NewSQLHints(db, "sqlite3")
	require.NoError(t, err, "failed to setup sqlitehints db")

	runTestWith(t, sh)
}
