package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gorilla/mux"
	"github.com/nullstone-io/go-lambda-api-sdk/function_url"
	"github.com/nullstone-modules/pg-db-admin/api"
)

const (
	eventTypeCreateDatabase = "create-database"
	eventTypeCreateUser     = "create-user"
	eventTypeCreateDbAccess = "create-db-access"
)

var (
	router   *mux.Router
	dbBroker *DbBroker
)

func main() {
	dbBroker, _ = NewDbBroker(context.TODO())
	router = api.CreateRouter(dbBroker)
	lambda.Start(HandleRequest)
}

func HandleRequest(ctx context.Context, rawEvent json.RawMessage) (any, error) {
	if ok, event := isFunctionUrlEvent(rawEvent); ok {
		return function_url.Handle(ctx, event, router)
	}
	if ok, event := isAdminEvent(rawEvent); ok {
		return handleAdminEvent(ctx, event)
	}
	return nil, nil
}

func isAdminEvent(rawEvent json.RawMessage) (bool, AdminEvent) {
	var event AdminEvent
	if err := json.Unmarshal(rawEvent, &event); err != nil {
		return false, event
	}
	return event.Type != "", event
}

func isFunctionUrlEvent(raw json.RawMessage) (bool, events.LambdaFunctionURLRequest) {
	var event events.LambdaFunctionURLRequest
	if err := json.Unmarshal(raw, &event); err != nil {
		return false, event
	}
	return event.RequestContext.HTTP.Method != "", event
}
