package workflows

import (
	"database/sql"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"log"
)

func EnsureUser(db *sql.DB, newUser postgresql.Role) error {
	log.Printf("ensuring user %q\n", newUser.Name)
	roles := postgresql.Roles{Db: db}
	_, err := roles.Ensure(newUser)
	return err
}
