package main

import (
	"context"
	"encoding/json"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
)

// Handles invocations from a Terraform `aws_lambda_invocation` data source, which re-invokes on every
// plan/apply. Data-source invocations carry no `tf` envelope, so this event is matched by type alone
// and must dispatch before the legacy handler, which claims any payload with a non-empty `type`.

const ensureRolePasswordEventType = "ensure_role_password"

type EnsureRolePasswordEvent struct {
	Type string                      `json:"type"`
	Data EnsureRolePasswordEventData `json:"data"`
}

type EnsureRolePasswordEventData struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func IsEnsureRolePasswordEvent(rawEvent json.RawMessage) (bool, EnsureRolePasswordEvent) {
	var event EnsureRolePasswordEvent
	if err := json.Unmarshal(rawEvent, &event); err != nil {
		return false, event
	}
	return event.Type == ensureRolePasswordEventType, event
}

func HandleEnsureRolePassword(ctx context.Context, event EnsureRolePasswordEvent, store *postgresql.Store) (any, error) {
	return store.Roles.EnsurePassword(event.Data.Name, event.Data.Password)
}
