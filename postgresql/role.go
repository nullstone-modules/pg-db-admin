package postgresql

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.com/nullstone-io/go-rest-api"
	"log"
	"strings"
)

type Role struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	// Do not error if trying to create a role that already exists
	// Instead, read the existing, set the password, and return
	UseExisting bool `json:"useExisting"`
	// SkipPasswordUpdate informs Create to skip updating the role's password if the role already exists
	SkipPasswordUpdate bool `json:"-"`

	MemberOf   []string       `json:"memberOf"`
	Attributes RoleAttributes `json:"attributes"`
}

type RoleAttributes struct {
	CreateDb   bool `json:"createDb"`
	CreateRole bool `json:"createRole"`
}

var _ rest.DataAccess[string, Role] = &Roles{}

type Roles struct {
	DbOpener DbOpener
}

func (r *Roles) Create(role Role) (*Role, error) {
	if role.UseExisting {
		if existing, err := r.Read(role.Name); err != nil {
			return nil, err
		} else if existing != nil {
			log.Printf("[Create] Role %q already exists, updating...\n", role.Name)
			return r.Update(role.Name, role)
		}
	}

	db, err := r.DbOpener.OpenDatabase("")
	if err != nil {
		return nil, err
	}

	fmt.Printf("Creating role %q\n", role.Name)
	if _, err := db.Exec(r.generateCreateSql(role)); err != nil {
		return nil, fmt.Errorf("error creating user %q: %w", role.Name, err)
	}
	return &role, nil
}

func (r *Roles) Read(key string) (*Role, error) {
	db, err := r.DbOpener.OpenDatabase("")
	if err != nil {
		return nil, err
	}

	var name string
	row := db.QueryRow(`SELECT rolname from pg_roles WHERE rolname = $1`, key)
	if err := row.Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &Role{Name: name}, nil
}

func (r *Roles) Update(key string, role Role) (*Role, error) {
	db, err := r.DbOpener.OpenDatabase("")
	if err != nil {
		return nil, err
	}

	if role.Password == "" {
		return &role, nil
	}
	if role.SkipPasswordUpdate {
		log.Printf("Skipping password update for %q\n", role.Name)
		role.Password = ""
		return &role, nil
	}
	log.Printf("Setting password for %q\n", role.Name)
	updateSql := fmt.Sprintf(`ALTER ROLE %s WITH PASSWORD %s`, pq.QuoteIdentifier(role.Name), pq.QuoteLiteral(role.Password))
	if _, err := db.Exec(updateSql); err != nil {
		return nil, fmt.Errorf("error setting password: %w", err)
	}
	log.Printf("Password set for %q\n", role.Name)
	return &role, nil
}

func (r *Roles) Drop(key string) (bool, error) {
	return true, nil
}

func (*Roles) generateCreateSql(role Role) string {
	b := bytes.NewBufferString("CREATE ROLE ")
	fmt.Fprint(b, pq.QuoteIdentifier(role.Name), " WITH LOGIN")
	if role.Attributes.CreateRole {
		fmt.Fprint(b, " CREATEROLE")
	}
	if role.Attributes.CreateDb {
		fmt.Fprint(b, " CREATEDB")
	}
	if len(role.MemberOf) > 0 {
		safeRoleNames := make([]string, 0)
		for _, m := range role.MemberOf {
			safeRoleNames = append(safeRoleNames, pq.QuoteIdentifier(m))
		}
		fmt.Fprintf(b, "IN ROLE %s", strings.Join(safeRoleNames, ","))
	}
	if role.Password != "" {
		fmt.Fprint(b, " PASSWORD ")
		fmt.Fprint(b, pq.QuoteLiteral(role.Password))
	}
	return b.String()
}
