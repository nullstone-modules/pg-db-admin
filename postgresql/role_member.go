package postgresql

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.com/nullstone-io/go-rest-api"
	"log"
)

// RoleMember adds Member to the Target role
type RoleMember struct {
	// Member receives all the permissions for Target
	Member string `json:"member"`

	// Target is the role that gains an additional Member
	Target string `json:"target"`

	// WithAdminOption permits Member to grant it to others
	WithAdminOption bool `json:"withAdminOption"`

	// Do not error if trying to create a role membership that already exists
	// Instead, return the existing
	UseExisting bool `json:"useExisting"`
}

type RoleMemberKey struct {
	Member string
	Target string
}

var _ rest.DataAccess[RoleMemberKey, RoleMember] = &RoleMembers{}

type RoleMembers struct {
	DbOpener DbOpener
}

func (r *RoleMembers) Create(membership RoleMember) (*RoleMember, error) {
	if membership.UseExisting {
		key := RoleMemberKey{
			Member: membership.Member,
			Target: membership.Target,
		}
		if existing, err := r.Read(key); err != nil {
			return nil, err
		} else if existing != nil {
			log.Printf("[Create] Role membership (role=%s, member=%s) already exists\n", key.Target, key.Member)
			return existing, nil
		}
	}

	db, err := r.DbOpener.OpenDatabase("")
	if err != nil {
		return nil, err
	}

	sq := fmt.Sprintf("GRANT %s TO %s", pq.QuoteIdentifier(membership.Target), pq.QuoteIdentifier(membership.Member))
	if membership.WithAdminOption {
		sq = sq + " WITH ADMIN OPTION"
	}

	log.Printf("Creating role membership (role=%s, member=%s)\n", membership.Target, membership.Member)
	_, err = db.Exec(sq)
	return &membership, err
}

func (r *RoleMembers) Read(key RoleMemberKey) (*RoleMember, error) {
	db, err := r.DbOpener.OpenDatabase("")
	if err != nil {
		return nil, err
	}

	sq := `SELECT
pg_get_userbyid(member) as role,
	pg_get_userbyid(roleid) as grant_role,
	admin_option
FROM pg_auth_members
WHERE pg_get_userbyid(member) = $1 AND pg_get_userbyid(roleid) = $2`

	membership := RoleMember{
		Member: key.Member,
		Target: key.Target,
	}
	row := db.QueryRow(sq, key.Member, key.Target)
	if err := row.Scan(&membership.Member, &membership.Target, &membership.WithAdminOption); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &membership, nil
}

func (r *RoleMembers) Update(key RoleMemberKey, membership RoleMember) (*RoleMember, error) {
	return &membership, nil
}

func (r *RoleMembers) Drop(key RoleMemberKey) (bool, error) {
	return true, nil
}
