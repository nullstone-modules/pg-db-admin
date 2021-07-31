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
		return createDatabase(ctx, event.Metadata)
	case eventTypeCreateUser:
		return createUser(ctx, event.Metadata)
	default:
		return fmt.Errorf("unknown event %q", event.Type)
	}
}

func createDatabase(ctx context.Context, metadata map[string]string) error {
	newDatabase := postgresql.DefaultDatabase()
	newDatabase.Name, _ = metadata["databaseName"]
	if newDatabase.Name == "" {
		return fmt.Errorf("cannot create database: databaseName is required")
	}
	newDatabase.Owner, _ = metadata["owner"]
	if newDatabase.Owner == "" {
		return fmt.Errorf("cannot create database: owner is required")
	}

	connUrl, err := getConnectionUrl(ctx)
	if err != nil {
		return fmt.Errorf("error retrieving postgres connection url: %w", err)
	}

	db, err := sql.Open("postgres", connUrl)
	if err != nil {
		return fmt.Errorf("error connecting to postgres: %w", err)
	}
	defer db.Close()

	dbInfo, err := postgresql.CalcDbConnectionInfo(db)
	if err != nil {
		return fmt.Errorf("error introspecting postgres cluster: %w", err)
	}

	if err := newDatabase.Create(db, *dbInfo); err != nil {
		return fmt.Errorf("error creating database: %w", err)
	}
	return nil
}

func createUser(ctx context.Context, metadata map[string]string) error {
	newUser := postgresql.Role{}
	newUser.Name, _ = metadata["username"]
	if newUser.Name == "" {
		return fmt.Errorf("cannot create user: username is required")
	}
	newUser.Password, _ = metadata["password"]
	if newUser.Password == "" {
		return fmt.Errorf("cannot create user: password is required")
	}

	connUrl, err := getConnectionUrl(ctx)
	if err != nil {
		return fmt.Errorf("error retrieving postgres connection url: %w", err)
	}

	db, err := sql.Open("postgres", connUrl)
	if err != nil {
		return fmt.Errorf("error connecting to postgres: %w", err)
	}
	defer db.Close()

	if err := newUser.Create(db); err != nil {
		return fmt.Errorf("error creating user: %w", err)
	}
	return nil
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
