package postgresql

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.com/nullstone-io/go-rest-api"
	"log"
	"regexp"
)

// ParameterPrivilege grants Role the privilege to `SET` a configuration parameter that otherwise
// requires a superuser context -- `session_replication_role`, for example, which a bulk load has to
// set to `replica` to disable foreign key triggers.
//
// Membership in a managed superuser role (`cloudsqlsuperuser`, `rds_superuser`) does not help here:
// postgres never inherits role *attributes* through membership, so a member of a superuser-ish role
// is still refused a superuser-context parameter. `GRANT SET ON PARAMETER` (postgres 15+) is the
// narrowest fix -- it hands over exactly one parameter instead of the whole role.
//
// Parameter privileges are cluster-wide (`pg_parameter_acl` is a shared catalog), so this resource
// intentionally has no database field and connects to the default database.
type ParameterPrivilege struct {
	Role      string `json:"role"`
	Parameter string `json:"parameter"`
}

func (p ParameterPrivilege) Key() ParameterPrivilegeKey {
	return ParameterPrivilegeKey{
		Role:      p.Role,
		Parameter: p.Parameter,
	}
}

type ParameterPrivilegeKey struct {
	Role      string
	Parameter string
}

// validParameterName constrains what may be interpolated into `GRANT SET ON PARAMETER <name>`.
//
// A configuration parameter is not an identifier: postgres takes it as a bare token, and quoting it
// with pq.QuoteIdentifier produces `GRANT SET ON PARAMETER "session_replication_role"`, which
// postgres accepts but which offers no protection for a name that is chosen by module configuration.
// Constraining the name here is what makes the interpolation safe.
//
// Extension parameters (`pg_stat_statements.track`) are excluded deliberately: they are not
// superuser-context parameters, and allowing '.' widens the set of characters that reach the
// statement for no benefit.
var validParameterName = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

func validateParameterName(parameter string) error {
	if parameter == "" {
		return fmt.Errorf("parameter is required")
	}
	if !validParameterName.MatchString(parameter) {
		return fmt.Errorf("invalid configuration parameter %q: expected a name matching %s", parameter, validParameterName)
	}
	return nil
}

// verifyParameterPrivilegesSupported reports an unsupported server rather than skipping the grant.
//
// A silent no-op is the dangerous outcome: the workload would believe it holds a privilege it does
// not, and would only discover otherwise partway through a restore.
func verifyParameterPrivilegesSupported(info *DbInfo) error {
	if info.SupportedFeatures.IsSupported(FeatureParameterPrivileges) {
		return nil
	}
	return fmt.Errorf("granting SET on a configuration parameter requires postgres 15 or later, but the server is %s; "+
		"upgrade the instance or remove the parameter from the capability's settable parameters", info.DbVersion)
}

var _ rest.DataAccess[ParameterPrivilegeKey, ParameterPrivilege] = &ParameterPrivileges{}

type ParameterPrivileges struct {
	DbOpener DbOpener
}

func (p *ParameterPrivileges) Create(obj ParameterPrivilege) (*ParameterPrivilege, error) {
	return p.Update(obj.Key(), obj)
}

// Read introspects the privilege so that the resource is idempotent, rather than reporting success
// for a grant that was never applied
func (p *ParameterPrivileges) Read(key ParameterPrivilegeKey) (*ParameterPrivilege, error) {
	db, err := p.DbOpener.OpenDatabase("")
	if err != nil {
		return nil, err
	}

	info, err := CalcDbConnectionInfo(db)
	if err != nil {
		return nil, fmt.Errorf("error analyzing database: %w", err)
	}
	// The read is version gated as well as the write: `has_parameter_privilege` does not exist before
	// postgres 15, so an ungated read fails with an opaque "function does not exist" instead of
	// the explanation above
	if err := verifyParameterPrivilegesSupported(info); err != nil {
		return nil, err
	}
	if err := validateParameterName(key.Parameter); err != nil {
		return nil, err
	}

	// has_parameter_privilege raises an error for a role that does not exist, which is the ordinary
	// state when this is read before the role has been created
	exists, err := roleExists(db, key.Role)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	var granted bool
	sq := `SELECT has_parameter_privilege($1, $2, 'SET')`
	if err := db.QueryRow(sq, key.Role, key.Parameter).Scan(&granted); err != nil {
		return nil, fmt.Errorf("error reading SET privilege on parameter %q for role %q: %w", key.Parameter, key.Role, err)
	}
	if !granted {
		return nil, nil
	}

	return &ParameterPrivilege{Role: key.Role, Parameter: key.Parameter}, nil
}

// Update grants the privilege.
//
// Unlike DefaultGrants, this acquires no temporary role membership first: a parameter grant is
// refused for anyone but a superuser, or a role that a superuser previously granted the parameter
// to WITH GRANT OPTION. Neither membership in a managed superuser role nor `SET ROLE` to it changes
// that, so there is nothing useful to acquire.
func (p *ParameterPrivileges) Update(key ParameterPrivilegeKey, obj ParameterPrivilege) (*ParameterPrivilege, error) {
	obj.Role, obj.Parameter = key.Role, key.Parameter

	db, err := p.DbOpener.OpenDatabase("")
	if err != nil {
		return nil, err
	}

	info, err := CalcDbConnectionInfo(db)
	if err != nil {
		return nil, fmt.Errorf("error analyzing database: %w", err)
	}
	if err := verifyParameterPrivilegesSupported(info); err != nil {
		return nil, err
	}
	if err := validateParameterName(obj.Parameter); err != nil {
		return nil, err
	}

	log.Printf("Granting SET on parameter %q to role %q\n", obj.Parameter, obj.Role)
	sq := fmt.Sprintf(`GRANT SET ON PARAMETER %s TO %s`, obj.Parameter, pq.QuoteIdentifier(obj.Role))
	if _, err := db.Exec(sq); err != nil {
		// The likely cause is worth naming: postgres only lets a superuser grant a parameter, or a role
		// that a superuser previously granted it to WITH GRANT OPTION. On a managed engine the admin is
		// neither, and the resulting "permission denied for parameter" says nothing about why
		return nil, fmt.Errorf("error granting SET on parameter %q to role %q as %q (superuser=%t): %w; "+
			"this requires a superuser, or a role that holds %q WITH GRANT OPTION -- membership in "+
			"cloudsqlsuperuser/rds_superuser is not sufficient",
			obj.Parameter, obj.Role, info.CurrentUser, info.IsSuperuser, err, obj.Parameter)
	}
	return &obj, nil
}

// Drop leaves the privilege in place, matching the other privilege resources in this package
// The terraform side points destroy at `/skip`, so this is never reached from a module
func (p *ParameterPrivileges) Drop(key ParameterPrivilegeKey) (bool, error) {
	return true, nil
}

// roleExists reports whether a role is present, so that callers can distinguish "no such role" from
// "role holds no privilege"
func roleExists(db *sql.DB, role string) (bool, error) {
	var exists bool
	sq := `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`
	if err := db.QueryRow(sq, role).Scan(&exists); err != nil {
		return false, fmt.Errorf("could not check whether role %q exists: %w", role, err)
	}
	return exists, nil
}
