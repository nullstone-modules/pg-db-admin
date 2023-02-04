package acc

import (
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"testing"
)

func createStore(t *testing.T) *postgresql.Store {
	return postgresql.NewStore("postgres://pda:pda@localhost:8432/postgres?sslmode=disable")
}
