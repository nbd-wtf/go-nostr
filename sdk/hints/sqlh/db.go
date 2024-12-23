package sqlh

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/sdk/hints"
)

type SQLHints struct {
	*sqlx.DB

	interop interop
	saves   [7]*sqlx.Stmt
	topN    *sqlx.Stmt
}

// NewSQLHints takes an sqlx.DB connection (db) and a database type name (driverName ).
// driverName must be either "postgres" or "sqlite3" -- this is so we can slightly change the queries.
func NewSQLHints(db *sql.DB, driverName string) (SQLHints, error) {
	sh := SQLHints{DB: sqlx.NewDb(db, driverName)}

	switch driverName {
	case "sqlite3":
		sh.interop = sqliteInterop
	case "postgres":
		sh.interop = postgresInterop
	default:
		return sh, fmt.Errorf("unknown database driver '%s'", driverName)
	}

	// create table and indexes
	cols := strings.Builder{}
	cols.Grow(len(hints.KeyBasePoints) * 20)
	for i := range hints.KeyBasePoints {
		name := hints.HintKey(i).String()
		cols.WriteString(name)
		cols.WriteString(" integer")
		if i == len(hints.KeyBasePoints)-1 {
			cols.WriteString(")")
		} else {
			cols.WriteString(",")
		}
	}

	_, err := sh.Exec(`CREATE TABLE IF NOT EXISTS nostr_sdk_pubkey_relays (pubkey text, relay text, ` + cols.String())
	if err != nil {
		return SQLHints{}, err
	}

	_, err = sh.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS pkr ON nostr_sdk_pubkey_relays (pubkey, relay)`)
	if err != nil {
		return SQLHints{}, err
	}

	_, err = sh.Exec(`CREATE INDEX IF NOT EXISTS bypk ON nostr_sdk_pubkey_relays (pubkey)`)
	if err != nil {
		return SQLHints{}, err
	}

	// prepare statements
	for i := range hints.KeyBasePoints {
		col := hints.HintKey(i).String()

		stmt, err := sh.Preparex(
			`INSERT INTO nostr_sdk_pubkey_relays (pubkey, relay, ` + col + `) VALUES (` + sh.interop.generateBindingSpots(0, 3) + `)
			 ON CONFLICT (pubkey, relay) DO UPDATE SET ` + col + ` = ` + sh.interop.maxFunc + `(` + sh.interop.generateBindingSpots(3, 1) + `, coalesce(excluded.` + col + `, 0))`,
		)
		if err != nil {
			fmt.Println(
				`INSERT INTO nostr_sdk_pubkey_relays (pubkey, relay, ` + col + `) VALUES (` + sh.interop.generateBindingSpots(0, 3) + `)
			 ON CONFLICT (pubkey, relay) DO UPDATE SET ` + col + ` = ` + sh.interop.maxFunc + `(` + sh.interop.generateBindingSpots(3, 1) + `, coalesce(excluded.` + col + `, 0))`,
			)
			return sh, fmt.Errorf("failed to prepare statement for %s: %w", col, err)
		}
		sh.saves[i] = stmt
	}

	{
		stmt, err := sh.Preparex(
			`SELECT relay FROM nostr_sdk_pubkey_relays WHERE pubkey = ` + sh.interop.generateBindingSpots(0, 1) + ` ORDER BY (` + sh.scorePartialQuery() + `) DESC LIMIT ` + sh.interop.generateBindingSpots(1, 1),
		)
		if err != nil {
			return sh, fmt.Errorf("failed to prepare statement for querying: %w", err)
		}
		sh.topN = stmt
	}

	return sh, nil
}

func (sh SQLHints) TopN(pubkey string, n int) []string {
	res := make([]string, 0, n)
	err := sh.topN.Select(&res, pubkey, n)
	if err != nil && err != sql.ErrNoRows {
		nostr.InfoLogger.Printf("[sdk/hints/sql] unexpected error on query for %s: %s\n",
			pubkey, err)
	}
	return res
}

func (sh SQLHints) Save(pubkey string, relay string, key hints.HintKey, ts nostr.Timestamp) {
	if now := nostr.Now(); ts > now {
		ts = now
	}

	_, err := sh.saves[key].Exec(pubkey, relay, ts, ts)
	if err != nil {
		nostr.InfoLogger.Printf("[sdk/hints/sql] unexpected error on insert for %s, %s, %d: %s\n",
			pubkey, relay, ts, err)
	}
}

func (sh SQLHints) PrintScores() {
	fmt.Println("= print scores")

	allpubkeys := make([]string, 0, 50)
	if err := sh.Select(&allpubkeys, `SELECT DISTINCT pubkey FROM nostr_sdk_pubkey_relays`); err != nil {
		panic(err)
	}

	allrelays := make([]struct {
		PubKey string  `db:"pubkey"`
		Relay  string  `db:"relay"`
		Score  float64 `db:"score"`
	}, 0, 20)
	for _, pubkey := range allpubkeys {
		fmt.Println("== relay scores for", pubkey)
		if err := sh.Select(&allrelays,
			`SELECT pubkey, relay, coalesce(`+sh.scorePartialQuery()+`, 0) AS score
			 FROM nostr_sdk_pubkey_relays WHERE pubkey = `+sh.interop.generateBindingSpots(0, 1)+` ORDER BY score DESC`, pubkey); err != nil {
			panic(err)
		}

		for i, re := range allrelays {
			fmt.Printf("  %3d :: %30s ::> %12d\n", i, re.Relay, int(re.Score))
		}
	}
}

func (sh SQLHints) scorePartialQuery() string {
	calc := strings.Builder{}
	calc.Grow(len(hints.KeyBasePoints) * (11 + 25 + 32 + 4 + 4 + 9 + 12 + 25 + 12 + 25 + 19 + 3))

	for i, points := range hints.KeyBasePoints {
		col := hints.HintKey(i).String()
		multiplier := strconv.FormatInt(points, 10)

		calc.WriteString(`(CASE WHEN `)
		calc.WriteString(col)
		calc.WriteString(` IS NOT NULL THEN 10000000000 * `)
		calc.WriteString(multiplier)
		calc.WriteString(` / power(`)
		calc.WriteString(sh.interop.maxFunc)
		calc.WriteString(`(1, (`)
		calc.WriteString(sh.interop.getUnixEpochFunc)
		calc.WriteString(` + 86400) - `)
		calc.WriteString(col)
		calc.WriteString(`), 1.3) ELSE 0 END)`)

		if i != len(hints.KeyBasePoints)-1 {
			calc.WriteString(` + `)
		}
	}

	return calc.String()
}
