package acc

import (
	"database/sql"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"testing"
)

func createDb(t *testing.T) (*sql.DB, postgresql.Store) {
	connUrl := "postgres://pda:pda@localhost:8432/postgres?sslmode=disable"
	db, err := sql.Open("postgres", connUrl)
	if err != nil {
		t.Fatalf("error connecting to postgres: %s", err)
	}
	return db, postgresql.NewStore(connUrl)
}
