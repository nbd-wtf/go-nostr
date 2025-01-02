//go:build !js

package test

import (
	"database/sql"
	"os"
	"testing"

	"github.com/nbd-wtf/go-nostr/sdk/hints/sqlh"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestSQLiteHintsModernC(t *testing.T) {
	path := "/tmp/tmpsdkhintssqlite"
	os.RemoveAll(path)

	db, err := sql.Open("sqlite", path)

	require.NoError(t, err, "failed to create sqlitehints db")
	db.SetMaxOpenConns(1)

	sh, err := sqlh.NewSQLHints(db, "sqlite3")
	require.NoError(t, err, "failed to setup sqlitehints db")

	runTestWith(t, sh)
}
