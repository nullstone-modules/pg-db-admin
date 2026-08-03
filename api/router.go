package api

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/nullstone-io/go-rest-api"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"log"
	"net/http"
	"strings"
)

func CreateRouter(store *postgresql.Store) *mux.Router {
	r := mux.NewRouter()

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%d %s %s\n", http.StatusNotFound, r.Method, r.RequestURI)
		http.NotFound(w, r)
	})
	r.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%d %s %s\n", http.StatusMethodNotAllowed, r.Method, r.RequestURI)
		http.Error(w, "", http.StatusMethodNotAllowed)
	})

	r.Methods(http.MethodDelete).Path("/skip").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

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
		DataAccess: store.RoleMembers,
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
		DataAccess: store.SchemaPrivileges,
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

	tablePrivileges := rest.Resource[postgresql.TablePrivilegeKey, postgresql.TablePrivilege]{
		DataAccess: store.TablePrivileges,
		KeyParser: func(r *http.Request) (postgresql.TablePrivilegeKey, error) {
			vars := mux.Vars(r)
			key := postgresql.TablePrivilegeKey{Database: vars["database"]}
			key.Schema, key.Role, _ = strings.Cut(vars["id"], "::")
			return key, nil
		},
	}
	r.Methods(http.MethodPost).Path("/databases/{database}/table_privileges").HandlerFunc(tablePrivileges.Create)
	r.Methods(http.MethodGet).Path("/databases/{database}/table_privileges/{id}").HandlerFunc(tablePrivileges.Get)
	r.Methods(http.MethodPut).Path("/databases/{database}/table_privileges/{id}").HandlerFunc(tablePrivileges.Update)
	r.Methods(http.MethodDelete).Path("/databases/{database}/table_privileges/{id}").HandlerFunc(tablePrivileges.Delete)

	publications := rest.Resource[postgresql.PublicationKey, postgresql.Publication]{
		DataAccess: store.Publications,
		KeyParser: func(r *http.Request) (postgresql.PublicationKey, error) {
			vars := mux.Vars(r)
			return postgresql.PublicationKey{
				Database: vars["database"],
				Name:     vars["name"],
			}, nil
		},
	}
	r.Methods(http.MethodPost).Path("/databases/{database}/publications").HandlerFunc(publications.Create)
	r.Methods(http.MethodGet).Path("/databases/{database}/publications/{name}").HandlerFunc(publications.Get)
	r.Methods(http.MethodPut).Path("/databases/{database}/publications/{name}").HandlerFunc(publications.Update)
	r.Methods(http.MethodDelete).Path("/databases/{database}/publications/{name}").HandlerFunc(publications.Delete)

	replicationSlots := rest.Resource[postgresql.ReplicationSlotKey, postgresql.ReplicationSlot]{
		DataAccess: store.ReplicationSlots,
		KeyParser: func(r *http.Request) (postgresql.ReplicationSlotKey, error) {
			vars := mux.Vars(r)
			return postgresql.ReplicationSlotKey{
				Database: vars["database"],
				Name:     vars["name"],
			}, nil
		},
	}
	r.Methods(http.MethodPost).Path("/databases/{database}/replication_slots").HandlerFunc(replicationSlots.Create)
	r.Methods(http.MethodGet).Path("/databases/{database}/replication_slots/{name}").HandlerFunc(replicationSlots.Get)
	r.Methods(http.MethodPut).Path("/databases/{database}/replication_slots/{name}").HandlerFunc(replicationSlots.Update)
	r.Methods(http.MethodDelete).Path("/databases/{database}/replication_slots/{name}").HandlerFunc(replicationSlots.Delete)

	defaultGrants := rest.Resource[postgresql.DefaultGrantKey, postgresql.DefaultGrant]{
		DataAccess: store.DefaultGrants,
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

	parameterPrivileges := rest.Resource[postgresql.ParameterPrivilegeKey, postgresql.ParameterPrivilege]{
		DataAccess: store.ParameterPrivileges,
		KeyParser: func(r *http.Request) (postgresql.ParameterPrivilegeKey, error) {
			vars := mux.Vars(r)
			return postgresql.ParameterPrivilegeKey{
				Role:      vars["role"],
				Parameter: vars["parameter"],
			}, nil
		},
	}
	r.Methods(http.MethodPost).Path("/roles/{role}/parameter_privileges").HandlerFunc(parameterPrivileges.Create)
	r.Methods(http.MethodGet).Path("/roles/{role}/parameter_privileges/{parameter}").HandlerFunc(parameterPrivileges.Get)
	r.Methods(http.MethodPut).Path("/roles/{role}/parameter_privileges/{parameter}").HandlerFunc(parameterPrivileges.Update)
	r.Methods(http.MethodDelete).Path("/roles/{role}/parameter_privileges/{parameter}").HandlerFunc(parameterPrivileges.Delete)

	// Reports server configuration (wal_level, current user, replication slot usage)
	// This lets callers detect an instance that cannot support logical replication before they try to use it
	r.Methods(http.MethodGet).Path("/settings").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		settings, err := store.Settings.Read()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		raw, err := json.Marshal(settings)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(raw)
	})

	return r
}
