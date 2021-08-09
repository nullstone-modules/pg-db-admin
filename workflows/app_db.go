package workflows

import (
	"database/sql"
	"fmt"
	"net/url"
)

func getAppDb(connUrl string, databaseName string) (*sql.DB, error) {
	u, err := url.Parse(connUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid connection url %q: %w", connUrl, err)
	}
	u.Path = fmt.Sprintf("/%s", url.PathEscape(databaseName))

	return sql.Open("postgres", u.String())
}
