package postgresql

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
)

type Role struct {
	Name     string
	Password string
}

func (r Role) Create(conn *pgx.Conn) error {
	ctx := context.Background()
	sql := `SELECT 1 FROM pg_roles WHERE rolname = $1`
	row := conn.QueryRow(ctx, sql, r.Name)
	var result int
	if err := row.Scan(&result); err == pgx.ErrNoRows {
		fmt.Printf("creating role %q\n", r.Name)
		// Role does not exist yet, create
		nameIdentifier := pgx.Identifier{r.Name}
		sql := fmt.Sprintf(`CREATE USER %s WITH PASSWORD $1`, nameIdentifier.Sanitize())
		if _, err := conn.Exec(ctx, sql, r.Password); err != nil {
			return fmt.Errorf("error creating user %q: %w", r.Name, err)
		}
	} else if err != nil {
		return fmt.Errorf("error searching for existing role: %w", err)
	}
	return nil
}
