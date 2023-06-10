package pg_db_admin

// This is the entrypoint for a GCP Cloud Function
// A Cloud Function (2nd gen) *must* be built using GCP Cloud Build
// This requires us to do the following:
//   - Package all source code (including vendor) in the zip file
//   - main.go *must* be at the root of the zip file
//   - package name in main.go must match module name defined in go.mod (cannot be `package main`)
//
// This entrypoint does not run code; it only registers a trigger that is used by the runtime upon execution

import (
	"fmt"
	_ "github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/nullstone-modules/pg-db-admin/api"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"os"
)

var (
	dbConnUrlEnvVar = "DB_CONN_URL"
)

func init() {
	fmt.Println("Initializing pg-db-admin...")
	store := postgresql.NewStore(os.Getenv(dbConnUrlEnvVar))
	router := api.CreateRouter(store)
	functions.HTTP("pg-db-admin", router.ServeHTTP)
}
