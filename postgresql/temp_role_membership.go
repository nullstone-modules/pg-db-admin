package postgresql

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
)

// TempRoleMembership is used to perform commands if user is not a superuser
// For instance, when using AWS RDS, user is not given superuser
type TempRoleMembership struct {
	Role   string
	Member string
}

// Grant grants the role *role* to the user *member*.
// It returns false if the grant is not needed because the user is already
// a member of this role.
func (t TempRoleMembership) Grant(db *sql.DB, currentUser string) (*TempGrant, error) {
	if t.Member == t.Role {
		return nil, nil
	}
	isMember, err := isMemberOfRole(db, t.Member, t.Role)
	if err != nil {
		return nil, err
	}
	if isMember {
		return nil, nil
	}

	// Take a lock on db currentUser to avoid multiple database creation at the same time
	// It can fail if they grant the same owner to current at the same time as it's not done in transaction.
	lockTxn, err := db.Begin()
	if err := t.pgLockRole(lockTxn, currentUser); err != nil {
		return nil, err
	}

	sql := fmt.Sprintf("GRANT %s TO %s", pq.QuoteIdentifier(t.Role), pq.QuoteIdentifier(t.Member))
	if _, err := db.Exec(sql); err != nil {
		lockTxn.Rollback()
		return nil, fmt.Errorf("Error granting role %s to %s: %w", t.Role, t.Member, err)
	}
	return &TempGrant{
		Tx:     lockTxn,
		Role:   t.Role,
		Member: t.Member,
	}, nil
}

// Lock a role and all his members to avoid concurrent updates on some resources
func (t TempRoleMembership) pgLockRole(txn *sql.Tx, role string) error {
	if _, err := txn.Exec("SELECT pg_advisory_xact_lock(oid::bigint) FROM pg_roles WHERE rolname = $1", role); err != nil {
		return fmt.Errorf("could not get advisory lock for role %s: %w", role, err)
	}

	if _, err := txn.Exec(
		"SELECT pg_advisory_xact_lock(member::bigint) FROM pg_auth_members JOIN pg_roles ON roleid = pg_roles.oid WHERE rolname = $1",
		role,
	); err != nil {
		return fmt.Errorf("could not get advisory lock for members of role %s: %w", role, err)
	}

	return nil
}

type TempGrant struct {
	Tx     *sql.Tx
	Role   string
	Member string
}

// Revoke revokes the role *role* from the user *member*.
// It returns false if the revoke is not needed because the user is not a member of this role.
func (t TempGrant) Revoke(db *sql.DB) error {
	defer t.Tx.Rollback()

	if t.Member == t.Role {
		return nil
	}

	isMember, err := isMemberOfRole(db, t.Member, t.Role)
	if err != nil {
		return err
	}
	if !isMember {
		return nil
	}

	sql := fmt.Sprintf("REVOKE %s FROM %s", pq.QuoteIdentifier(t.Role), pq.QuoteIdentifier(t.Member))
	if _, err := db.Exec(sql); err != nil {
		return fmt.Errorf("Error revoking role %s from %s: %w", t.Role, t.Member, err)
	}
	return nil
}

func isMemberOfRole(db *sql.DB, member, role string) (bool, error) {
	var noval int
	err := db.QueryRow(
		"SELECT 1 FROM pg_auth_members WHERE pg_get_userbyid(roleid) = $1 AND pg_get_userbyid(member) = $2",
		role, member,
	).Scan(&noval)

	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("could not read role membership: %w", err)
	}

	return true, nil
}
