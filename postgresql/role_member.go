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
}

type RoleMemberKey struct {
	Member string
	Target string
}

var _ rest.DataAccess[RoleMemberKey, RoleMember] = &RoleMembers{}

type RoleMembers struct {
	Db *sql.DB
}

func (r *RoleMembers) Read(key RoleMemberKey) (*RoleMember, error) {
	sq := `SELECT
pg_get_userbyid(member) as role,
	pg_get_userbyid(roleid) as grant_role,
	admin_option
FROM pg_auth_members
WHERE pg_get_userbyid(member) = $1 AND pg_get_userbyid(roleid) = $2`

	grant := RoleMember{
		Member: key.Member,
		Target: key.Target,
	}
	row := r.Db.QueryRow(sq, key.Member, key.Target)
	if err := row.Scan(&grant.Member, &grant.Target, &grant.WithAdminOption); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &grant, nil
}

func (r *RoleMembers) Exists(grant RoleMember) (bool, error) {
	existing, err := r.Read(RoleMemberKey{
		Member: grant.Member,
		Target: grant.Target,
	})
	return existing != nil, err
}

func (r *RoleMembers) Create(grant RoleMember) (*RoleMember, error) {
	sq := fmt.Sprintf("GRANT %s TO %s", pq.QuoteIdentifier(grant.Target), pq.QuoteIdentifier(grant.Member))
	if grant.WithAdminOption {
		sq = sq + " WITH ADMIN OPTION"
	}

	_, err := r.Db.Exec(sq)
	return &grant, err
}

func (r *RoleMembers) Ensure(grant RoleMember) (*RoleMember, error) {
	if exists, err := r.Exists(grant); exists {
		log.Printf("role grant %q => %q already exists\n", grant.Member, grant.Target)
		return &grant, nil
	} else if err != nil {
		return nil, fmt.Errorf("error checking for role grant %q => %q: %w", grant.Member, grant.Target, err)
	}
	return r.Create(grant)
}

func (r *RoleMembers) Update(key RoleMemberKey, grant RoleMember) (*RoleMember, error) {
	return &grant, nil
}

func (r *RoleMembers) Drop(key RoleMemberKey) (bool, error) {
	return true, nil
}
