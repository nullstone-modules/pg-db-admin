package postgresql

import (
	"fmt"
	"github.com/go-multierror/multierror"
	"github.com/lib/pq"
	"github.com/nullstone-io/go-rest-api"
	"strings"
)

// RoleDefaultPrivilege defines a template of privileges that Role will be granted to Database
// This grants default privileges on schema objects created by Role in Database to Target
type RoleDefaultPrivilege struct {
	Role     string `json:"role"`
	Target   string `json:"target"`
	Database string `json:"database"`
}

func (priv RoleDefaultPrivilege) Key() RoleDefaultPrivilegeKey {
	return RoleDefaultPrivilegeKey{
		Role:     priv.Role,
		Target:   priv.Target,
		Database: priv.Database,
	}
}

type RoleDefaultPrivilegeKey struct {
	Role     string
	Target   string
	Database string
}

var _ rest.DataAccess[RoleDefaultPrivilegeKey, RoleDefaultPrivilege] = &RoleDefaultPrivileges{}

type RoleDefaultPrivileges struct {
	BaseConnectionUrl string
}

func (r *RoleDefaultPrivileges) Read(key RoleDefaultPrivilegeKey) (*RoleDefaultPrivilege, error) {
	db, err := OpenDatabase(r.BaseConnectionUrl, key.Database)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// TODO: Introspect
	obj := RoleDefaultPrivilege{
		Role:     key.Role,
		Database: key.Database,
		Target:   key.Target,
	}
	return &obj, nil
}

func (r *RoleDefaultPrivileges) Exists(priv RoleDefaultPrivilege) (bool, error) {
	existing, err := r.Read(priv.Key())
	return existing != nil, err
}

func (r *RoleDefaultPrivileges) Create(priv RoleDefaultPrivilege) (*RoleDefaultPrivilege, error) {
	return r.Update(priv.Key(), priv)
}

func (r *RoleDefaultPrivileges) Ensure(priv RoleDefaultPrivilege) (*RoleDefaultPrivilege, error) {
	return r.Update(priv.Key(), priv)
}

func (r *RoleDefaultPrivileges) Update(key RoleDefaultPrivilegeKey, priv RoleDefaultPrivilege) (*RoleDefaultPrivilege, error) {
	db, err := OpenDatabase(r.BaseConnectionUrl, priv.Database)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	info, err := CalcDbConnectionInfo(db)
	if err != nil {
		return nil, fmt.Errorf("error analyzing database: %w", err)
	}

	var grant Revoker = NoopRevoker{}
	var tempErr error
	if !info.IsSuperuser {
		grant, tempErr = GrantRoleMembership(db, priv.Role, info.CurrentUser)
		// We only care about this error if the privilege sql didn't work down below
	}

	quotedUserName := pq.QuoteIdentifier(priv.Role)
	quotedTarget := pq.QuoteIdentifier(priv.Target)

	sq := strings.Join([]string{
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s GRANT ALL PRIVILEGES ON TABLES TO %s;`, quotedUserName, quotedTarget),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s GRANT ALL PRIVILEGES ON SEQUENCES TO %s;`, quotedUserName, quotedTarget),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s GRANT ALL PRIVILEGES ON FUNCTIONS TO %s;`, quotedUserName, quotedTarget),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s GRANT ALL PRIVILEGES ON TYPES TO %s;`, quotedUserName, quotedTarget),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s GRANT ALL PRIVILEGES ON SCHEMAS TO %s;`, quotedUserName, quotedTarget),
	}, " ")
	errs := make([]error, 0)
	if _, err := db.Exec(sq); err != nil {
		if tempErr != nil {
			errs = append(errs, fmt.Errorf("error granting temporary membership: %w", tempErr))
		}
		errs = append(errs, fmt.Errorf("error altering default privileges: %w", err))
	}
	if grant != nil {
		if revokeErr := grant.Revoke(db); revokeErr != nil {
			errs = append(errs, fmt.Errorf("error revoking temporary membership: %w", revokeErr))
		}
	}
	if len(errs) > 0 {
		return nil, multierror.New(errs)
	}
	return &priv, nil
}

func (r *RoleDefaultPrivileges) Drop(key RoleDefaultPrivilegeKey) (bool, error) {
	return true, nil
}
