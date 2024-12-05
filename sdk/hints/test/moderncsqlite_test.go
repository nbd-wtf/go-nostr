//go:build !js

package test

import (
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/nbd-wtf/go-nostr/sdk/hints/sqlite"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestSQLiteHintsModernC(t *testing.T) {
	path := "/tmp/tmpsdkhintssqlite"
	os.RemoveAll(path)

	db, err := sqlx.Connect("sqlite", path)

	require.NoError(t, err, "failed to create sqlitehints db")
	db.SetMaxOpenConns(1)

	sh, err := sqlite.NewSQLiteHints(db)
	require.NoError(t, err, "failed to setup sqlitehints db")

	runTestWith(t, sh)
}
