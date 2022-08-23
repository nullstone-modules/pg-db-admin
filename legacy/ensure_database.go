package legacy

import (
	"fmt"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"log"
)

func EnsureDatabase(store postgresql.Store, newDatabase postgresql.Database) error {
	log.Printf("ensuring database %q\n", newDatabase.Name)

	// Create a role with the same name as the database to give ownership
	if _, err := store.Roles.Ensure(postgresql.Role{Name: newDatabase.Owner}); err != nil {
		return fmt.Errorf("error ensuring database owner role: %w", err)
	}
	_, err := store.Databases.Ensure(newDatabase)
	return err
}
