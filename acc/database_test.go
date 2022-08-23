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

	db := createDb(t)
	defer db.Close()

	databases := postgresql.Databases{Db: db}
	roles := postgresql.Roles{Db: db}

	databaseName := "database-test-database"

	ownerRole, err := roles.Ensure(postgresql.Role{Name: databaseName})
	require.NoError(t, err, "error creating owner role")

	_, err = databases.Create(postgresql.Database{Name: databaseName, Owner: ownerRole.Name})
	require.NoError(t, err, "unexpected error")

	find, err := databases.Read(databaseName)
	require.NoError(t, err, "read database")
	assert.Equal(t, ownerRole.Name, find.Owner, "mismatched owner")
}
