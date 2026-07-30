package acc

import (
	_ "github.com/lib/pq"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

// TestPublicationInAnotherOwnersDatabase covers creating a publication in a database the admin does not own.
//
// `CREATE PUBLICATION` checks the CREATE privilege on the current database, which `CREATE DATABASE` grants to
// the database's owner and nobody else. The admin is a CREATEROLE role that created the owner, not the owner
// itself, so it has to borrow the owner's privileges. A superuser never hits this.
func TestPublicationInAnotherOwnersDatabase(t *testing.T) {
	if os.Getenv("ACC") != "1" {
		t.Skip("Set ACC=1 to run e2e tests")
	}

	const (
		admin        = "pub-perm-admin"
		owner        = "pub-perm-owner"
		password     = "pub-perm-password"
		databaseName = "pub_perm_test"
		publication  = "pub_perm_publication"
	)

	superDb := openDb(t, "postgres")
	_, err := superDb.Exec(`DROP DATABASE IF EXISTS ` + databaseName + ` WITH (FORCE)`)
	require.NoError(t, err, "reset database")
	dropRole(t, superDb, owner)
	dropRole(t, superDb, admin)

	_, err = superDb.Exec(`CREATE ROLE "pub-perm-admin" LOGIN PASSWORD '` + password + `' CREATEROLE`)
	require.NoError(t, err, "create admin role")

	adminStore := createStoreAs(t, admin, password)
	defer adminStore.Close()

	// The admin creates the owner role, so it holds ADMIN OPTION on it but none of its privileges
	_, err = adminStore.Roles.Create(postgresql.Role{Name: owner, Password: password, UseExisting: true})
	require.NoError(t, err, "create owner role")

	// The database belongs to the owner, so the admin has no CREATE privilege on it
	_, err = superDb.Exec(`CREATE DATABASE ` + databaseName + ` OWNER "pub-perm-owner"`)
	require.NoError(t, err, "create database owned by the owner role")

	ownerDb := openDbAs(t, owner, password, databaseName)
	defer ownerDb.Close()
	_, err = ownerDb.Exec(`CREATE TABLE items ( id SERIAL NOT NULL )`)
	require.NoError(t, err, "create table owned by the owner role")

	key := postgresql.PublicationKey{Database: databaseName, Name: publication}
	// An explicit table list keeps this to privileges: `FOR ALL TABLES` and `FOR TABLES IN SCHEMA` require a
	// real superuser on stock postgres, which cloud sql grants to cloudsqlsuperuser members but this test has not.
	_, err = adminStore.Publications.Create(postgresql.Publication{
		Name:        key.Name,
		Database:    key.Database,
		Tables:      []postgresql.PublicationTable{{Schema: "public", Name: "items"}},
		UseExisting: true,
	})
	require.NoError(t, err, "create publication in a database owned by another role")

	find, err := adminStore.Publications.Read(key)
	require.NoError(t, err, "read publication")
	require.NotNil(t, find, "publication not found")
	assert.Equal(t, []postgresql.PublicationTable{{Schema: "public", Name: "items"}}, find.Tables)

	t.Run("re-creating adopts it", func(t *testing.T) {
		_, err := adminStore.Publications.Create(postgresql.Publication{
			Name:        key.Name,
			Database:    key.Database,
			Tables:      []postgresql.PublicationTable{{Schema: "public", Name: "items"}},
			UseExisting: true,
		})
		require.NoError(t, err, "re-create existing publication")
	})

	t.Run("releases the temporary membership", func(t *testing.T) {
		var stillInherits bool
		require.NoError(t, superDb.QueryRow(`SELECT pg_has_role($1, $2, 'USAGE')`, admin, owner).Scan(&stillInherits))
		assert.False(t, stillInherits, "the admin should not keep the owner's privileges after the statement")
	})
}
