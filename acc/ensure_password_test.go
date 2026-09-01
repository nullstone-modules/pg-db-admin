package acc

import (
	"database/sql"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

// rolePasswordHash reads the stored hash directly. SCRAM salts every ALTER, so an unchanged hash
// proves the login probe skipped the ALTER -- re-setting even the same password would change it.
func rolePasswordHash(t *testing.T, db *sql.DB, name string) string {
	var hash sql.NullString
	err := db.QueryRow(`SELECT rolpassword FROM pg_authid WHERE rolname = $1`, name).Scan(&hash)
	require.NoError(t, err, "read rolpassword for %q", name)
	return hash.String
}

func TestEnsurePassword(t *testing.T) {
	if os.Getenv("ACC") != "1" {
		t.Skip("Set ACC=1 to run e2e tests")
	}

	store := createStore(t)
	defer store.Close()
	db := openDb(t, "postgres")
	defer db.Close()

	name := "ensure-password-role"
	dropRole(t, db, name)

	t.Run("missing role errors", func(t *testing.T) {
		_, err := store.Roles.EnsurePassword(name, "ensure-pw-1")
		require.ErrorIs(t, err, postgresql.ErrRoleNotFound)
	})

	t.Run("passwordless role is healed", func(t *testing.T) {
		// The restore-access scenario: the role was created with LOGIN but no password
		_, err := store.Roles.Create(postgresql.Role{Name: name})
		require.NoError(t, err, "create passwordless role")

		result, err := store.Roles.EnsurePassword(name, "ensure-pw-1")
		require.NoError(t, err)
		require.True(t, result.Changed, "expected the NULL password to be set")
		openDbAs(t, name, "ensure-pw-1", "postgres").Close()
	})

	t.Run("valid password is untouched", func(t *testing.T) {
		before := rolePasswordHash(t, db, name)
		result, err := store.Roles.EnsurePassword(name, "ensure-pw-1")
		require.NoError(t, err)
		require.False(t, result.Changed)
		require.Equal(t, before, rolePasswordHash(t, db, name), "expected no ALTER for a working password")
	})

	t.Run("invalid password is replaced", func(t *testing.T) {
		before := rolePasswordHash(t, db, name)
		result, err := store.Roles.EnsurePassword(name, "ensure-pw-2")
		require.NoError(t, err)
		require.True(t, result.Changed)
		require.NotEqual(t, before, rolePasswordHash(t, db, name))
		openDbAs(t, name, "ensure-pw-2", "postgres").Close()
	})
}

func TestRoleUpdateSkipsValidPassword(t *testing.T) {
	if os.Getenv("ACC") != "1" {
		t.Skip("Set ACC=1 to run e2e tests")
	}

	store := createStore(t)
	defer store.Close()
	db := openDb(t, "postgres")
	defer db.Close()

	name := "update-skip-role"
	dropRole(t, db, name)

	_, err := store.Roles.Create(postgresql.Role{Name: name, Password: "update-skip-pw-1"})
	require.NoError(t, err, "create role")
	before := rolePasswordHash(t, db, name)

	// Same password: the login probe short-circuits the ALTER
	_, err = store.Roles.Update(name, postgresql.Role{Name: name, Password: "update-skip-pw-1"})
	require.NoError(t, err)
	require.Equal(t, before, rolePasswordHash(t, db, name), "expected no ALTER for a working password")

	// Different password: still alters as before
	_, err = store.Roles.Update(name, postgresql.Role{Name: name, Password: "update-skip-pw-2"})
	require.NoError(t, err)
	require.NotEqual(t, before, rolePasswordHash(t, db, name))
	openDbAs(t, name, "update-skip-pw-2", "postgres").Close()
}
