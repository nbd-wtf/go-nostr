package sqlite

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/sdk/hints"
)

type SQLiteHints struct {
	*sqlx.DB

	saves [7]*sqlx.Stmt
	topN  *sqlx.Stmt
}

func NewSQLiteHints(db *sqlx.DB) (SQLiteHints, error) {
	sh := SQLiteHints{DB: db}

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

	_, err := sh.Exec(`CREATE TABLE pubkey_relays (pubkey text, relay text, ` + cols.String())
	if err != nil {
		return SQLiteHints{}, err
	}

	_, err = sh.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS pkr ON pubkey_relays (pubkey, relay)`)
	if err != nil {
		return SQLiteHints{}, err
	}

	_, err = sh.Exec(`CREATE INDEX IF NOT EXISTS bypk ON pubkey_relays (pubkey)`)
	if err != nil {
		return SQLiteHints{}, err
	}

	// prepare statements
	for i := range hints.KeyBasePoints {
		col := hints.HintKey(i).String()

		stmt, err := sh.Preparex(
			`INSERT INTO pubkey_relays (pubkey, relay, ` + col + `) VALUES (?, ?, ?)
			 ON CONFLICT (pubkey, relay) DO UPDATE SET ` + col + ` = max(?, coalesce(` + col + `, 0))`,
		)
		if err != nil {
			return sh, fmt.Errorf("failed to prepare statement for %s: %w", col, err)
		}
		sh.saves[i] = stmt
	}

	{
		stmt, err := sh.Preparex(
			`SELECT relay FROM pubkey_relays WHERE pubkey = ? ORDER BY (` + scorePartialQuery() + `) DESC LIMIT ?`,
		)
		if err != nil {
			return sh, fmt.Errorf("failed to prepare statement for querying: %w", err)
		}
		sh.topN = stmt
	}

	return sh, nil
}

func (sh SQLiteHints) TopN(pubkey string, n int) []string {
	res := make([]string, 0, n)
	err := sh.topN.Select(&res, pubkey, n)
	if err != nil && err != sql.ErrNoRows {
		nostr.InfoLogger.Printf("[sdk/hints/sqlite] unexpected error on query for %s: %s\n",
			pubkey, err)
	}
	return res
}

func (sh SQLiteHints) Save(pubkey string, relay string, key hints.HintKey, ts nostr.Timestamp) {
	if now := nostr.Now(); ts > now {
		ts = now
	}

	_, err := sh.saves[key].Exec(pubkey, relay, ts, ts)
	if err != nil {
		nostr.InfoLogger.Printf("[sdk/hints/sqlite] unexpected error on insert for %s, %s, %d: %s\n",
			pubkey, relay, ts, err)
	}
}

func (sh SQLiteHints) PrintScores() {
	fmt.Println("= print scores")

	allpubkeys := make([]string, 0, 50)
	if err := sh.Select(&allpubkeys, `SELECT DISTINCT pubkey FROM pubkey_relays`); err != nil {
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
			`SELECT pubkey, relay, coalesce(`+scorePartialQuery()+`, 0) AS score
			 FROM pubkey_relays WHERE pubkey = ? ORDER BY score DESC`, pubkey); err != nil {
			panic(err)
		}

		for i, re := range allrelays {
			fmt.Printf("  %3d :: %30s ::> %12d\n", i, re.Relay, int(re.Score))
		}
	}
}

func scorePartialQuery() string {
	calc := strings.Builder{}
	calc.Grow(len(hints.KeyBasePoints) * (10 + 25 + 51 + 25 + 24 + 4 + 12 + 3))

	for i, points := range hints.KeyBasePoints {
		col := hints.HintKey(i).String()
		multiplier := strconv.FormatInt(points, 10)

		calc.WriteString(`(CASE WHEN `)
		calc.WriteString(col)
		calc.WriteString(` IS NOT NULL THEN 10000000000 * `)
		calc.WriteString(multiplier)
		calc.WriteString(` / power(max(1, (unixepoch() + 86400) - `)
		calc.WriteString(col)
		calc.WriteString(`), 1.3) ELSE 0 END)`)

		if i != len(hints.KeyBasePoints)-1 {
			calc.WriteString(` + `)
		}
	}

	return calc.String()
}
