package acc

import (
	"database/sql"
	"testing"
)

func createDb(t *testing.T) *sql.DB {
	connUrl := "postgres://pda:pda@localhost:8432/postgres?sslmode=disable"
	db, err := sql.Open("postgres", connUrl)
	if err != nil {
		t.Fatalf("error connecting to postgres: %s", err)
	}
	return db
}
