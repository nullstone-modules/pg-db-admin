package postgresql

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
)

// TempRoleMembership grants role membership of the target Role to the CurrentUser
// This is used to perform commands if user is not a superuser
// For instance, when using AWS RDS, user is not given superuser
type TempRoleMembership struct {
	Role        string
	CurrentUser string
}

// Grant grants the role *role* to the user *member*.
// It returns false if the grant is not needed because the user is already
// a member of this role.
func (t TempRoleMembership) Grant(conn *pgx.Conn) (*TempGrant, error) {
	if t.CurrentUser == t.Role {
		return nil, nil
	}

	ctx := context.Background()

	isMember, err := isMemberOfRole(conn, t.CurrentUser, t.Role)
	if err != nil {
		return nil, err
	}
	if isMember {
		return nil, nil
	}

	fmt.Printf("Granting %q temporary access to role %q\n", t.CurrentUser, t.Role)

	// Take a lock on db currentUser to avoid multiple database creation at the same time
	// It can fail if they grant the same owner to current at the same time as it's not done in transaction.
	lockTxn, err := conn.Begin(ctx)
	if err := t.pgLockRole(lockTxn, t.CurrentUser); err != nil {
		return nil, err
	}

	roleIdentifier := pgx.Identifier{t.Role}
	curUserIdentifier := pgx.Identifier{t.CurrentUser}
	sql := fmt.Sprintf("GRANT %s TO %s", roleIdentifier.Sanitize(), curUserIdentifier.Sanitize())
	if _, err := conn.Exec(ctx, sql); err != nil {
		lockTxn.Rollback(ctx)
		return nil, fmt.Errorf("error granting role %s to %s: %w", t.Role, t.CurrentUser, err)
	}
	return &TempGrant{
		Tx:          lockTxn,
		Role:        t.Role,
		CurrentUser: t.CurrentUser,
	}, nil
}

// Lock a role and all his members to avoid concurrent updates on some resources
func (t TempRoleMembership) pgLockRole(txn pgx.Tx, role string) error {
	sql := `SELECT pg_advisory_xact_lock(oid::bigint) FROM pg_roles WHERE rolname = $1`
	if _, err := txn.Exec(context.Background(), sql, role); err != nil {
		return fmt.Errorf("could not get advisory lock for role %s: %w", role, err)
	}

	sql = `SELECT pg_advisory_xact_lock(member::bigint) FROM pg_auth_members JOIN pg_roles ON roleid = pg_roles.oid WHERE rolname = $1`
	if _, err := txn.Exec(context.Background(), sql, role); err != nil {
		return fmt.Errorf("could not get advisory lock for members of role %s: %w", role, err)
	}

	return nil
}

type TempGrant struct {
	Tx          pgx.Tx
	Role        string
	CurrentUser string
}

// Revoke revokes the role *role* from the user *member*.
// It returns false if the revoke is not needed because the user is not a member of this role.
func (t TempGrant) Revoke(conn *pgx.Conn) error {
	defer t.Tx.Rollback(context.Background())

	if t.CurrentUser == t.Role {
		return nil
	}

	isMember, err := isMemberOfRole(conn, t.CurrentUser, t.Role)
	if err != nil {
		return err
	}
	if !isMember {
		return nil
	}

	fmt.Printf("Revoking %q temporary access to role %q\n", t.CurrentUser, t.Role)

	roleIdentifier := pgx.Identifier{t.Role}
	memberIdentifier := pgx.Identifier{t.CurrentUser}
	sql := fmt.Sprintf("REVOKE %s FROM %s", roleIdentifier.Sanitize(), memberIdentifier.Sanitize())
	if _, err := conn.Exec(context.Background(), sql); err != nil {
		return fmt.Errorf("error revoking role %s from %s: %w", t.Role, t.CurrentUser, err)
	}
	return nil
}

func isMemberOfRole(conn *pgx.Conn, member, role string) (bool, error) {
	var noval int
	sql := `SELECT 1 FROM pg_auth_members WHERE pg_get_userbyid(roleid) = $1 AND pg_get_userbyid(member) = $2`
	err := conn.QueryRow(context.Background(), sql, role, member).Scan(&noval)

	switch {
	case err == pgx.ErrNoRows:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("could not read role membership: %w", err)
	}

	return true, nil
}
