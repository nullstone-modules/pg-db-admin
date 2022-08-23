package acc

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/nullstone-modules/pg-db-admin/legacy"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// TestFull tests the entire workflow of create-database, create-user, create-db-access
func TestFull(t *testing.T) {
	if os.Getenv("ACC") != "1" {
		t.Skip("Set ACC=1 to run e2e tests")
	}

	newDatabase := postgresql.Database{
		Name:  "test-database",
		Owner: "test-database",
	}
	newUser := postgresql.Role{
		Name:     "test-user",
		Password: "test-password",
	}
	secondUser := postgresql.Role{
		Name:     "second-user",
		Password: "second-password",
	}

	connect := func(t *testing.T, database, user, password string) (*sql.DB, postgresql.Store) {
		u := url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(user, password),
			Host:     "localhost:8432",
			Path:     "/" + database,
			RawQuery: "sslmode=disable",
		}
		db, err := sql.Open("postgres", u.String())
		require.NoError(t, err, fmt.Sprintf("connecting to %q", database))
		return db, postgresql.NewStore(db, u.String())
	}

	ensureFull := func(t *testing.T, store postgresql.Store, database postgresql.Database, user postgresql.Role, testSuffix string) {
		_, err := store.Roles.Ensure(postgresql.Role{Name: database.Owner})
		require.NoErrorf(t, err, "ensure database owner role %s", testSuffix)
		_, err = store.Databases.Ensure(database)
		require.NoErrorf(t, err, "ensure database %s", testSuffix)
		_, err = store.Roles.Ensure(user)
		require.NoErrorf(t, err, "ensure user %s", testSuffix)
		require.NoErrorf(t, legacy.GrantDbAccess(store, user.Name, database.Name), "grant db access %s", testSuffix)
	}

	t.Run("initial setup", func(t *testing.T) {
		// This connection is used by the admin user from the `postgres` db
		rootDb, store := connect(t, "postgres", "pda", "pda")
		defer rootDb.Close()

		// Run through creation with first user
		ensureFull(t, store, newDatabase, newUser, "#1")
	})

	time.Sleep(500 * time.Millisecond)

	t.Run("connect with new user", func(t *testing.T) {
		db, _ := connect(t, newDatabase.Name, newUser.Name, newUser.Password)
		defer db.Close()
		require.NoError(t, db.Ping(), "connect to app db using newly created user")
	})

	t.Run("create schema using new user", func(t *testing.T) {
		db, _ := connect(t, newDatabase.Name, newUser.Name, newUser.Password)
		defer db.Close()

		// Attempt to create schema objects
		_, err := db.Exec("CREATE TABLE todos ( id SERIAL NOT NULL, name varchar(255) );")
		require.NoError(t, err, "create table")
	})

	t.Run("insert data using new user", func(t *testing.T) {
		db, _ := connect(t, newDatabase.Name, newUser.Name, newUser.Password)
		defer db.Close()

		// Attempt to insert records
		sq := strings.Join([]string{
			`INSERT INTO todos (name) VALUES ('item1');`,
			`INSERT INTO todos (name) VALUES ('item2');`,
			`INSERT INTO todos (name) VALUES ('item3');`,
		}, "")
		_, err := db.Exec(sq)
		require.NoError(t, err, "insert todos")
	})

	t.Run("retrieve data using new user", func(t *testing.T) {
		db, _ := connect(t, newDatabase.Name, newUser.Name, newUser.Password)
		defer db.Close()

		// Attempt to retrieve them
		results := make([]string, 0)
		rows, err := db.Query(`SELECT * FROM todos`)
		require.NoError(t, err, "query todos")
		defer rows.Close()
		for rows.Next() {
			var id int
			var name string
			require.NoError(t, rows.Scan(&id, &name), "scan record")
			results = append(results, name)
		}
		assert.Equal(t, []string{"item1", "item2", "item3"}, results)
	})

	t.Run("create second user after schema and data creation", func(t *testing.T) {
		// This connection is used by the admin user from the `postgres` db
		rootDb, store := connect(t, "postgres", "pda", "pda")
		defer rootDb.Close()

		ensureFull(t, store, newDatabase, secondUser, "#2")
	})

	t.Run("retrieve data from second user", func(t *testing.T) {
		db, _ := connect(t, newDatabase.Name, secondUser.Name, secondUser.Password)
		defer db.Close()

		// Attempt to retrieve them
		results := make([]string, 0)
		rows, err := db.Query(`SELECT * FROM todos`)
		require.NoError(t, err, "query todos")
		defer rows.Close()
		for rows.Next() {
			var id int
			var name string
			require.NoError(t, rows.Scan(&id, &name), "scan record")
			results = append(results, name)
		}
		assert.Equal(t, []string{"item1", "item2", "item3"}, results)
	})
}
