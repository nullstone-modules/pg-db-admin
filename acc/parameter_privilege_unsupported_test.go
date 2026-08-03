package acc

import (
	_ "github.com/lib/pq"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

// TestParameterPrivilegeUnsupportedVersion covers a server older than postgres 15.
//
// Both halves of the resource have to fail loudly there. `GRANT SET ON PARAMETER` is a syntax error
// before 15, and neither has_parameter_privilege() nor pg_parameter_acl exists, so an ungated read
// fails with "function does not exist". Reporting either as success would leave a workload believing
// it holds a privilege it does not, and it would only find out partway through a restore.
//
// Point ACC_PG14_URL at a pre-15 server to run this, for example:
//
//	docker run -d --name pg14 -e POSTGRES_USER=pda -e POSTGRES_PASSWORD=pda -p 8433:5432 postgres:14
//	ACC=1 ACC_PG14_URL="postgres://pda:pda@localhost:8433/postgres?sslmode=disable" go test ./acc/ -run Unsupported
func TestParameterPrivilegeUnsupportedVersion(t *testing.T) {
	if os.Getenv("ACC") != "1" {
		t.Skip("Set ACC=1 to run e2e tests")
	}
	connUrl := os.Getenv("ACC_PG14_URL")
	if connUrl == "" {
		t.Skip("Set ACC_PG14_URL to a pre-15 postgres to run this test")
	}

	const (
		role      = "pp-old"
		parameter = "session_replication_role"
	)

	store := postgresql.NewStore(connUrl)
	defer store.Close()

	_, err := store.Roles.Create(postgresql.Role{Name: role, Password: "pp-password", UseExisting: true})
	require.NoError(t, err, "create app role")

	key := postgresql.ParameterPrivilegeKey{Role: role, Parameter: parameter}

	_, err = store.ParameterPrivileges.Create(postgresql.ParameterPrivilege{Role: role, Parameter: parameter})
	require.Error(t, err, "the grant must fail rather than silently do nothing")
	assert.Contains(t, err.Error(), "requires postgres 15 or later", "the error should explain why")

	_, err = store.ParameterPrivileges.Read(key)
	require.Error(t, err, "the read must fail rather than report the privilege as absent")
	assert.Contains(t, err.Error(), "requires postgres 15 or later", "the error should explain why")
}
