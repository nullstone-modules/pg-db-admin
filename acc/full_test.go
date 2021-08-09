package acc

import (
	"database/sql"
	_ "github.com/lib/pq"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/nullstone-modules/pg-db-admin/workflows"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/url"
	"strings"
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

	// Attempt connection to newly-created app db
	u, _ := url.Parse(connUrl)
	u.Path = "/test-database"
	u.User = url.UserPassword(newUser.Name, newUser.Password)
	appDb, err := sql.Open("postgres", u.String())
	require.NoError(t, err, "connect to app db")
	defer appDb.Close()

	// Attempt to create schema objects
	_, err = appDb.Exec("CREATE TABLE todos ( id SERIAL NOT NULL, name varchar(255) );")
	require.NoError(t, err, "create table")

	// Attempt to insert records
	sq := strings.Join([]string{
		`INSERT INTO todos (name) VALUES ('item1');`,
		`INSERT INTO todos (name) VALUES ('item2');`,
		`INSERT INTO todos (name) VALUES ('item3');`,
	}, "")
	_, err = appDb.Exec(sq)
	require.NoError(t, err, "insert todos")

	// Attempt to retrieve them
	results := make([]string, 0)
	rows, err := appDb.Query(`SELECT * FROM todos`)
	require.NoError(t, err, "query todos")
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		require.NoError(t, rows.Scan(&id, &name), "scan record")
		results = append(results, name)
	}
	assert.Equal(t, []string{"item1", "item2", "item3"}, results)
}
