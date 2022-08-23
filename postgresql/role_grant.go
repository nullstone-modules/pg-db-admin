package postgresql

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.com/nullstone-io/go-rest-api"
	"log"
	"strings"
)

// RoleGrant adds Member to the Target role
type RoleGrant struct {
	Id string `json:"id"`

	// Member receives all the permissions for Target
	Member string `json:"member"`

	// Target is the role that gains an additional Member
	Target string `json:"target"`

	// WithAdminOption permits Member to grant it to others
	WithAdminOption bool `json:"withAdminOption"`
}

var _ rest.DataAccess[string, RoleGrant] = &RoleGrants{}

type RoleGrants struct {
	Db *sql.DB
}

func (r *RoleGrants) ParseKey(val string) (string, error) {
	return val, nil
}

func (r *RoleGrants) Read(key string) (*RoleGrant, error) {
	sq := `SELECT
pg_get_userbyid(member) as role,
	pg_get_userbyid(roleid) as grant_role,
	admin_option
FROM pg_auth_members
WHERE pg_get_userbyid(member) = $1 AND pg_get_userbyid(roleid) = $2`

	var grant RoleGrant
	grant.Id = key
	grant.Member, grant.Target, _ = strings.Cut(key, "::")

	row := r.Db.QueryRow(sq, grant.Member, grant.Target)
	if err := row.Scan(&grant.Member, &grant.Target, &grant.WithAdminOption); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &grant, nil
}

func (r *RoleGrants) Exists(grant RoleGrant) (bool, error) {
	existing, err := r.Read(grant.Id)
	return existing != nil, err
}

func (r *RoleGrants) Create(grant RoleGrant) (*RoleGrant, error) {
	sq := fmt.Sprintf("GRANT %s TO %s", pq.QuoteIdentifier(grant.Target), pq.QuoteIdentifier(grant.Member))
	if grant.WithAdminOption {
		sq = sq + " WITH ADMIN OPTION"
	}

	_, err := r.Db.Exec(sq)
	return &grant, err
}

func (r *RoleGrants) Ensure(grant RoleGrant) (*RoleGrant, error) {
	if exists, err := r.Exists(grant); exists {
		log.Printf("role grant %q => %q already exists\n", grant.Member, grant.Target)
		return &grant, nil
	} else if err != nil {
		return nil, fmt.Errorf("error checking for role grant %q => %q: %w", grant.Member, grant.Target, err)
	}
	return r.Create(grant)
}

func (r *RoleGrants) Update(key string, grant RoleGrant) (*RoleGrant, error) {
	return &grant, nil
}

func (r *RoleGrants) Drop(key string) (bool, error) {
	return true, nil
}
