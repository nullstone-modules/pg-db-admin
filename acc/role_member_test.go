package acc

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

// hasRole asks the question the membership exists to answer: can member act with role's privileges.
// A row in pg_auth_members is not the same question since postgres 16.
func hasRole(t *testing.T, db *sql.DB, member, role string) bool {
	var has bool
	require.NoError(t, db.QueryRow(`SELECT pg_has_role($1, $2, 'USAGE')`, member, role).Scan(&has))
	return has
}

func TestRoleMemberEnsure(t *testing.T) {
	if os.Getenv("ACC") != "1" {
		t.Skip("Set ACC=1 to run e2e tests")
	}

	store := createStore(t)
	defer store.Close()
	db := openDb(t, "postgres")
	defer db.Close()

	target, other, member := "role-member-target", "role-member-other", "role-member-member"
	for _, name := range []string{member, target, other} {
		dropRole(t, db, name)
	}
	for _, name := range []string{target, other, member} {
		_, err := store.Roles.Create(postgresql.Role{Name: name})
		require.NoError(t, err, "create role %q", name)
	}

	key := postgresql.RoleMemberKey{Member: member, Target: target}

	t.Run("create grants", func(t *testing.T) {
		_, err := store.RoleMembers.Create(postgresql.RoleMember{Member: member, Target: target, UseExisting: true})
		require.NoError(t, err)
		require.True(t, hasRole(t, db, member, target))
	})

	t.Run("create again is a no-op", func(t *testing.T) {
		_, err := store.RoleMembers.Create(postgresql.RoleMember{Member: member, Target: target, UseExisting: true})
		require.NoError(t, err)
		require.True(t, hasRole(t, db, member, target))
	})

	t.Run("update with the same input is a no-op", func(t *testing.T) {
		_, err := store.RoleMembers.Update(key, postgresql.RoleMember{Member: member, Target: target, UseExisting: true})
		require.NoError(t, err)
		require.True(t, hasRole(t, db, member, target))
	})

	t.Run("update with a changed target grants the new membership", func(t *testing.T) {
		// What Terraform sends when a capability's input changes on an existing invocation. This used
		// to be a no-op, which left the restore role unable to act as the new owner.
		require.False(t, hasRole(t, db, member, other))
		_, err := store.RoleMembers.Update(key, postgresql.RoleMember{Member: member, Target: other, UseExisting: true})
		require.NoError(t, err)
		require.True(t, hasRole(t, db, member, other))
	})

	t.Run("a non-inheriting row is repaired", func(t *testing.T) {
		info, err := postgresql.CalcDbConnectionInfo(db)
		require.NoError(t, err)
		if !info.SupportedFeatures.IsSupported(postgresql.FeatureRoleInheritOption) {
			t.Skip("INHERIT FALSE memberships need postgres 16")
		}

		// The postgres 16 trap: a row exists, and confers nothing
		_, err = db.Exec(fmt.Sprintf(`REVOKE %s FROM %s`, pq.QuoteIdentifier(target), pq.QuoteIdentifier(member)))
		require.NoError(t, err)
		_, err = db.Exec(fmt.Sprintf(`GRANT %s TO %s WITH INHERIT FALSE`, pq.QuoteIdentifier(target), pq.QuoteIdentifier(member)))
		require.NoError(t, err)
		existing, err := store.RoleMembers.Read(key)
		require.NoError(t, err)
		require.NotNil(t, existing, "the row is there")
		require.False(t, hasRole(t, db, member, target), "and it confers nothing")

		_, err = store.RoleMembers.Create(postgresql.RoleMember{Member: member, Target: target, UseExisting: true})
		require.NoError(t, err)
		require.True(t, hasRole(t, db, member, target), "the row must not be mistaken for the privilege")
	})

	t.Run("admin option is granted", func(t *testing.T) {
		_, err := store.RoleMembers.Create(postgresql.RoleMember{Member: member, Target: target, WithAdminOption: true})
		require.NoError(t, err)
		existing, err := store.RoleMembers.Read(key)
		require.NoError(t, err)
		require.NotNil(t, existing)
		require.True(t, existing.WithAdminOption)
		require.True(t, hasRole(t, db, member, target))
	})
}
