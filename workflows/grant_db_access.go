package workflows

import (
	"database/sql"
	"fmt"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"log"
)

func GrantDbAccess(db *sql.DB, appDb *sql.DB, user postgresql.Role, database postgresql.Database) error {
	log.Printf("Granting db access to user %q on database %q\n", user.Name, database.Name)

	dbInfo, err := postgresql.CalcDbConnectionInfo(db)
	if err != nil {
		return fmt.Errorf("error introspecting postgres: %w", err)
	}
	if err := database.Read(db); err != nil {
		return fmt.Errorf("unable to read database %q: %w", database.Name, err)
	}

	if err := grantRole(db, user, database); err != nil {
		return err
	}
	if err := postgresql.GrantDefaultPrivileges(dbInfo, appDb, user, database); err != nil {
		return err
	}
	return postgresql.GrantDbAndSchemaPrivileges(appDb, user, database)
}

// grantRole adds user as a member of the database owner role
func grantRole(db *sql.DB, user postgresql.Role, database postgresql.Database) error {
	log.Printf("Granting %q membership to %q", user.Name, database.Owner)
	newRoleGrant := postgresql.RoleGrant{
		Member: user.Name,
		Target: database.Owner,
	}
	if err := newRoleGrant.Ensure(db); err != nil {
		return fmt.Errorf("error granting role grant for %q to %q: %w", newRoleGrant.Member, newRoleGrant.Target, err)
	}
	return nil
}
