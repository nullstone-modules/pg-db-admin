package workflows

import (
	"database/sql"
	"fmt"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"log"
)

func EnsureUser(connUrl string, newUser postgresql.Role) error {
	log.Printf("ensuring user %q\n", newUser.Name)

	db, err := sql.Open("postgres", connUrl)
	if err != nil {
		return fmt.Errorf("error connecting to postgres: %w", err)
	}
	defer db.Close()

	return newUser.Ensure(db)
}
