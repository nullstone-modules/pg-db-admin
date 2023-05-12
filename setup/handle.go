package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nullstone-modules/pg-db-admin/aws/secrets"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"net/url"
)

const (
	adminRoleUsernamePrefix = "nullstone_admin_role"
)

type Event struct {
	Setup bool `json:"setup"`
}

type EventResult struct {
	AdminRoleName   string `json:"adminRoleName"`
	SecretVersionId string `json:"secretVersionId"`
}

func IsEvent(rawEvent json.RawMessage) (bool, Event) {
	var event Event
	if err := json.Unmarshal(rawEvent, &event); err != nil {
		return false, event
	}
	return event.Setup, event
}

// Handle performs initial setup for a database
// We configure an admin role with a sufficiently unique name and add them to rds_superuser
// This is done to avoid a scenario where db_admin will not function properly
// This happens when the db_admin user has the same name as the database that a user wants to gain access
// In short, db_admin attempts the following membership chain (creating a cycle) <admin-role> -> <app-role> -> <admin-role>
// This admin user alters the membership chain to be <admin-role> -> <app-role> -> <database-owner>
func Handle(ctx context.Context, event Event, store *postgresql.Store, adminConnUrlSecretId string) (*EventResult, error) {
	toCreate, err := generateAdminRole(ctx, adminConnUrlSecretId)
	if err != nil {
		return nil, fmt.Errorf("unable to generate admin role: %w", err)
	}
	adminRole, err := store.Roles.Create(toCreate)
	if err != nil {
		return nil, fmt.Errorf("error ensure admin role: %w", err)
	} else if adminRole.Password == "" {
		// The role already exists, we're done
		versionId, err := secrets.GetLatestVersionId(ctx, adminConnUrlSecretId)
		if err != nil {
			return nil, fmt.Errorf("error retrieving admin secret version id: %w", err)
		}
		return &EventResult{SecretVersionId: versionId}, nil
	}

	// Build a connection url using the setup url, but with the admin role credentials
	adminConnUrl := urlWithUserinfo(store.ConnectionUrl(), adminRole.Name, adminRole.Password)
	// Set the value of the secret in secrets manager that holds the admin credentials
	versionId, err := secrets.SetString(ctx, adminConnUrlSecretId, adminConnUrl)
	if err != nil {
		return nil, fmt.Errorf("error saving admin credentials to a secret (%s): %w", adminConnUrlSecretId, err)
	}
	return &EventResult{SecretVersionId: versionId}, nil
}

func generateAdminRole(ctx context.Context, adminConnUrlSecretId string) (postgresql.Role, error) {
	role := postgresql.Role{
		UseExisting:        true,
		SkipPasswordUpdate: true,
		MemberOf:           []string{"rds_superuser"},
		Attributes: postgresql.RoleAttributes{
			Inherit:    true,
			CreateDb:   true,
			CreateRole: true,
		},
	}

	existingConnUrl, err := secrets.GetString(ctx, adminConnUrlSecretId)
	if u, err2 := url.Parse(existingConnUrl); err != nil || existingConnUrl == "" || err2 != nil {
		// Generate Name, Password
		role.Name, role.Password, err = randomRoleCreds(adminRoleUsernamePrefix)
		if err != nil {
			return role, fmt.Errorf("error generating user credentials: %w", err)
		}
	} else {
		role.Name = u.User.Username()
		role.Password, _ = u.User.Password()
	}

	return role, nil
}

func urlWithUserinfo(connUrl, username, password string) string {
	u, _ := url.Parse(connUrl)
	u.User = url.UserPassword(username, password)
	return u.String()
}
