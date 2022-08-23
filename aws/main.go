package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/nullstone-io/go-lambda-api-sdk/function_url"
	"github.com/nullstone-modules/pg-db-admin/api"
	"github.com/nullstone-modules/pg-db-admin/legacy"
	"log"
	"os"
)

const (
	dbConnUrlSecretIdEnvVar = "DB_CONN_URL_SECRET_ID"
)

func main() {
	dbConnUrl, err := fetchConnUrlFromSecrets(context.TODO())
	if err != nil {
		log.Println(err.Error())
	}
	lambda.Start(HandleRequest(dbConnUrl))
}

func HandleRequest(dbConnUrl string) func(ctx context.Context, rawEvent json.RawMessage) (any, error) {
	return func(ctx context.Context, rawEvent json.RawMessage) (any, error) {
		if ok, event := isFunctionUrlEvent(rawEvent); ok {
			router := api.CreateRouter(dbConnUrl)
			return function_url.Handle(ctx, event, router)
		}
		if ok, event := legacy.IsEvent(rawEvent); ok {
			return legacy.Handle(ctx, event, dbConnUrl)
		}
		return nil, nil
	}
}

func isFunctionUrlEvent(raw json.RawMessage) (bool, events.LambdaFunctionURLRequest) {
	var event events.LambdaFunctionURLRequest
	if err := json.Unmarshal(raw, &event); err != nil {
		return false, event
	}
	return event.RequestContext.HTTP.Method != "", event
}

func fetchConnUrlFromSecrets(ctx context.Context) (string, error) {
	awsConfig, err := config.LoadDefaultConfig(ctx)
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
