package crud_invoke

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nullstone-io/go-rest-api"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
)

// This package handles invocations from a Terraform `aws_lambda_invocation` CRUD resource
// When the resource has an attribute `lifecycle_scope = "CRUD"`,
//   the payload will contain `tf` member with information about the action and previous input

type Event struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
	Tf   EventTf         `json:"tf"`
}

type EventTf struct {
	Action    string `json:"action"`
	PrevInput any    `json:"prev_input"`
}

func IsEvent(rawEvent json.RawMessage) (bool, Event) {
	var event Event
	if err := json.Unmarshal(rawEvent, &event); err != nil {
		return false, event
	}
	return event.Tf.Action != "", event
}

func Handle(ctx context.Context, event Event, store *postgresql.Store) (any, error) {
	crudHandler := CrudByName(store, event.Type)
	if crudHandler == nil {
		return nil, fmt.Errorf("unknown event 'type' %q", event.Type)
	}

	return crudHandler.Handle(event.Tf.Action, event.Data)
}

func CrudByName(s *postgresql.Store, name string) CrudHandler {
	switch name {
	case "databases":
		return Crud[string, postgresql.Database]{DataAccess: s.Databases}
	case "roles":
		return Crud[string, postgresql.Role]{DataAccess: s.Roles}
	case "role_members":
		return Crud[postgresql.RoleMemberKey, postgresql.RoleMember]{DataAccess: s.RoleMembers}
	case "schema_privileges":
		return Crud[postgresql.SchemaPrivilegeKey, postgresql.SchemaPrivilege]{DataAccess: s.SchemaPrivileges}
	case "default_grants":
		return Crud[postgresql.DefaultGrantKey, postgresql.DefaultGrant]{DataAccess: s.DefaultGrants}
	case "materialized_views":
		return Crud[postgresql.MaterializedViewsGrantKey, postgresql.MaterializedViewsGrant]{DataAccess: s.MaterializedViews}
	default:
		return nil
	}
}

type CrudHandler interface {
	Handle(action string, raw json.RawMessage) (any, error)
}

type Keyer[TKey any] interface {
	Key() TKey
}

type Crud[TKey any, T Keyer[TKey]] struct {
	DataAccess rest.DataAccess[TKey, T]
}

func (h Crud[TKey, T]) Handle(action string, raw json.RawMessage) (any, error) {
	var obj T
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("unable to parse input payload: %w", err)
	}

	switch action {
	case "create":
		return h.DataAccess.Create(obj)
	case "update":
		return h.DataAccess.Update(obj.Key(), obj)
	case "delete":
		return h.DataAccess.Drop(obj.Key())
	default:
		return nil, fmt.Errorf("unknown event 'action' %q", action)
	}
}
