package api

import (
	"database/sql"
	"github.com/gorilla/mux"
	"github.com/nullstone-io/go-rest-api"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"log"
	"net/http"
)

func CreateRouter(dbConnUrl string) *mux.Router {
	r := mux.NewRouter()

	var db *sql.DB
	dbMiddleware := mux.MiddlewareFunc(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var err error
			db, err = sql.Open("postgres", dbConnUrl)
			if err != nil {
				log.Printf("unable to connect to database: %s\n", err)
				http.Error(w, "unable to connect to database", http.StatusInternalServerError)
				return
			}

			handler.ServeHTTP(w, r)
		})
	})
	r.Use(dbMiddleware)

	databases := rest.Resource[string, postgresql.Database]{
		DataAccess:      &postgresql.Databases{Db: db},
		IdPathParameter: "name",
	}
	r.Methods(http.MethodPost).Path("/databases").HandlerFunc(databases.Create)
	r.Methods(http.MethodGet).Path("/databases/{name}").HandlerFunc(databases.Get)
	r.Methods(http.MethodPut).Path("/databases/{name}").HandlerFunc(databases.Update)
	r.Methods(http.MethodDelete).Path("/databases/{name}").HandlerFunc(databases.Delete)

	roles := rest.Resource[string, postgresql.Role]{
		DataAccess:      &postgresql.Roles{Db: db},
		IdPathParameter: "name",
	}
	r.Methods(http.MethodPost).Path("/roles").HandlerFunc(roles.Create)
	r.Methods(http.MethodGet).Path("/roles/{name}").HandlerFunc(roles.Get)
	r.Methods(http.MethodPut).Path("/roles/{name}").HandlerFunc(roles.Update)
	r.Methods(http.MethodDelete).Path("/roles/{name}").HandlerFunc(roles.Delete)

	roleGrants := rest.Resource[string, postgresql.RoleGrant]{
		DataAccess:      &postgresql.RoleGrants{Db: db},
		IdPathParameter: "id",
	}
	r.Methods(http.MethodPost).Path("/role_grants").HandlerFunc(roleGrants.Create)
	r.Methods(http.MethodGet).Path("/role_grants/{id}").HandlerFunc(roleGrants.Get)
	r.Methods(http.MethodPut).Path("/role_grants/{id}").HandlerFunc(roleGrants.Update)
	r.Methods(http.MethodDelete).Path("/role_grants/{id}").HandlerFunc(roleGrants.Delete)

	return r
}
