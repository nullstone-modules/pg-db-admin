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
	switch event.Type {
	case eventTypeCreateDatabase:
		return ensureDatabase(ctx, event.Metadata)
	case eventTypeCreateUser:
		return ensureUser(ctx, event.Metadata)
	case eventTypeCreateDbAccess:
		return grantUserDbAccess(ctx, event.Metadata)
	default:
		return fmt.Errorf("unknown event %q", event.Type)
	}
}

func ensureDatabase(ctx context.Context, metadata map[string]string) error {
	newDatabase := postgresql.DefaultDatabase()
	newDatabase.Name, _ = metadata["databaseName"]
	if newDatabase.Name == "" {
		return fmt.Errorf("cannot create database: databaseName is required")
	}

	db, err := getDb(ctx)
	if err != nil {
		return fmt.Errorf("error connecting to postgres: %w", err)
	}
	defer db.Close()

	dbInfo, err := postgresql.CalcDbConnectionInfo(db)
	if err != nil {
		return fmt.Errorf("error introspecting postgres cluster: %w", err)
	}

	// Create a role with the same name as the database to give ownership
	ownerRole := postgresql.Role{Name: newDatabase.Name}
	if err := ownerRole.Ensure(db); err != nil {
		return fmt.Errorf("error ensuring database owner role: %w", err)
	}
	newDatabase.Owner = ownerRole.Name

	return newDatabase.Ensure(db, *dbInfo)
}

func ensureUser(ctx context.Context, metadata map[string]string) error {
	newUser := postgresql.Role{}
	newUser.Name, _ = metadata["username"]
	if newUser.Name == "" {
		return fmt.Errorf("cannot create user: username is required")
	}
	newUser.Password, _ = metadata["password"]
	if newUser.Password == "" {
		return fmt.Errorf("cannot create user: password is required")
	}

	db, err := getDb(ctx)
	if err != nil {
		return fmt.Errorf("error connecting to postgres: %w", err)
	}
	defer db.Close()

	return newUser.Ensure(db)
}

func grantUserDbAccess(ctx context.Context, metadata map[string]string) error {
	user := postgresql.Role{}
	user.Name, _ = metadata["username"]
	if user.Name == "" {
		return fmt.Errorf("cannot grant user access to db: username is required")
	}
	database := postgresql.DefaultDatabase()
	database.Name, _ = metadata["databaseName"]
	if database.Name == "" {
		return fmt.Errorf("cannot grant user access to db: database name is required")
	}

	db, err := getDb(ctx)
	if err != nil {
		return fmt.Errorf("error connecting to postgres: %w", err)
	}
	defer db.Close()

	if err := database.Read(db); err != nil {
		return fmt.Errorf("unable to read database %q: %w", database.Name, err)
	}

	newRoleGrant := postgresql.RoleGrant{
		Member: user.Name,
		Target: database.Owner,
	}
	return newRoleGrant.Ensure(db)
}

func getDb(ctx context.Context) (*sql.DB, error) {
	connUrl, err := getConnectionUrl(ctx)
	if err != nil {
		return nil, fmt.Errorf("error retrieving postgres connection url: %w", err)
	}

	return sql.Open("postgres", connUrl)
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
