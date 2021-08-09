package acc

import (
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/nullstone-modules/pg-db-admin/workflows"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestFull tests the entire workflow of create-database, create-user, create-db-access
func TestFull(t *testing.T) {
	//if os.Getenv("ACC") != "1" {
	//	t.Skip("Set ACC=1 to run e2e tests")
	//}
	connUrl := "postgres://pda:pda@localhost:8432/postgres?sslmode=disable"

	newDatabase := postgresql.Database{
		Name: "test-database",
	}
	newUser := postgresql.Role{
		Name:     "test-user",
		Password: "test-password",
	}

	require.NoErrorf(t, workflows.EnsureDatabase(connUrl, newDatabase), "ensure database")
	require.NoError(t, workflows.EnsureUser(connUrl, newUser), "ensure user")
	require.NoError(t, workflows.GrantDbAccess(connUrl, newUser, newDatabase), "grant db access")

	// TODO: Verify new user can connect
}
