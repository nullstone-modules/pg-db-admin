package legacy

import (
	"fmt"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"log"
)

func EnsureDatabase(store *postgresql.Store, newDatabase postgresql.Database) error {
	log.Printf("ensuring database %q\n", newDatabase.Name)

	// Create a ownerRole with the same name as the database to give ownership
	ownerRole := postgresql.Role{
		Name:        newDatabase.Owner,
		UseExisting: true,
	}
	if _, err := store.Roles.Create(ownerRole); err != nil {
		return fmt.Errorf("error ensuring database owner ownerRole: %w", err)
	}
	newDatabase.UseExisting = true
	_, err := store.Databases.Create(newDatabase)
	return err
}
