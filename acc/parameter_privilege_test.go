package acc

import (
	_ "github.com/lib/pq"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

// TestParameterPrivilege covers granting a role the right to SET a superuser-context parameter.
//
// Membership in a superuser-ish role does not confer this: postgres does not inherit role attributes
// through membership. The only thing that makes `SET session_replication_role = replica` work for a
// plain application role is a privilege on the parameter itself.
func TestParameterPrivilege(t *testing.T) {
	if os.Getenv("ACC") != "1" {
		t.Skip("Set ACC=1 to run e2e tests")
	}

	const (
		role      = "pp-app"
		password  = "pp-password"
		parameter = "session_replication_role"
	)

	superDb := openDb(t, "postgres")
	// The grant survives a DROP ROLE only if it was revoked first, and the acceptance database is
	// long-lived, so an earlier run would otherwise leave the privilege in place and mask a regression
	superDb.Exec(`REVOKE SET ON PARAMETER ` + parameter + ` FROM "pp-app" CASCADE`)
	dropRole(t, superDb, role)

	store := createStore(t)
	defer store.Close()

	// A role that does not exist yet reads as "no privilege" rather than erroring: terraform reads
	// before the role has been created
	missing, err := store.ParameterPrivileges.Read(postgresql.ParameterPrivilegeKey{Role: role, Parameter: parameter})
	require.NoError(t, err, "read for a role that does not exist")
	assert.Nil(t, missing, "a role that does not exist holds no privilege")

	_, err = store.Roles.Create(postgresql.Role{Name: role, Password: password, UseExisting: true})
	require.NoError(t, err, "create app role")

	before, err := store.ParameterPrivileges.Read(postgresql.ParameterPrivilegeKey{Role: role, Parameter: parameter})
	require.NoError(t, err, "read before the grant")
	require.Nil(t, before, "role should not hold the privilege before the grant")

	// Without the privilege, setting the parameter is refused
	appDb := openDbAs(t, role, password, "postgres")
	_, err = appDb.Exec(`SET ` + parameter + ` = replica`)
	require.Error(t, err, "setting the parameter should be denied before the grant")
	appDb.Close()

	created, err := store.ParameterPrivileges.Create(postgresql.ParameterPrivilege{Role: role, Parameter: parameter})
	require.NoError(t, err, "grant SET on the parameter")
	require.NotNil(t, created)
	assert.Equal(t, role, created.Role)
	assert.Equal(t, parameter, created.Parameter)

	after, err := store.ParameterPrivileges.Read(postgresql.ParameterPrivilegeKey{Role: role, Parameter: parameter})
	require.NoError(t, err, "read after the grant")
	require.NotNil(t, after, "role should hold the privilege after the grant")

	// The privilege is only worth anything if the role can actually set the parameter
	grantedDb := openDbAs(t, role, password, "postgres")
	defer grantedDb.Close()
	_, err = grantedDb.Exec(`SET ` + parameter + ` = replica`)
	require.NoError(t, err, "setting the parameter should be permitted after the grant")
	var current string
	require.NoError(t, grantedDb.QueryRow(`SHOW `+parameter).Scan(&current))
	assert.Equal(t, "replica", current, "the parameter should be applied to the session")

	// Re-applying is a no-op, not an error -- every apply reissues this resource
	_, err = store.ParameterPrivileges.Create(postgresql.ParameterPrivilege{Role: role, Parameter: parameter})
	assert.NoError(t, err, "granting an already-granted privilege should succeed")
}

// TestParameterPrivilegeRejectsUnsafeName covers the validation that makes interpolating the parameter
// name safe. The name cannot be quoted as an identifier, so anything that is not a plain configuration
// parameter has to be refused before it reaches the statement.
func TestParameterPrivilegeRejectsUnsafeName(t *testing.T) {
	if os.Getenv("ACC") != "1" {
		t.Skip("Set ACC=1 to run e2e tests")
	}

	const role = "pp-reject"

	superDb := openDb(t, "postgres")
	dropRole(t, superDb, role)

	store := createStore(t)
	defer store.Close()

	_, err := store.Roles.Create(postgresql.Role{Name: role, Password: "pp-password", UseExisting: true})
	require.NoError(t, err, "create app role")

	for _, parameter := range []string{
		"session_replication_role; DROP ROLE \"pp-reject\"",
		"session_replication_role, log_statement",
		"pg_stat_statements.track",
		"Session_Replication_Role",
		"",
	} {
		t.Run(parameter, func(t *testing.T) {
			_, err := store.ParameterPrivileges.Create(postgresql.ParameterPrivilege{Role: role, Parameter: parameter})
			assert.Error(t, err, "parameter %q should be rejected", parameter)
		})
	}

	// The injection attempt above must not have taken effect
	var stillExists bool
	require.NoError(t, superDb.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, role).Scan(&stillExists))
	assert.True(t, stillExists, "the role should not have been dropped by an injected statement")
}
