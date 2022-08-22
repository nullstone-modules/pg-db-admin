package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"net/url"
	"os"
)

const (
	dbConnUrlSecretIdEnvVar = "DB_CONN_URL_SECRET_ID"
)

type DbBroker struct {
	connectionUrl string
}

func NewDbBroker(ctx context.Context) (*DbBroker, error) {
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error accessing aws: %w", err)
	}
	sm := secretsmanager.NewFromConfig(awsConfig)
	secretId := os.Getenv(dbConnUrlSecretIdEnvVar)
	out, err := sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(secretId)})
	if err != nil {
		return nil, fmt.Errorf("error accessing secret: %w", err)
	}
	if out.SecretString == nil {
		return nil, nil
	}
	return &DbBroker{connectionUrl: *out.SecretString}, nil
}

func (b *DbBroker) ConnectionUrl() string {
	return b.connectionUrl
}

func (b *DbBroker) GetAppDb(databaseName string) (*sql.DB, error) {
	u, err := url.Parse(b.connectionUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid connection url %q: %w", b.connectionUrl, err)
	}
	u.Path = fmt.Sprintf("/%s", url.PathEscape(databaseName))

	return sql.Open("postgres", u.String())
}
