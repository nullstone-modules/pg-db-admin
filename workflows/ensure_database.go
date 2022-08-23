package workflows

import (
	"database/sql"
	"fmt"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"log"
)

func EnsureDatabase(db *sql.DB, newDatabase postgresql.Database) error {
	log.Printf("ensuring database %q\n", newDatabase.Name)

	// Create a role with the same name as the database to give ownership
	roles := postgresql.Roles{Db: db}
	if _, err := roles.Ensure(postgresql.Role{Name: newDatabase.Name}); err != nil {
		return fmt.Errorf("error ensuring database owner role: %w", err)
	}
	databases := postgresql.Databases{Db: db}
	_, err := databases.Ensure(newDatabase)
	return err
}
