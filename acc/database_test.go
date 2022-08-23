package acc

import (
	_ "github.com/lib/pq"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestDatabase(t *testing.T) {
	if os.Getenv("ACC") != "1" {
		t.Skip("Set ACC=1 to run e2e tests")
	}

	db, store := createDb(t)
	defer db.Close()

	databaseName := "database-test-database"

	ownerRole, err := store.Roles.Ensure(postgresql.Role{Name: databaseName})
	require.NoError(t, err, "error creating owner role")

	_, err = store.Databases.Create(postgresql.Database{Name: databaseName, Owner: ownerRole.Name})
	require.NoError(t, err, "unexpected error")

	find, err := store.Databases.Read(databaseName)
	require.NoError(t, err, "read database")
	assert.Equal(t, ownerRole.Name, find.Owner, "mismatched owner")
}
