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
}

var _ rest.DataAccess[string, Role] = &Roles{}

type Roles struct {
	BaseConnectionUrl string
}

func (r *Roles) Read(key string) (*Role, error) {
	db, err := OpenDatabase(r.BaseConnectionUrl, "")
	if err != nil {
		return nil, err
	}
	defer db.Close()

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

func (r *Roles) Exists(role Role) (bool, error) {
	existing, err := r.Read(role.Name)
	return existing != nil, err
}

func (r *Roles) Create(role Role) (*Role, error) {
	db, err := OpenDatabase(r.BaseConnectionUrl, "")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	fmt.Printf("Creating role %q\n", role.Name)
	sq := r.generateCreateSql(role)
	if _, err := db.Exec(sq); err != nil {
		return nil, fmt.Errorf("error creating user %q: %w", role.Name, err)
	}
	return &role, nil
}

func (r *Roles) Ensure(role Role) (*Role, error) {
	db, err := OpenDatabase(r.BaseConnectionUrl, "")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if exists, err := r.Exists(role); exists {
		log.Printf("Role %q already exists\n", role.Name)
		if role.Password != "" {
			log.Printf("Setting password for %q\n", role.Name)
			if err := r.setPassword(db, role); err != nil {
				return nil, err
			}
			log.Printf("Password set for %q\n", role.Name)
		}
		return &role, nil
	} else if err != nil {
		return nil, fmt.Errorf("error checking for role %q: %w", role.Name, err)
	}
	return r.Create(role)
}

func (r *Roles) Update(key string, role Role) (*Role, error) {
	db, err := OpenDatabase(r.BaseConnectionUrl, "")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if role.Password == "" {
		return &role, nil
	}
	return &role, r.setPassword(db, role)
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

func (r *Roles) setPassword(db *sql.DB, role Role) error {
	_, err := db.Exec(fmt.Sprintf(`ALTER ROLE %s WITH PASSWORD %s`, pq.QuoteIdentifier(role.Name), pq.QuoteLiteral(role.Password)))
	if err != nil {
		return fmt.Errorf("error setting password: %w", err)
	}
	return nil
}
