//go:build !js && !sqlite_math_functions

package test

import (
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/nbd-wtf/go-nostr/sdk/hints/sqlite"
	"github.com/stretchr/testify/require"
	_ "github.com/tursodatabase/go-libsql"
)

func TestSQLiteHintsLibsql(t *testing.T) {
	path := "/tmp/tmpsdkhintssqlite"
	os.RemoveAll(path)

	db, err := sqlx.Connect("libsql", "file://"+path)

	require.NoError(t, err, "failed to create sqlitehints db")
	db.SetMaxOpenConns(1)

	sh, err := sqlite.NewSQLiteHints(db)
	require.NoError(t, err, "failed to setup sqlitehints db")

	runTestWith(t, sh)
}
