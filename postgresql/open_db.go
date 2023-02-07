package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"time"
)

var (
	dbOpenConnTimeout = 3 * time.Second
)

func OpenDatabase(connUrl string, databaseName string) (*sql.DB, error) {
	if databaseName != "" {
		u, err := url.Parse(connUrl)
		if err != nil {
			return nil, fmt.Errorf("invalid connection url %q: %w", connUrl, err)
		}
		u.Path = fmt.Sprintf("/%s", url.PathEscape(databaseName))
		log.Printf("Opening postgres connection to %s with user %q\n", u.Host, u.User.Username())
		connUrl = u.String()
	}

	db, err := sql.Open("postgres", connUrl)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbOpenConnTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("error establishing connection to postgres: %w", err)
	}
	log.Println("Postgres connection established")
	return db, nil
}
