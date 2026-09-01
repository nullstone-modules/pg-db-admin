package acc

import (
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestRole(t *testing.T) {
	if os.Getenv("ACC") != "1" {
		t.Skip("Set ACC=1 to run e2e tests")
	}

	store := createStore(t)
	defer store.Close()

	dropRole(t, openDb(t, "postgres"), "role-test-user")

	_, err := store.Roles.Create(postgresql.Role{
		Name:     "role-test-user",
		Password: "role-test-password",
	})
	require.NoError(t, err, "unexpected error")

	find, err := store.Roles.Read("role-test-user")
	require.NoError(t, err, "read user")
	require.NotNil(t, find)
}

// TestRoleReplication verifies the REPLICATION attribute, which a logical replication consumer needs
// and which the admin role itself needs before it can create a replication slot
func TestRoleReplication(t *testing.T) {
	if os.Getenv("ACC") != "1" {
		t.Skip("Set ACC=1 to run e2e tests")
	}

	store := createStore(t)
	defer store.Close()

	t.Run("granted at creation", func(t *testing.T) {
		name := "role-test-replicator"
		dropRole(t, openDb(t, "postgres"), name)

		_, err := store.Roles.Create(postgresql.Role{
			Name:       name,
			Password:   "role-test-password",
			Attributes: postgresql.RoleAttributes{Replication: true},
		})
		require.NoError(t, err, "create role")

		find, err := store.Roles.Read(name)
		require.NoError(t, err, "read role")
		require.NotNil(t, find)
		assert.True(t, find.Attributes.Replication, "expected the role to have REPLICATION")
	})

	t.Run("added to an existing role", func(t *testing.T) {
		name := "role-test-existing-replicator"
		dropRole(t, openDb(t, "postgres"), name)

		_, err := store.Roles.Create(postgresql.Role{Name: name, Password: "role-test-password"})
		require.NoError(t, err, "create role")

		find, err := store.Roles.Read(name)
		require.NoError(t, err, "read role")
		require.NotNil(t, find)
		require.False(t, find.Attributes.Replication, "expected the role to start without REPLICATION")

		_, err = store.Roles.Create(postgresql.Role{
			Name:        name,
			UseExisting: true,
			Attributes:  postgresql.RoleAttributes{Replication: true},
		})
		require.NoError(t, err, "add REPLICATION to the existing role")

		find, err = store.Roles.Read(name)
		require.NoError(t, err, "read role")
		require.NotNil(t, find)
		assert.True(t, find.Attributes.Replication, "expected the role to have REPLICATION")
	})
}
