package legacy

import (
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"log"
)

func EnsureUser(store *postgresql.Store, newUser postgresql.Role) error {
	log.Printf("ensuring user %q\n", newUser.Name)
	newUser.UseExisting = true
	_, err := store.Roles.Create(newUser)
	return err
}
