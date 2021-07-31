package acc

import (
	_ "github.com/lib/pq"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
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

	dbInfo, err := postgresql.CalcDbConnectionInfo(db)
	require.NoError(t, err, "calc db info")

	database := postgresql.Database{
		Name:             "test-database",
		Owner:            "test-user",
	}
	require.NoError(t, database.Create(db, *dbInfo), "unexpected error")

	find := &postgresql.Database{Name: "test-database"}
	require.NoError(t, find.Read(db), "read database")
}
