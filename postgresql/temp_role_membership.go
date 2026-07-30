package postgresql

import (
	"database/sql"
	"fmt"
	"github.com/go-multierror/multierror"
	"github.com/lib/pq"
	"log"
	"strings"
)

// GrantRoleMembership grants role membership of the target 'role' to the current user
// This is used to perform commands if user is not a superuser
// For instance, when using AWS RDS, user is not given superuser
// It returns a nil Revoker if the grant is not needed because the user already has the role's privileges.
func GrantRoleMembership(db *sql.DB, role string, info *DbInfo) (Revoker, error) {
	currentUser := info.CurrentUser
	if currentUser == role {
		return nil, nil
	}

	hasPrivileges, err := hasPrivilegesOfRole(db, currentUser, role)
	if err != nil {
		return nil, err
	}
	if hasPrivileges {
		return nil, nil
	}

	log.Printf("Granting %q temporary access to role %q\n", currentUser, role)

	// Take a lock on db currentUser to avoid multiple database creation at the same time
	// It can fail if they grant the same owner to current at the same time as it's not done in transaction.
	lockTxn, err := db.Begin()
	if err := pgLockRole(lockTxn, currentUser); err != nil {
		return nil, err
	}

	if _, err := db.Exec(grantRoleSql(role, info)); err != nil {
		lockTxn.Rollback()
		return nil, fmt.Errorf("error granting role %s to %s: %w", role, currentUser, err)
	}
	return &TempGrant{
		Tx:          lockTxn,
		Role:        role,
		CurrentUser: currentUser,
	}, nil
}

// GrantRoleMemberships acquires the privileges of every role in 'roles' at once
//
// This takes the advisory lock once for the whole batch. Granting the roles one at a time would deadlock:
// each grant locks the same key (the current user), and the lock is held until its membership is revoked.
//
// Membership errors are returned alongside a Revoker for whatever was acquired: the caller usually only
// cares about them if the statement it is about to run fails.
func GrantRoleMemberships(db *sql.DB, roles []string, info *DbInfo) (Revoker, error) {
	needed := make([]string, 0, len(roles))
	for _, role := range roles {
		if role == info.CurrentUser {
			continue
		}
		hasPrivileges, err := hasPrivilegesOfRole(db, info.CurrentUser, role)
		if err != nil {
			return NoopRevoker{}, err
		}
		if !hasPrivileges {
			needed = append(needed, role)
		}
	}
	if len(needed) < 1 {
		return NoopRevoker{}, nil
	}

	log.Printf("Granting %q temporary access to roles [%s]\n", info.CurrentUser, strings.Join(needed, ", "))

	// Take a lock on the current user to avoid concurrent invocations granting the same memberships
	lockTxn, err := db.Begin()
	if err != nil {
		return NoopRevoker{}, fmt.Errorf("error starting transaction: %w", err)
	}
	if err := pgLockRole(lockTxn, info.CurrentUser); err != nil {
		lockTxn.Rollback()
		return NoopRevoker{}, err
	}

	grant := &TempGrants{Tx: lockTxn, CurrentUser: info.CurrentUser}
	errs := make([]error, 0)
	for _, role := range needed {
		if _, err := db.Exec(grantRoleSql(role, info)); err != nil {
			errs = append(errs, fmt.Errorf("error granting role %s to %s: %w", role, info.CurrentUser, err))
			continue
		}
		grant.Roles = append(grant.Roles, role)
	}
	if len(errs) > 0 {
		return grant, multierror.New(errs)
	}
	return grant, nil
}

// grantRoleSql grants role membership to the current user
//
// Postgres 16 grants a CREATEROLE user ADMIN OPTION on the roles it creates, but with INHERIT FALSE.
// That membership is not enough to act on the role's objects, so the inherit option has to be requested
// explicitly -- otherwise the grant is a no-op and the privilege statement that follows is denied.
func grantRoleSql(role string, info *DbInfo) string {
	sq := fmt.Sprintf("GRANT %s TO %s", pq.QuoteIdentifier(role), pq.QuoteIdentifier(info.CurrentUser))
	if info.SupportedFeatures.IsSupported(FeatureRoleInheritOption) {
		sq += " WITH INHERIT TRUE"
	}
	return sq
}

// Lock a role and all his members to avoid concurrent updates on some resources
func pgLockRole(txn *sql.Tx, role string) error {
	sql := `SELECT pg_advisory_xact_lock(oid::bigint) FROM pg_roles WHERE rolname = $1`
	if _, err := txn.Exec(sql, role); err != nil {
		return fmt.Errorf("could not get advisory lock for role %s: %w", role, err)
	}

	sql = `SELECT pg_advisory_xact_lock(member::bigint) FROM pg_auth_members JOIN pg_roles ON roleid = pg_roles.oid WHERE rolname = $1`
	if _, err := txn.Exec(sql, role); err != nil {
		return fmt.Errorf("could not get advisory lock for members of role %s: %w", role, err)
	}

	return nil
}

type Revoker interface {
	Revoke(db *sql.DB) error
}

type NoopRevoker struct {
}

func (t NoopRevoker) Revoke(db *sql.DB) error {
	return nil
}

// TempGrants releases a batch of temporary memberships acquired under a single advisory lock
type TempGrants struct {
	Tx          *sql.Tx
	Roles       []string
	CurrentUser string
}

// Revoke releases every membership in the batch, continuing past a failure so that one stuck role
// does not leave the rest of them granted
func (t TempGrants) Revoke(db *sql.DB) error {
	defer t.Tx.Rollback()

	errs := make([]error, 0)
	for _, role := range t.Roles {
		if role == t.CurrentUser {
			continue
		}
		hasPrivileges, err := hasPrivilegesOfRole(db, t.CurrentUser, role)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !hasPrivileges {
			continue
		}

		log.Printf("Revoking %q temporary access to role %q\n", t.CurrentUser, role)
		sq := fmt.Sprintf("REVOKE %s FROM %s", pq.QuoteIdentifier(role), pq.QuoteIdentifier(t.CurrentUser))
		if _, err := db.Exec(sq); err != nil {
			errs = append(errs, fmt.Errorf("error revoking role %s from %s: %w", role, t.CurrentUser, err))
		}
	}
	if len(errs) > 0 {
		return multierror.New(errs)
	}
	return nil
}

type TempGrant struct {
	Tx          *sql.Tx
	Role        string
	CurrentUser string
}

// Revoke revokes the role *role* from the user *member*.
// It returns false if the revoke is not needed because the user is not a member of this role.
func (t TempGrant) Revoke(db *sql.DB) error {
	defer t.Tx.Rollback()

	if t.CurrentUser == t.Role {
		return nil
	}

	hasPrivileges, err := hasPrivilegesOfRole(db, t.CurrentUser, t.Role)
	if err != nil {
		return err
	}
	if !hasPrivileges {
		return nil
	}

	log.Printf("Revoking %q temporary access to role %q\n", t.CurrentUser, t.Role)

	sql := fmt.Sprintf("REVOKE %s FROM %s", pq.QuoteIdentifier(t.Role), pq.QuoteIdentifier(t.CurrentUser))
	if _, err := db.Exec(sql); err != nil {
		return fmt.Errorf("error revoking role %s from %s: %w", t.Role, t.CurrentUser, err)
	}
	return nil
}

// hasPrivilegesOfRole reports whether member holds the privileges of role.
//
// This deliberately asks pg_has_role(..., 'USAGE') rather than looking for a row in pg_auth_members.
// Since postgres 16 a CREATEROLE user is automatically granted ADMIN OPTION on the roles it creates, but
// with INHERIT FALSE -- so a membership row exists while the member holds none of the role's privileges.
// Reading the row would report "already a member" and skip a grant that is actually required.
func hasPrivilegesOfRole(db *sql.DB, member, role string) (bool, error) {
	var hasPrivileges bool
	sq := `SELECT pg_has_role($1, $2, 'USAGE')`
	if err := db.QueryRow(sq, member, role).Scan(&hasPrivileges); err != nil {
		return false, fmt.Errorf("could not read role membership: %w", err)
	}
	return hasPrivileges, nil
}
