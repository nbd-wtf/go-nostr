//go:build !js && !sqlite_math_functions

package test

import (
	"database/sql"
	"os"
	"testing"

	"github.com/nbd-wtf/go-nostr/sdk/hints/sqlh"
	"github.com/stretchr/testify/require"
	_ "github.com/tursodatabase/go-libsql"
)

func TestSQLiteHintsLibsql(t *testing.T) {
	path := "/tmp/tmpsdkhintssqlite"
	os.RemoveAll(path)

	db, err := sql.Open("libsql", "file://"+path)

	require.NoError(t, err, "failed to create sqlitehints db")
	db.SetMaxOpenConns(1)

	sh, err := sqlh.NewSQLHints(db, "sqlite3")
	require.NoError(t, err, "failed to setup sqlitehints db")

	runTestWith(t, sh)
}
