package api

import (
	"github.com/gorilla/mux"
	"github.com/nullstone-io/go-rest-api"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"net/http"
)

func CreateRouter(dbBroker rest.DbBroker) *mux.Router {
	r := mux.NewRouter()

	databases := rest.Resource[*postgresql.Database]{
		Db:              dbBroker,
		IdPathParameter: "name",
	}
	r.Methods(http.MethodPost).Path("databases").HandlerFunc(databases.Create)
	r.Methods(http.MethodGet).Path("databases/{name}").HandlerFunc(databases.Get)
	r.Methods(http.MethodPut).Path("databases/{name}").HandlerFunc(databases.Update)
	r.Methods(http.MethodDelete).Path("databases/{name}").HandlerFunc(databases.Delete)

	roles := rest.Resource[*postgresql.Role]{
		Db:              dbBroker,
		IdPathParameter: "name",
	}
	r.Methods(http.MethodPost).Path("roles").HandlerFunc(roles.Create)
	r.Methods(http.MethodGet).Path("roles/{name}").HandlerFunc(roles.Get)
	r.Methods(http.MethodPut).Path("roles/{name}").HandlerFunc(roles.Update)
	r.Methods(http.MethodDelete).Path("roles/{name}").HandlerFunc(roles.Delete)

	roleAccess := RoleAccess{DbBroker: dbBroker}
	r.Methods(http.MethodGet).Path("role_access").HandlerFunc(roleAccess.List)
	r.Methods(http.MethodPost).Path("role_access").HandlerFunc(roleAccess.Create)
	r.Methods(http.MethodGet).Path("role_access/{name}").HandlerFunc(roleAccess.Get)
	r.Methods(http.MethodPut).Path("role_access/{name}").HandlerFunc(roleAccess.Update)
	r.Methods(http.MethodDelete).Path("role_access/{name}").HandlerFunc(roleAccess.Delete)

	return r
}
