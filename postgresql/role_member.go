package postgresql

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.com/nullstone-io/go-rest-api"
	"log"
	"strings"
)

// RoleMember adds Member to the Target role
type RoleMember struct {
	// Member receives all the permissions for Target
	Member string `json:"member"`

	// Target is the role that gains an additional Member
	Target string `json:"target"`

	// WithAdminOption permits Member to grant it to others
	WithAdminOption bool `json:"withAdminOption"`

	// UseExisting is accepted for compatibility and no longer changes anything: a membership is
	// ensured rather than created, so one that already confers Target's privileges is left alone
	// whether or not this is set.
	UseExisting bool `json:"useExisting"`
}

func (r RoleMember) Key() RoleMemberKey {
	return RoleMemberKey{
		Member: r.Member,
		Target: r.Target,
	}
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
	return r.ensure(membership)
}

// Read reports the membership row as postgres records it.
//
// A row is not proof that Member holds Target's privileges: since postgres 16 it can carry
// INHERIT FALSE and confer nothing. Create and Update do not rely on this; they ask pg_has_role.
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

// Update ensures the membership the new input describes.
//
// Terraform sends an update rather than a create whenever an invocation's input changes, so a
// capability that points an existing membership at a different target lands here. Treating it as
// a no-op left the new membership silently ungranted.
func (r *RoleMembers) Update(key RoleMemberKey, membership RoleMember) (*RoleMember, error) {
	return r.ensure(membership)
}

func (r *RoleMembers) Drop(key RoleMemberKey) (bool, error) {
	return true, nil
}

// ensure grants Target to Member unless Member already holds Target's privileges.
//
// Whether anything needs doing is decided by pg_has_role(..., 'USAGE'), not by a row in
// pg_auth_members. Since postgres 16 a CREATEROLE user is granted ADMIN OPTION on the roles it
// creates with INHERIT FALSE, so a row can exist while the member holds none of the role's
// privileges; the row would say "already a member" and skip a grant that is genuinely required.
//
// The grant asks for INHERIT TRUE explicitly where the server supports it. Postgres 16 defaults the
// option from the member's own INHERIT attribute, and a membership that does not inherit is not
// what any caller of this endpoint wants.
func (r *RoleMembers) ensure(membership RoleMember) (*RoleMember, error) {
	if membership.Member == "" || membership.Target == "" {
		return nil, fmt.Errorf("role membership requires both member and target")
	}

	db, err := r.DbOpener.OpenDatabase("")
	if err != nil {
		return nil, err
	}

	// ADMIN OPTION is not visible through pg_has_role, so a membership that asks for it is always
	// re-granted. GRANT of an existing membership is harmless: postgres reports a notice, and on 16+
	// updates the options of the grant.
	if !membership.WithAdminOption {
		has, err := hasPrivilegesOfRole(db, membership.Member, membership.Target)
		if err != nil {
			return nil, err
		}
		if has {
			log.Printf("Role membership (role=%s, member=%s) already confers privileges\n", membership.Target, membership.Member)
			return &membership, nil
		}
	}

	info, err := CalcDbConnectionInfo(db)
	if err != nil {
		return nil, fmt.Errorf("error analyzing database: %w", err)
	}

	sq := fmt.Sprintf("GRANT %s TO %s", pq.QuoteIdentifier(membership.Target), pq.QuoteIdentifier(membership.Member))
	options := make([]string, 0, 2)
	if membership.WithAdminOption {
		options = append(options, "ADMIN OPTION")
	}
	if info.SupportedFeatures.IsSupported(FeatureRoleInheritOption) {
		options = append(options, "INHERIT TRUE")
	}
	if len(options) > 0 {
		sq += " WITH " + strings.Join(options, ", ")
	}

	log.Printf("Granting role membership (role=%s, member=%s)\n", membership.Target, membership.Member)
	if _, err := db.Exec(sq); err != nil {
		return nil, fmt.Errorf("error granting role %q to %q: %w", membership.Target, membership.Member, err)
	}
	return &membership, nil
}
