package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/nullstone-io/go-lambda-api-sdk/function_url"
	"github.com/nullstone-modules/pg-db-admin/api"
	"github.com/nullstone-modules/pg-db-admin/aws/secrets"
	crud_invoke "github.com/nullstone-modules/pg-db-admin/crud-invoke"
	"github.com/nullstone-modules/pg-db-admin/legacy"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/nullstone-modules/pg-db-admin/setup"
	"log"
	"os"
	"time"
)

const (
	// dbSetupConnUrlSecretIdEnvVar is a secret id containing a connection url for running initial setup
	dbSetupConnUrlSecretIdEnvVar = "DB_SETUP_CONN_URL_SECRET_ID"
	// adminConnUrlSecretIdEnvVar is a secret id containing a connection url for performing db admin operations
	dbAdminConnUrlSecretIdEnvVar = "DB_ADMIN_CONN_URL_SECRET_ID"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var dbSetupConnUrl string
	if setupConnUrlSecretId := os.Getenv(dbSetupConnUrlSecretIdEnvVar); setupConnUrlSecretId == "" {
		log.Println("Skipping setup connection url secret")
	} else {
		log.Printf("Retrieving setup connection url secret (%s)\n", setupConnUrlSecretId)
		var err error
		dbSetupConnUrl, err = secrets.GetString(ctx, setupConnUrlSecretId)
		if err != nil {
			log.Println(err.Error())
		}
	}
	adminConnUrlSecretId := os.Getenv(dbAdminConnUrlSecretIdEnvVar)
	log.Printf("Retrieving admin connection url secret (%s)\n", adminConnUrlSecretId)
	dbAdminConnUrl, err := secrets.GetString(ctx, adminConnUrlSecretId)
	if err != nil {
		log.Println(err.Error())
	}

	setupStore := postgresql.NewStore(dbSetupConnUrl)
	defer setupStore.Close()
	adminStore := postgresql.NewStore(dbAdminConnUrl)
	defer adminStore.Close()

	lambda.Start(HandleRequest(setupStore, adminStore))
}

func HandleRequest(setupStore, adminStore *postgresql.Store) func(ctx context.Context, rawEvent json.RawMessage) (any, error) {
	return func(ctx context.Context, rawEvent json.RawMessage) (any, error) {
		if ok, event := setup.IsEvent(rawEvent); ok {
			log.Println("Initial Setup Event")
			return setup.Handle(ctx, event, setupStore, os.Getenv(dbAdminConnUrlSecretIdEnvVar))
		}
		if ok, event := crud_invoke.IsEvent(rawEvent); ok {
			log.Println("Invocation (CRUD) Event", event.Tf.Action, event.Type)
			return crud_invoke.Handle(ctx, event, adminStore)
		}

		if ok, event := isFunctionUrlEvent(rawEvent); ok {
			router := api.CreateRouter(adminStore)
			log.Println("Function URL Event", event.RequestContext.HTTP.Method, event.RequestContext.HTTP.Path)
			res, err := function_url.Handle(ctx, event, router)
			log.Println("Function URL Response", res.StatusCode)
			return res, err
		}
		if ok, event := legacy.IsEvent(rawEvent); ok {
			log.Println("Legacy Event", event.Type)
			return legacy.Handle(ctx, event, adminStore)
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
