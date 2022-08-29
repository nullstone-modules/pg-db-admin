package postgresql

import (
	"database/sql"
	"fmt"
	"net/url"
)

func OpenDatabase(connUrl string, databaseName string) (*sql.DB, error) {
	if databaseName != "" {
		u, err := url.Parse(connUrl)
		if err != nil {
			return nil, fmt.Errorf("invalid connection url %q: %w", connUrl, err)
		}
		u.Path = fmt.Sprintf("/%s", url.PathEscape(databaseName))
		connUrl = u.String()
	}

	return sql.Open("postgres", connUrl)
}
