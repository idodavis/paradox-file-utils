// dbservice_test.go covers the current schema having no games catalog table.
package services

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func TestInitSchemaHasNoGamesTable(t *testing.T) {
	db := sqlx.MustConnect("sqlite", ":memory:")
	d := &DbService{DB: db}
	if err := d.initSchema(); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.Get(&n,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'games'`,
	); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("games table exists after initSchema")
	}
	if err := db.Get(&n,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'game_installs'`,
	); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("game_installs missing after initSchema")
	}
}
