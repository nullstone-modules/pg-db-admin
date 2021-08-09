package workflows

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"log"
	"strings"
)

func GrantDbAccess(connUrl string, user postgresql.Role, database postgresql.Database) error {
	log.Printf("granting user %q db access to %q\n", user.Name, database.Name)

	if err := grantRole(connUrl, user, database); err != nil {
		return err
	}

	return grantAllPrivileges(connUrl, user, database)
}

func grantRole(connUrl string, user postgresql.Role, database postgresql.Database) error {
	db, err := sql.Open("postgres", connUrl)
	if err != nil {
		return fmt.Errorf("error connecting to postgres: %w", err)
	}
	defer db.Close()

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

func grantAllPrivileges(connUrl string, user postgresql.Role, database postgresql.Database) error {
	db, err := getAppDb(connUrl, database.Name)
	if err != nil {
		return fmt.Errorf("error connecting to postgres: %w", err)
	}
	defer db.Close()

	sq := strings.Join([]string{
		// CREATE | USAGE
		fmt.Sprintf(`GRANT ALL PRIVILEGES ON SCHEMA public TO %q;`, pq.QuoteIdentifier(user.Name)),
		// CREATE | CONNECT | TEMPORARY | TEMP
		fmt.Sprintf(`GRANT ALL PRIVILEGES ON DATABASE %q TO %q;`, pq.QuoteIdentifier(database.Name), pq.QuoteIdentifier(user.Name)),
	}, " ")

	if _, err := db.Exec(sq); err != nil {
		return fmt.Errorf("error granting privileges: %w", err)
	}
	return nil
}
