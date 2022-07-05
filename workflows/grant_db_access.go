package workflows

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"log"
	"strings"
)

func GrantDbAccess(db *sql.DB, appDb *sql.DB, user postgresql.Role, database postgresql.Database) error {
	log.Printf("Granting user %q db access to %q\n", user.Name, database.Name)
	if err := grantRole(db, user, database); err != nil {
		return err
	}
	if err := configureDefaultPrivileges(appDb, user, database); err != nil {
		return err
	}
	return grantAllPrivileges(appDb, user, database)
}

// grantRole adds user as a member of the database owner role
func grantRole(db *sql.DB, user postgresql.Role, database postgresql.Database) error {
	if err := database.Read(db); err != nil {
		return fmt.Errorf("unable to read database %q: %w", database.Name, err)
	}

	newRoleGrant := postgresql.RoleGrant{
		Member: user.Name,
		Target: database.Owner,
	}
	if err := newRoleGrant.Ensure(db); err != nil {
		return fmt.Errorf("error granting role grant for %q to %q: %w", newRoleGrant.Member, newRoleGrant.Target, err)
	}
	return nil
}

// grantAllPrivileges grants user privileges to create schema and connect to the database and public schema
func grantAllPrivileges(db *sql.DB, user postgresql.Role, database postgresql.Database) error {
	sq := strings.Join([]string{
		// CREATE | USAGE
		fmt.Sprintf(`GRANT ALL PRIVILEGES ON SCHEMA public TO %s;`, pq.QuoteIdentifier(user.Name)),
		// CREATE | CONNECT | TEMPORARY | TEMP
		fmt.Sprintf(`GRANT ALL PRIVILEGES ON DATABASE %s TO %s;`, pq.QuoteIdentifier(database.Name), pq.QuoteIdentifier(user.Name)),
	}, " ")

	if _, err := db.Exec(sq); err != nil {
		return fmt.Errorf("error granting privileges: %w", err)
	}
	return nil
}

// configureDefaultPrivileges configures default privileges for any objects created by the user
//   This ensures that any objects created by user in the future will be accessible to the database owner role
//   Since grantRole adds role membership to database owner role, this effectively gives any new users access to objects
func configureDefaultPrivileges(db *sql.DB, user postgresql.Role, database postgresql.Database) error {
	quotedUserName := pq.QuoteIdentifier(user.Name)
	quotedDbOwner := pq.QuoteIdentifier(database.Owner)

	sq := strings.Join([]string{
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s GRANT ALL PRIVILEGES ON TABLES TO %s;`, quotedUserName, quotedDbOwner),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s GRANT ALL PRIVILEGES ON SEQUENCES TO %s;`, quotedUserName, quotedDbOwner),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s GRANT ALL PRIVILEGES ON FUNCTIONS TO %s;`, quotedUserName, quotedDbOwner),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s GRANT ALL PRIVILEGES ON TYPES TO %s;`, quotedUserName, quotedDbOwner),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s GRANT ALL PRIVILEGES ON SCHEMAS TO %s;`, quotedUserName, quotedDbOwner),
	}, " ")
	if _, err := db.Exec(sq); err != nil {
		return fmt.Errorf("error altering default privileges: %w", err)
	}
	return nil
}
