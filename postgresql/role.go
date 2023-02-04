package postgresql

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.com/nullstone-io/go-rest-api"
	"log"
)

type Role struct {
	Name     string `json:"name"`
	Password string `json:"password"`

	// Do not error if trying to create a role that already exists
	// Instead, read the existing, set the password, and return
	UseExisting bool `json:"useExisting"`
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
	if role.Password != "" {
		fmt.Fprint(b, " PASSWORD ")
		fmt.Fprint(b, pq.QuoteLiteral(role.Password))
	}
	return b.String()
}
