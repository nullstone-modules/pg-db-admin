package postgresql

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
)

type Role struct {
	Name     string
	Password string
}

func (r Role) Create(db *sql.DB) error {
	sq := `SELECT 1 FROM pg_roles WHERE rolname = $1`
	row := db.QueryRow(sq, r.Name)
	var result int
	if err := row.Scan(&result); err == sql.ErrNoRows {
		fmt.Printf("creating role %q\n", r.Name)
		// Role does not exist yet, create
		sql := fmt.Sprintf(`CREATE USER %s WITH PASSWORD %s`, pq.QuoteIdentifier(r.Name), pq.QuoteLiteral(r.Password))
		if _, err := db.Exec(sql); err != nil {
			return fmt.Errorf("error creating user %q: %w", r.Name, err)
		}
	} else if err != nil {
		return fmt.Errorf("error searching for existing role: %w", err)
	}
	return nil
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

func (r Role) Read(db *sql.DB) error {
	var name string
	row := db.QueryRow( `SELECT rolname from pg_roles WHERE rolname = $1`, r.Name)
	if err := row.Scan(&name); err != nil {
		return err
	}
	return nil
}
