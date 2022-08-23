package workflows

import (
	"database/sql"
	"fmt"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"log"
)

func GrantDbAccess(db *sql.DB, appDb *sql.DB, user postgresql.Role, database postgresql.Database) error {
	log.Printf("Granting db access to user %q on database %q\n", user.Name, database.Name)

	databases := postgresql.Databases{Db: db}
	if _, err := databases.Read(database.Name); err != nil {
		return fmt.Errorf("unable to read database %q: %w", database.Name, err)
	}
	if err := grantRole(db, user, database); err != nil {
		return err
	}
	if err := postgresql.GrantDefaultPrivileges(appDb, user, database); err != nil {
		return err
	}
	return postgresql.GrantDbAndSchemaPrivileges(appDb, user, database)
}

// grantRole adds user as a member of the database owner role
func grantRole(db *sql.DB, user postgresql.Role, database postgresql.Database) error {
	log.Printf("Granting %q membership to %q", user.Name, database.Owner)
	roleGrants := postgresql.RoleGrants{Db: db}
	newRoleGrant := postgresql.RoleGrant{
		Id:     fmt.Sprintf("%s::%s", user.Name, database.Owner),
		Member: user.Name,
		Target: database.Owner,
	}
	if _, err := roleGrants.Ensure(newRoleGrant); err != nil {
		return fmt.Errorf("error granting role grant for %q to %q: %w", newRoleGrant.Member, newRoleGrant.Target, err)
	}
	return nil
}
