package api

import (
	"database/sql"
	"github.com/gorilla/mux"
	"github.com/nullstone-io/go-rest-api"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"log"
	"net/http"
	"strings"
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
			defer db.Close()

			handler.ServeHTTP(w, r)
		})
	})
	r.Use(dbMiddleware)

	store := postgresql.NewStore(db, dbConnUrl)

	databases := &rest.Resource[string, postgresql.Database]{
		DataAccess: store.Databases,
		KeyParser:  rest.PathParameterKeyParser("name"),
	}
	r.Methods(http.MethodPost).Path("/databases").HandlerFunc(databases.Create)
	r.Methods(http.MethodGet).Path("/databases/{name}").HandlerFunc(databases.Get)
	r.Methods(http.MethodPut).Path("/databases/{name}").HandlerFunc(databases.Update)
	r.Methods(http.MethodDelete).Path("/databases/{name}").HandlerFunc(databases.Delete)

	roles := rest.Resource[string, postgresql.Role]{
		DataAccess: store.Roles,
		KeyParser:  rest.PathParameterKeyParser("name"),
	}
	r.Methods(http.MethodPost).Path("/roles").HandlerFunc(roles.Create)
	r.Methods(http.MethodGet).Path("/roles/{name}").HandlerFunc(roles.Get)
	r.Methods(http.MethodPut).Path("/roles/{name}").HandlerFunc(roles.Update)
	r.Methods(http.MethodDelete).Path("/roles/{name}").HandlerFunc(roles.Delete)

	roleMembers := rest.Resource[postgresql.RoleMemberKey, postgresql.RoleMember]{
		DataAccess: &postgresql.RoleMembers{Db: db},
		KeyParser: func(r *http.Request) (postgresql.RoleMemberKey, error) {
			vars := mux.Vars(r)
			return postgresql.RoleMemberKey{
				Member: vars["member"],
				Target: vars["target"],
			}, nil
		},
	}
	r.Methods(http.MethodPost).Path("/roles/{target}/members").HandlerFunc(roleMembers.Create)
	r.Methods(http.MethodGet).Path("/roles/{target}/members/{member}").HandlerFunc(roleMembers.Get)
	r.Methods(http.MethodPut).Path("/roles/{target}/members/{member}").HandlerFunc(roleMembers.Update)
	r.Methods(http.MethodDelete).Path("/roles/{target}/members/{member}").HandlerFunc(roleMembers.Delete)

	schemaPrivileges := rest.Resource[postgresql.SchemaPrivilegeKey, postgresql.SchemaPrivilege]{
		DataAccess: &postgresql.SchemaPrivileges{BaseConnectionUrl: dbConnUrl},
		KeyParser: func(r *http.Request) (postgresql.SchemaPrivilegeKey, error) {
			vars := mux.Vars(r)
			return postgresql.SchemaPrivilegeKey{
				Database: vars["database"],
				Role:     vars["role"],
			}, nil
		},
	}
	r.Methods(http.MethodPost).Path("/databases/{database}/schema_privileges").HandlerFunc(schemaPrivileges.Create)
	r.Methods(http.MethodGet).Path("/databases/{database}/schema_privileges/{role}").HandlerFunc(schemaPrivileges.Get)
	r.Methods(http.MethodPut).Path("/databases/{database}/schema_privileges/{role}").HandlerFunc(schemaPrivileges.Update)
	r.Methods(http.MethodDelete).Path("/databases/{database}/schema_privileges/{role}").HandlerFunc(schemaPrivileges.Delete)

	defaultGrants := rest.Resource[postgresql.DefaultGrantKey, postgresql.DefaultGrant]{
		DataAccess: &postgresql.DefaultGrants{BaseConnectionUrl: dbConnUrl},
		KeyParser: func(r *http.Request) (postgresql.DefaultGrantKey, error) {
			vars := mux.Vars(r)
			id := vars["id"]
			key := postgresql.DefaultGrantKey{Role: vars["role"]}
			key.Target, key.Database, _ = strings.Cut(id, "::")
			return key, nil
		},
	}
	r.Methods(http.MethodPost).Path("/roles/{role}/default_grants").HandlerFunc(defaultGrants.Create)
	r.Methods(http.MethodGet).Path("/roles/{role}/default_grants/{id}").HandlerFunc(defaultGrants.Get)
	r.Methods(http.MethodPut).Path("/roles/{role}/default_grants/{id}").HandlerFunc(defaultGrants.Update)
	r.Methods(http.MethodDelete).Path("/roles/{role}/default_grants/{id}").HandlerFunc(defaultGrants.Delete)

	return r
}
