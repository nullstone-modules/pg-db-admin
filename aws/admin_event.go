package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/nullstone-modules/pg-db-admin/workflows"
)

type AdminEvent struct {
	Type     string            `json:"type"`
	Metadata map[string]string `json:"metadata"`
}

func handleAdminEvent(ctx context.Context, event AdminEvent, dbConnUrl string) (any, error) {
	db, err := sql.Open("postgres", dbConnUrl)
	if err != nil {
		return nil, fmt.Errorf("error connecting to db: %w", err)
	}
	defer db.Close()

	switch event.Type {
	case eventTypeCreateDatabase:
		newDatabase := postgresql.Database{}
		newDatabase.Name, _ = event.Metadata["databaseName"]
		if newDatabase.Name == "" {
			return nil, fmt.Errorf("cannot create database: databaseName is required")
		}
		newDatabase.Owner = newDatabase.Name
		return nil, workflows.EnsureDatabase(db, newDatabase)
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
		return nil, workflows.EnsureUser(db, newUser)
	case eventTypeCreateDbAccess:
		user := postgresql.Role{}
		user.Name, _ = event.Metadata["username"]
		if user.Name == "" {
			return nil, fmt.Errorf("cannot grant user access to db: username is required")
		}
		database := postgresql.Database{}
		database.Name, _ = event.Metadata["databaseName"]
		if database.Name == "" {
			return nil, fmt.Errorf("cannot grant user access to db: database name is required")
		}

		appDb, err := postgresql.OpenDatabase(dbConnUrl, database.Name)
		if err != nil {
			return nil, fmt.Errorf("error connecting to app db %q: %w", database.Name, err)
		}
		defer appDb.Close()

		return nil, workflows.GrantDbAccess(db, appDb, user, database)
	default:
		return nil, fmt.Errorf("unknown event %q", event.Type)
	}
}
