package legacy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
)

const (
	eventTypeCreateDatabase = "create-database"
	eventTypeCreateUser     = "create-user"
	eventTypeCreateDbAccess = "create-db-access"
)

type AdminEvent struct {
	Type     string            `json:"type"`
	Metadata map[string]string `json:"metadata"`
}

func IsEvent(rawEvent json.RawMessage) (bool, AdminEvent) {
	var event AdminEvent
	if err := json.Unmarshal(rawEvent, &event); err != nil {
		return false, event
	}
	return event.Type != "", event
}

func Handle(ctx context.Context, event AdminEvent, dbConnUrl string) (any, error) {
	db, err := sql.Open("postgres", dbConnUrl)
	if err != nil {
		return nil, fmt.Errorf("error connecting to db: %w", err)
	}
	defer db.Close()
	store := postgresql.NewStore(db, dbConnUrl)

	switch event.Type {
	case eventTypeCreateDatabase:
		newDatabase := postgresql.Database{}
		newDatabase.Name, _ = event.Metadata["databaseName"]
		if newDatabase.Name == "" {
			return nil, fmt.Errorf("cannot create database: databaseName is required")
		}
		return nil, EnsureDatabase(store, newDatabase)
	case eventTypeCreateUser:
		newUser := postgresql.Role{}
		newUser.Name, _ = event.Metadata["username"]
		if newUser.Name == "" {
			return nil, fmt.Errorf("cannot create user: username is required")
		}
		newUser.Password, _ = event.Metadata["password"]
		if newUser.Password == "" {
			return nil, fmt.Errorf("cannot create user: password is required")
		}
		return nil, EnsureUser(store, newUser)
	case eventTypeCreateDbAccess:
		username, _ := event.Metadata["username"]
		if username == "" {
			return nil, fmt.Errorf("cannot grant user access to db: username is required")
		}
		databaseName, _ := event.Metadata["databaseName"]
		if databaseName == "" {
			return nil, fmt.Errorf("cannot grant user access to db: database name is required")
		}
		return nil, GrantDbAccess(store, username, databaseName)
	default:
		return nil, fmt.Errorf("unknown event %q", event.Type)
	}
}
