package postgresql

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"log"
)

// RoleGrant adds Member to the Target role
type RoleGrant struct {
	// Member receives all the permissions for Target
	Member string

	// Target is the role that gains an additional Member
	Target string

	// WithAdminOption permits Member to grant it to others
	WithAdminOption bool
}

func (g RoleGrant) Ensure(db *sql.DB) error {
	if exists, err := g.Exists(db); exists {
		log.Printf("role grant %q => %q already exists\n", g.Member, g.Target)
		return nil
	} else if err != nil {
		return fmt.Errorf("error checking for role grant %q => %q: %w", g.Member, g.Target, err)
	}
	if err := g.Create(db); err != nil {
		return fmt.Errorf("error creating role grant %q => %q: %w", g.Member, g.Target, err)
	}
	return nil
}

func (g RoleGrant) Create(db *sql.DB) error {
	sq := fmt.Sprintf("GRANT %s TO %s", pq.QuoteIdentifier(g.Target), pq.QuoteIdentifier(g.Member))
	if g.WithAdminOption {
		sq = sq + " WITH ADMIN OPTION"
	}

	_, err := db.Exec(sq)
	return err
}

func (g RoleGrant) Exists(db *sql.DB) (bool, error) {
	check := RoleGrant{Member: g.Member, Target: g.Target}
	if err := check.Read(db); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (g RoleGrant) Read(db *sql.DB) error {
	sq := `SELECT
pg_get_userbyid(member) as role,
	pg_get_userbyid(roleid) as grant_role,
	admin_option
FROM pg_auth_members
WHERE pg_get_userbyid(member) = $1 AND pg_get_userbyid(roleid) = $2`

	var member string
	var target string
	var withAdminOption bool
	row := db.QueryRow(sq, g.Member, g.Target)
	if err := row.Scan(&member, &target, &withAdminOption); err != nil {
		return err
	}
	g.Member = member
	g.Target = target
	g.WithAdminOption = withAdminOption
	return nil
}
