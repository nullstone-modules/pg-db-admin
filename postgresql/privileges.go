package postgresql

import (
	"database/sql"
	"fmt"
	"github.com/go-multierror/multierror"
	"github.com/lib/pq"
	"strings"
)

// GrantDbAndSchemaPrivileges grants user privileges to create schema and connect to the database/public schema
func GrantDbAndSchemaPrivileges(db *sql.DB, user Role, database Database) error {
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

// GrantDefaultPrivileges configures default privileges for any objects created by the user
//   This ensures that any objects created by user in the future will be accessible to the database owner role
//   Since grantRole adds role membership to database owner role, this effectively gives any new users access to objects
func GrantDefaultPrivileges(db *sql.DB, user Role, database Database) error {
	info, err := CalcDbConnectionInfo(db)
	if err != nil {
		return fmt.Errorf("error analyzing database: %w", err)
	}

	var grant Revoker = NoopRevoker{}
	var tempErr error
	if !info.IsSuperuser {
		grant, tempErr = GrantRoleMembership(db, user.Name, info.CurrentUser)
		// We only care about this error if the privilege sql didn't work down below
	}

	quotedUserName := pq.QuoteIdentifier(user.Name)
	quotedDbOwner := pq.QuoteIdentifier(database.Owner)

	sq := strings.Join([]string{
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s GRANT ALL PRIVILEGES ON TABLES TO %s;`, quotedUserName, quotedDbOwner),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s GRANT ALL PRIVILEGES ON SEQUENCES TO %s;`, quotedUserName, quotedDbOwner),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s GRANT ALL PRIVILEGES ON FUNCTIONS TO %s;`, quotedUserName, quotedDbOwner),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s GRANT ALL PRIVILEGES ON TYPES TO %s;`, quotedUserName, quotedDbOwner),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s GRANT ALL PRIVILEGES ON SCHEMAS TO %s;`, quotedUserName, quotedDbOwner),
	}, " ")
	errs := make([]error, 0)
	if _, err := db.Exec(sq); err != nil {
		if tempErr != nil {
			errs = append(errs, fmt.Errorf("error granting temporary membership: %w", tempErr))
		}
		errs = append(errs, fmt.Errorf("error altering default privileges: %w", err))
	}
	if grant != nil {
		if revokeErr := grant.Revoke(db); revokeErr != nil {
			errs = append(errs, fmt.Errorf("error revoking temporary membership: %w", revokeErr))
		}
	}
	if len(errs) > 0 {
		return multierror.New(errs)
	}
	return nil
}
