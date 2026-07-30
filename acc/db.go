package acc

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/stretchr/testify/require"
	"testing"
)

const connUrl = "postgres://pda:pda@localhost:8432/postgres?sslmode=disable"

func createStore(t *testing.T) *postgresql.Store {
	return postgresql.NewStore(connUrl)
}

// connUrlAs builds a connection url for a role other than the superuser the tests normally use.
// Cloud SQL never gives out a real superuser, so the behavior that matters in production is the behavior
// of a CREATEROLE admin -- which differs enough that some tests have to connect as one.
func connUrlAs(user, password string) string {
	return "postgres://" + user + ":" + password + "@localhost:8432/postgres?sslmode=disable"
}

func createStoreAs(t *testing.T, user, password string) *postgresql.Store {
	return postgresql.NewStore(connUrlAs(user, password))
}

func openDbAs(t *testing.T, user, password, database string) *sql.DB {
	db, err := postgresql.OpenDatabase(connUrlAs(user, password), database)
	require.NoError(t, err, "connecting to %q as %q", database, user)
	return db
}

// openDb opens a direct connection so that tests can verify the effects of the store's operations
func openDb(t *testing.T, database string) *sql.DB {
	db, err := postgresql.OpenDatabase(connUrl, database)
	require.NoError(t, err, "connecting to %q", database)
	return db
}

// dropRole removes a role along with everything granted to it, so that a test starts from a known state
// The acceptance database is long-lived, so tests that assert on privileges have to clean up after previous runs
func dropRole(t *testing.T, db *sql.DB, name string) {
	quoted := pq.QuoteIdentifier(name)
	// Both statements fail when the role does not exist yet, which is the expected state on a fresh database
	db.Exec("DROP OWNED BY " + quoted)
	db.Exec("DROP ROLE IF EXISTS " + quoted)
}
