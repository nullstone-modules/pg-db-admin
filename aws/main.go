package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/nullstone-io/go-lambda-api-sdk/function_url"
	"github.com/nullstone-modules/pg-db-admin/api"
	"github.com/nullstone-modules/pg-db-admin/aws/secrets"
	"github.com/nullstone-modules/pg-db-admin/legacy"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"log"
	"os"
	"time"
)

const (
	dbConnUrlSecretIdEnvVar = "DB_CONN_URL_SECRET_ID"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	dbConnUrl, err := secrets.GetString(ctx, os.Getenv(dbConnUrlSecretIdEnvVar))
	if err != nil {
		log.Println(err.Error())
	}
	store := postgresql.NewStore(dbConnUrl)
	defer store.Close()
	lambda.Start(HandleRequest(store))
}

func HandleRequest(store *postgresql.Store) func(ctx context.Context, rawEvent json.RawMessage) (any, error) {
	return func(ctx context.Context, rawEvent json.RawMessage) (any, error) {
		if ok, event := isFunctionUrlEvent(rawEvent); ok {
			router := api.CreateRouter(store)
			log.Println("Function URL Event", event.RequestContext.HTTP.Method, event.RequestContext.HTTP.Path)
			res, err := function_url.Handle(ctx, event, router)
			log.Println("Function URL Response", res.StatusCode)
			return res, err
		}
		if ok, event := legacy.IsEvent(rawEvent); ok {
			log.Println("Legacy Event", event.Type)
			return legacy.Handle(ctx, event, store)
		}
		log.Println("Unknown Event", string(rawEvent))
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
