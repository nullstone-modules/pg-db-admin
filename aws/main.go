package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/nullstone-modules/pg-db-admin/workflows"
	"net/url"
	"os"
)

const (
	dbConnUrlSecretIdEnvVar = "DB_CONN_URL_SECRET_ID"

	eventTypeCreateDatabase = "create-database"
	eventTypeCreateUser     = "create-user"
	eventTypeCreateDbAccess = "create-db-access"
)

type AdminEvent struct {
	Type     string            `json:"type"`
	Metadata map[string]string `json:"metadata"`
}

func main() {
	lambda.Start(HandleRequest)
}

func HandleRequest(ctx context.Context, event AdminEvent) error {
	connUrl, err := getConnectionUrl(ctx)
	if err != nil {
		return err
	}

	db, err := sql.Open("postgres", connUrl)
	if err != nil {
		return fmt.Errorf("error connecting to db: %w", err)
	}
	defer db.Close()

	switch event.Type {
	case eventTypeCreateDatabase:
		newDatabase := postgresql.Database{}
		newDatabase.Name, _ = event.Metadata["databaseName"]
		if newDatabase.Name == "" {
			return fmt.Errorf("cannot create database: databaseName is required")
		}
		newDatabase.Owner = newDatabase.Name
		return workflows.EnsureDatabase(db, newDatabase)
	case eventTypeCreateUser:
		newUser := postgresql.Role{}
		newUser.Name, _ = event.Metadata["username"]
		if newUser.Name == "" {
			return fmt.Errorf("cannot create user: username is required")
		}
		newUser.Password, _ = event.Metadata["password"]
		if newUser.Password == "" {
			return fmt.Errorf("cannot create user: password is required")
		}
		return workflows.EnsureUser(db, newUser)
	case eventTypeCreateDbAccess:
		user := postgresql.Role{}
		user.Name, _ = event.Metadata["username"]
		if user.Name == "" {
			return fmt.Errorf("cannot grant user access to db: username is required")
		}
		database := postgresql.Database{}
		database.Name, _ = event.Metadata["databaseName"]
		if database.Name == "" {
			return fmt.Errorf("cannot grant user access to db: database name is required")
		}

		appDb, err := getAppDb(connUrl, database.Name)
		if err != nil {
			return fmt.Errorf("error connecting to app db %q: %w", database.Name, err)
		}

		return workflows.GrantDbAccess(db, appDb, user, database)
	default:
		return fmt.Errorf("unknown event %q", event.Type)
	}
}

func getAppDb(connUrl string, databaseName string) (*sql.DB, error) {
	u, err := url.Parse(connUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid connection url %q: %w", connUrl, err)
	}
	u.Path = fmt.Sprintf("/%s", url.PathEscape(databaseName))

	return sql.Open("postgres", u.String())
}

func getConnectionUrl(ctx context.Context) (string, error) {
	awsConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", fmt.Errorf("error accessing aws: %w", err)
	}
	sm := secretsmanager.NewFromConfig(awsConfig)
	secretId := os.Getenv(dbConnUrlSecretIdEnvVar)
	out, err := sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(secretId)})
	if err != nil {
		return "", fmt.Errorf("error accessing secret: %w", err)
	}
	if out.SecretString == nil {
		return "", nil
	}
	return *out.SecretString, nil
}
