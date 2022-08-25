package legacy

import (
	"fmt"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"log"
)

func GrantDbAccess(store postgresql.Store, username, databaseName string) error {
	log.Printf("Granting db access to user %q on database %q\n", username, databaseName)

	database, err := store.Databases.Read(databaseName)
	if err != nil {
		return fmt.Errorf("unable to read database %q: %w", databaseName, err)
	}
	if err := grantRole(store, username, database.Owner); err != nil {
		return err
	}
	if err := grantDefaultPrivileges(store, username, databaseName, database.Owner); err != nil {
		return err
	}
	return grantDbAndSchemaPrivileges(store, username, databaseName)
}

// grantRole adds user as a member of the database owner role
func grantRole(store postgresql.Store, username, databaseOwner string) error {
	log.Printf("Granting %q membership to %q", username, databaseOwner)
	newRoleGrant := postgresql.RoleMember{
		Member:      username,
		Target:      databaseOwner,
		UseExisting: true,
	}
	if _, err := store.RoleMembers.Create(newRoleGrant); err != nil {
		return fmt.Errorf("error granting role grant for %q to %q: %w", newRoleGrant.Member, newRoleGrant.Target, err)
	}
	return nil
}

func grantDefaultPrivileges(store postgresql.Store, roleName, databaseName, targetName string) error {
	log.Printf("Granting %q default privileges to %q", roleName, targetName)
	priv := postgresql.DefaultGrant{
		Role:     roleName,
		Database: databaseName,
		Target:   targetName,
	}
	_, err := store.DefaultGrants.Update(priv.Key(), priv)
	return err
}

func grantDbAndSchemaPrivileges(store postgresql.Store, roleName, databaseName string) error {
	log.Printf("Granting schema and database privileges on %q to %q", databaseName, roleName)
	priv := postgresql.SchemaPrivilege{
		Role:     roleName,
		Database: databaseName,
	}
	_, err := store.SchemaPrivileges.Update(priv.Key(), priv)
	return err
}
