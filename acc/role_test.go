package acc

import (
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestRole(t *testing.T) {
	if os.Getenv("ACC") != "1" {
		t.Skip("Set ACC=1 to run e2e tests")
	}

	db, store := createDb(t)
	defer db.Close()

	_, err := store.Roles.Create(postgresql.Role{
		Name:     "role-test-user",
		Password: "role-test-password",
	})
	require.NoError(t, err, "unexpected error")

	find, err := store.Roles.Read("role-test-user")
	require.NoError(t, err, "read user")
	require.NotNil(t, find)
}
