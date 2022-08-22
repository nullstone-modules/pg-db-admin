package postgresql

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"log"
)

type Role struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func (r Role) SetId(val string) {
	r.Name = val
}

func (r Role) Ensure(db *sql.DB) error {
	if exists, err := r.Exists(db); exists {
		log.Printf("Role %q already exists\n", r.Name)
		if r.Password != "" {
			log.Printf("Setting password for %q\n", r.Name)
			if err := r.setPassword(db); err != nil {
				return err
			}
			log.Printf("Password set for %q\n", r.Name)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("error checking for role %q: %w", r.Name, err)
	}
	if err := r.Create(db); err != nil {
		return fmt.Errorf("error creating role %q: %w", r.Name, err)
	}
	return nil
}

func (r Role) Create(db *sql.DB) error {
	fmt.Printf("Creating role %q\n", r.Name)
	sq := r.generateCreateSql()
	if _, err := db.Exec(sq); err != nil {
		return fmt.Errorf("error creating user %q: %w", r.Name, err)
	}
	return nil
}

func (r Role) generateCreateSql() string {
	b := bytes.NewBufferString("CREATE ROLE ")
	fmt.Fprint(b, pq.QuoteIdentifier(r.Name), " WITH LOGIN")
	if r.Password != "" {
		fmt.Fprint(b, " PASSWORD ")
		fmt.Fprint(b, pq.QuoteLiteral(r.Password))
	}
	return b.String()
}

func (r Role) Exists(db *sql.DB) (bool, error) {
	check := Role{Name: r.Name}
	if err := check.Read(db); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *Role) Read(db *sql.DB) error {
	var name string
	row := db.QueryRow(`SELECT rolname from pg_roles WHERE rolname = $1`, r.Name)
	if err := row.Scan(&name); err != nil {
		return err
	}
	return nil
}

func (r Role) Update(db *sql.DB) error {
	return nil
}

func (r Role) setPassword(db *sql.DB) error {
	_, err := db.Exec(fmt.Sprintf(`ALTER ROLE %s WITH PASSWORD %s`, pq.QuoteIdentifier(r.Name), pq.QuoteLiteral(r.Password)))
	if err != nil {
		return fmt.Errorf("error setting password: %w", err)
	}
	return nil
}

func (r Role) Drop(db *sql.DB) error {
	return nil
}
