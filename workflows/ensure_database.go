package workflows

import (
	"database/sql"
	"fmt"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"log"
)

func EnsureDatabase(connUrl string, newDatabase postgresql.Database) error {
	log.Printf("ensuring database %q\n", newDatabase.Name)

	db, err := sql.Open("postgres", connUrl)
	if err != nil {
		return fmt.Errorf("error connecting to postgres: %w", err)
	}
	defer db.Close()

	dbInfo, err := postgresql.CalcDbConnectionInfo(db)
	if err != nil {
		return fmt.Errorf("error introspecting postgres cluster: %w", err)
	}

	// Create a role with the same name as the database to give ownership
	if err := (postgresql.Role{Name: newDatabase.Name}).Ensure(db); err != nil {
		return fmt.Errorf("error ensuring database owner role: %w", err)
	}

	return newDatabase.Ensure(db, *dbInfo)
}
