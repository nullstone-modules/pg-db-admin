package postgresql

import (
	"database/sql"
	"fmt"
	"github.com/go-multierror/multierror"
	"github.com/lib/pq"
	"github.com/nullstone-io/go-rest-api"
	"log"
	"sort"
	"strings"
)

// AllSchemas selects every non-system schema in the database
const AllSchemas = "*"

// validTablePrivileges is an allowlist because privilege keywords cannot be quoted as identifiers or literals
var validTablePrivileges = map[string]bool{
	"SELECT":     true,
	"INSERT":     true,
	"UPDATE":     true,
	"DELETE":     true,
	"TRUNCATE":   true,
	"REFERENCES": true,
	"TRIGGER":    true,
}

// TablePrivilege grants table-level access on a schema to Role
//
//	GRANT USAGE ON SCHEMA <schema> TO <role>
//	GRANT <privileges> ON ALL TABLES IN SCHEMA <schema> TO <role>
//	ALTER DEFAULT PRIVILEGES [FOR ROLE ...] IN SCHEMA <schema> GRANT <privileges> ON TABLES TO <role>
//
// Unlike SchemaPrivilege -- which grants full access so an application can own and modify its schema --
// this grants a specific (typically read-only) set of privileges, e.g. for a replication consumer.
type TablePrivilege struct {
	Id       string `json:"id"`
	Database string `json:"database"`
	// Schema is the schema to grant on, or `*` for every non-system schema in the database
	Schema string `json:"schema"`
	Role   string `json:"role"`

	// Privileges defaults to `["SELECT"]`
	Privileges []string `json:"privileges"`

	// IncludeFuture also grants the privileges on tables created later, via ALTER DEFAULT PRIVILEGES
	IncludeFuture bool `json:"includeFuture"`
	// FutureFromRoles lists the roles whose future tables are covered by IncludeFuture
	// Default privileges only apply to objects created by a specific role, so covering an application's future tables
	// requires naming the role that creates them -- an empty list only covers tables created by the admin user itself
	FutureFromRoles []string `json:"futureFromRoles"`
	// FutureFromTableOwners adds every role that currently owns a table in the schema to FutureFromRoles
	FutureFromTableOwners bool `json:"futureFromTableOwners"`

	// GrantConnect also grants CONNECT on the database
	GrantConnect bool `json:"grantConnect"`
}

func (p *TablePrivilege) SetId() {
	p.Id = fmt.Sprintf("%s::%s", p.Schema, p.Role)
}

func (p TablePrivilege) Key() TablePrivilegeKey {
	return TablePrivilegeKey{
		Database: p.Database,
		Schema:   p.Schema,
		Role:     p.Role,
	}
}

type TablePrivilegeKey struct {
	Database string
	Schema   string
	Role     string
}

var _ rest.DataAccess[TablePrivilegeKey, TablePrivilege] = &TablePrivileges{}

type TablePrivileges struct {
	DbOpener DbOpener
}

func (t *TablePrivileges) Create(obj TablePrivilege) (*TablePrivilege, error) {
	return t.Update(obj.Key(), obj)
}

func (t *TablePrivileges) Read(key TablePrivilegeKey) (*TablePrivilege, error) {
	// Introspecting the effective privileges of a role across a schema (and its default privileges) is not
	// meaningfully comparable to the requested grant, so this echoes the key like the other grant resources
	obj := TablePrivilege{
		Database: key.Database,
		Schema:   key.Schema,
		Role:     key.Role,
	}
	obj.SetId()
	return &obj, nil
}

func (t *TablePrivileges) Update(key TablePrivilegeKey, obj TablePrivilege) (*TablePrivilege, error) {
	obj.Database, obj.Schema, obj.Role = key.Database, key.Schema, key.Role
	if obj.Schema == "" {
		obj.Schema = "public"
	}

	privileges, err := normalizePrivileges(obj.Privileges)
	if err != nil {
		return nil, err
	}

	db, err := t.DbOpener.OpenDatabase(obj.Database)
	if err != nil {
		return nil, err
	}

	schemas := []string{obj.Schema}
	if obj.Schema == AllSchemas {
		if schemas, err = readUserSchemas(db); err != nil {
			return nil, err
		}
	}

	info, err := CalcDbConnectionInfo(db)
	if err != nil {
		return nil, fmt.Errorf("error analyzing database: %w", err)
	}

	if obj.GrantConnect {
		sq := fmt.Sprintf(`GRANT CONNECT ON DATABASE %s TO %s`, pq.QuoteIdentifier(obj.Database), pq.QuoteIdentifier(obj.Role))
		if _, err := db.Exec(sq); err != nil {
			return nil, fmt.Errorf("error granting connect on database %q to %q: %w", obj.Database, obj.Role, err)
		}
	}

	for _, schema := range schemas {
		if err := t.grantSchema(db, obj, schema, privileges, info); err != nil {
			return nil, err
		}
	}

	obj.SetId()
	return &obj, nil
}

func (t *TablePrivileges) Drop(key TablePrivilegeKey) (bool, error) {
	return true, nil
}

func (t *TablePrivileges) grantSchema(db *sql.DB, obj TablePrivilege, schema string, privileges string, info *DbInfo) error {
	quotedRole := pq.QuoteIdentifier(obj.Role)
	quotedSchema := pq.QuoteIdentifier(schema)

	log.Printf("Granting %s on schema %q in database %q to %q\n", privileges, schema, obj.Database, obj.Role)
	sq := strings.Join([]string{
		fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO %s;`, quotedSchema, quotedRole),
		fmt.Sprintf(`GRANT %s ON ALL TABLES IN SCHEMA %s TO %s;`, privileges, quotedSchema, quotedRole),
	}, " ")
	if _, err := db.Exec(sq); err != nil {
		return fmt.Errorf("error granting %s on schema %q to %q: %w", privileges, schema, obj.Role, err)
	}

	if !obj.IncludeFuture {
		return nil
	}

	futureFrom, err := t.resolveFutureFromRoles(db, obj, schema)
	if err != nil {
		return err
	}
	for _, fromRole := range futureFrom {
		if err := t.alterDefaultPrivileges(db, obj, schema, privileges, fromRole, info); err != nil {
			return err
		}
	}
	// An empty FOR ROLE clause applies to objects created by the current user
	if len(futureFrom) < 1 {
		return t.alterDefaultPrivileges(db, obj, schema, privileges, "", info)
	}
	return nil
}

// alterDefaultPrivileges grants the privileges on tables that fromRole creates in the schema going forward
// Altering another role's default privileges requires membership in that role, so -- like DefaultGrants --
// a non-superuser admin temporarily grants itself membership
func (t *TablePrivileges) alterDefaultPrivileges(db *sql.DB, obj TablePrivilege, schema string, privileges string, fromRole string, info *DbInfo) error {
	forRole := ""
	var revoker Revoker = NoopRevoker{}
	var tempErr error
	if fromRole != "" {
		forRole = fmt.Sprintf("FOR ROLE %s ", pq.QuoteIdentifier(fromRole))
		if !info.IsSuperuser {
			revoker, tempErr = GrantRoleMembership(db, fromRole, info.CurrentUser)
			// We only care about this error if the privilege sql didn't work down below
		}
	}
	if revoker == nil {
		revoker = NoopRevoker{}
	}

	log.Printf("Granting %s on future tables in schema %q created by %q to %q\n", privileges, schema, orCurrentUser(fromRole, info), obj.Role)
	sq := fmt.Sprintf(`ALTER DEFAULT PRIVILEGES %sIN SCHEMA %s GRANT %s ON TABLES TO %s;`,
		forRole, pq.QuoteIdentifier(schema), privileges, pq.QuoteIdentifier(obj.Role))

	errs := make([]error, 0)
	if _, err := db.Exec(sq); err != nil {
		if tempErr != nil {
			errs = append(errs, fmt.Errorf("error granting temporary membership: %w", tempErr))
		}
		errs = append(errs, fmt.Errorf("error altering default privileges in schema %q for role %q: %w", schema, orCurrentUser(fromRole, info), err))
	}
	if revokeErr := revoker.Revoke(db); revokeErr != nil {
		errs = append(errs, fmt.Errorf("error revoking temporary membership: %w", revokeErr))
	}
	if len(errs) > 0 {
		return multierror.New(errs)
	}
	return nil
}

func (t *TablePrivileges) resolveFutureFromRoles(db *sql.DB, obj TablePrivilege, schema string) ([]string, error) {
	found := map[string]bool{}
	for _, role := range obj.FutureFromRoles {
		found[role] = true
	}

	if obj.FutureFromTableOwners {
		owners, err := readTableOwners(db, schema)
		if err != nil {
			return nil, err
		}
		for _, owner := range owners {
			found[owner] = true
		}
	}

	roles := keys(found)
	sort.Strings(roles)
	return roles, nil
}

// readUserSchemas lists the schemas that hold user data, excluding the catalogs postgres manages itself
func readUserSchemas(db *sql.DB) ([]string, error) {
	sq := `SELECT nspname FROM pg_namespace
WHERE nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
	AND nspname NOT LIKE 'pg\_temp\_%'
	AND nspname NOT LIKE 'pg\_toast\_temp\_%'
ORDER BY nspname`
	rows, err := db.Query(sq)
	if err != nil {
		return nil, fmt.Errorf("error reading schemas: %w", err)
	}
	defer rows.Close()

	schemas := make([]string, 0)
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, err
		}
		schemas = append(schemas, schema)
	}
	return schemas, rows.Err()
}

func readTableOwners(db *sql.DB, schema string) ([]string, error) {
	rows, err := db.Query(`SELECT DISTINCT tableowner FROM pg_tables WHERE schemaname = $1`, schema)
	if err != nil {
		return nil, fmt.Errorf("error reading table owners in schema %q: %w", schema, err)
	}
	defer rows.Close()

	owners := make([]string, 0)
	for rows.Next() {
		var owner string
		if err := rows.Scan(&owner); err != nil {
			return nil, err
		}
		owners = append(owners, owner)
	}
	return owners, rows.Err()
}

func normalizePrivileges(privileges []string) (string, error) {
	if len(privileges) < 1 {
		return "SELECT", nil
	}

	normalized := make([]string, 0, len(privileges))
	for _, privilege := range privileges {
		cur := strings.ToUpper(strings.TrimSpace(privilege))
		if !validTablePrivileges[cur] {
			return "", fmt.Errorf("invalid table privilege %q", privilege)
		}
		normalized = append(normalized, cur)
	}
	return strings.Join(normalized, ", "), nil
}

func orCurrentUser(role string, info *DbInfo) string {
	if role == "" {
		return info.CurrentUser
	}
	return role
}
