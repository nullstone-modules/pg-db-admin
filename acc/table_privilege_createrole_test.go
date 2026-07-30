package acc

import (
	_ "github.com/lib/pq"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

// TestTablePrivilegeAsCreateRoleAdmin reproduces the privilege model this service actually runs under.
//
// Cloud SQL never hands out a real superuser: the admin is a CREATEROLE role that created the application
// roles. Since postgres 16 that makes it a member of those roles WITH ADMIN OPTION but INHERIT FALSE, so it
// holds none of their privileges -- and `GRANT ... ON ALL TABLES IN SCHEMA` is denied on any table they own.
// A superuser never hits this, which is why the other tests do not catch it.
func TestTablePrivilegeAsCreateRoleAdmin(t *testing.T) {
	if os.Getenv("ACC") != "1" {
		t.Skip("Set ACC=1 to run e2e tests")
	}

	const (
		admin    = "tp-createrole-admin"
		owner    = "tp-createrole-owner"
		reader   = "tp-createrole-reader"
		password = "tp-createrole-password"
		// A dedicated schema, so this does not depend on who owns what in `public`
		schema = "tp_createrole"
	)

	superDb := openDb(t, "postgres")
	dropRole(t, superDb, reader)
	dropRole(t, superDb, owner)
	dropRole(t, superDb, admin)

	// A CREATEROLE admin, exactly like the one cloud sql provisions
	_, err := superDb.Exec(`CREATE ROLE "tp-createrole-admin" LOGIN PASSWORD '` + password + `' CREATEROLE`)
	require.NoError(t, err, "create admin role")

	adminStore := createStoreAs(t, admin, password)
	defer adminStore.Close()

	// The admin creates the owner role, which is what gives it ADMIN OPTION without INHERIT
	_, err = adminStore.Roles.Create(postgresql.Role{Name: owner, Password: password, UseExisting: true})
	require.NoError(t, err, "create owner role")
	_, err = adminStore.Roles.Create(postgresql.Role{Name: reader, Password: password, UseExisting: true})
	require.NoError(t, err, "create reader role")

	var inheritsOwner bool
	require.NoError(t, superDb.QueryRow(`SELECT pg_has_role($1, $2, 'USAGE')`, admin, owner).Scan(&inheritsOwner))
	require.False(t, inheritsOwner, "the admin must not already hold the owner's privileges, or this proves nothing")

	// The owner creates a table the admin has no privileges on
	_, err = superDb.Exec(`DROP SCHEMA IF EXISTS ` + schema + ` CASCADE`)
	require.NoError(t, err, "reset schema")
	_, err = superDb.Exec(`CREATE SCHEMA ` + schema + ` AUTHORIZATION "tp-createrole-owner"`)
	require.NoError(t, err, "create schema owned by the owner role")

	ownerDb := openDbAs(t, owner, password, "postgres")
	_, err = ownerDb.Exec(`CREATE TABLE ` + schema + `.items ( id SERIAL NOT NULL )`)
	require.NoError(t, err, "create table owned by the owner role")

	hasSelect := func(t *testing.T, table string) bool {
		var granted bool
		require.NoError(t, superDb.QueryRow(`SELECT has_table_privilege($1, $2, 'SELECT')`, reader, table).Scan(&granted))
		return granted
	}
	require.False(t, hasSelect(t, schema+".items"), "reader should not have access before the grant")

	_, err = adminStore.TablePrivileges.Create(postgresql.TablePrivilege{
		Database:              "postgres",
		Schema:                schema,
		Role:                  reader,
		IncludeFuture:         true,
		FutureFromTableOwners: true,
	})
	require.NoError(t, err, "grant table privileges as a CREATEROLE admin")

	assert.True(t, hasSelect(t, schema+".items"), "reader should have select on the owner's table")

	t.Run("covers tables the owner creates later", func(t *testing.T) {
		_, err := ownerDb.Exec(`CREATE TABLE ` + schema + `.future_items ( id SERIAL NOT NULL )`)
		require.NoError(t, err, "create future table")

		assert.True(t, hasSelect(t, schema+".future_items"), "reader should have select on tables created after the grant")
	})

	t.Run("releases the temporary membership", func(t *testing.T) {
		var stillInherits bool
		require.NoError(t, superDb.QueryRow(`SELECT pg_has_role($1, $2, 'USAGE')`, admin, owner).Scan(&stillInherits))
		assert.False(t, stillInherits, "the admin should not keep the owner's privileges after the grant")
	})

	t.Run("keeps the admin option it started with", func(t *testing.T) {
		var adminOption bool
		sq := `SELECT m.admin_option
FROM pg_auth_members m
	JOIN pg_roles r ON r.oid = m.roleid
	JOIN pg_roles mem ON mem.oid = m.member
WHERE r.rolname = $1 AND mem.rolname = $2`
		require.NoError(t, superDb.QueryRow(sq, owner, admin).Scan(&adminOption))
		assert.True(t, adminOption, "revoking the temporary membership must not strip the admin option")
	})
}
