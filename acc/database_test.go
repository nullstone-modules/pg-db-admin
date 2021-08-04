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

	database := postgresql.Database{Name: "test-database"}

	ownerRole := postgresql.Role{Name: database.Name}
	require.NoError(t, ownerRole.Ensure(db), "error creating owner role")
	database.Owner = ownerRole.Name

	dbInfo, err := postgresql.CalcDbConnectionInfo(db)
	require.NoError(t, err, "calc db info")

	require.NoError(t, database.Create(db, *dbInfo), "unexpected error")

	find := &postgresql.Database{Name: "test-database"}
	require.NoError(t, find.Read(db), "read database")
	assert.Equal(t, ownerRole.Name, find.Owner, "mismatched owner")
}
