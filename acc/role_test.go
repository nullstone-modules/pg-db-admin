package acc

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestRole(t *testing.T) {
	if os.Getenv("ACC") != "1" {
		t.Skip("Set ACC=1 to run e2e tests")
	}

	ctx := context.Background()
	connUrl := "postgres://pda:pda@localhost:8432/postgres?sslmode=disable"
	conn, err := pgx.Connect(ctx, connUrl)
	if err != nil {
		t.Fatalf("error connecting to postgres: %s", err)
	}
	defer conn.Close(ctx)

	role := postgresql.Role{
		Name:     "test-user",
		Password: "test-password",
	}
	require.NoError(t, role.Create(conn), "unexpected error")
	rows, err := conn.Query(ctx, `SELECT rolname from pg_roles`)
	require.NoError(t, err, "querying roles")
	for rows.Next() {
		var roleName string
		require.NoError(t, rows.Scan(&roleName), "scanning role name")
		if roleName == "test-user" {
			return
		}
	}
	t.Errorf("expected to create role, did not")
}
