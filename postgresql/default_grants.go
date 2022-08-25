package postgresql

import (
	"fmt"
	"github.com/go-multierror/multierror"
	"github.com/lib/pq"
	"github.com/nullstone-io/go-rest-api"
	"strings"
)

// DefaultGrant defines a template of privileges that Role will be granted to Database
// This grants default privileges on schema objects created by Role in Database to Target
type DefaultGrant struct {
	Id       string `json:"id"`
	Role     string `json:"role"`
	Target   string `json:"target"`
	Database string `json:"database"`
}

func (g *DefaultGrant) SetId() {
	g.Id = fmt.Sprintf("%s::%s", g.Target, g.Database)
}

func (g DefaultGrant) Key() DefaultGrantKey {
	return DefaultGrantKey{
		Role:     g.Role,
		Target:   g.Target,
		Database: g.Database,
	}
}

type DefaultGrantKey struct {
	Role     string
	Target   string
	Database string
}

var _ rest.DataAccess[DefaultGrantKey, DefaultGrant] = &DefaultGrants{}

type DefaultGrants struct {
	BaseConnectionUrl string
}

func (g *DefaultGrants) Create(grant DefaultGrant) (*DefaultGrant, error) {
	return g.Update(grant.Key(), grant)
}

func (g *DefaultGrants) Read(key DefaultGrantKey) (*DefaultGrant, error) {
	db, err := OpenDatabase(g.BaseConnectionUrl, key.Database)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// TODO: Introspect
	grant := DefaultGrant{
		Role:     key.Role,
		Database: key.Database,
		Target:   key.Target,
	}
	grant.SetId()
	return &grant, nil
}

func (g *DefaultGrants) Update(key DefaultGrantKey, grant DefaultGrant) (*DefaultGrant, error) {
	db, err := OpenDatabase(g.BaseConnectionUrl, grant.Database)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	info, err := CalcDbConnectionInfo(db)
	if err != nil {
		return nil, fmt.Errorf("error analyzing database: %w", err)
	}

	var revoker Revoker = NoopRevoker{}
	var tempErr error
	if !info.IsSuperuser {
		revoker, tempErr = GrantRoleMembership(db, grant.Role, info.CurrentUser)
		// We only care about this error if the privilege sql didn't work down below
	}

	quotedUserName := pq.QuoteIdentifier(grant.Role)
	quotedTarget := pq.QuoteIdentifier(grant.Target)

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
	if revoker != nil {
		if revokeErr := revoker.Revoke(db); revokeErr != nil {
			errs = append(errs, fmt.Errorf("error revoking temporary membership: %w", revokeErr))
		}
	}
	if len(errs) > 0 {
		return nil, multierror.New(errs)
	}
	grant.SetId()
	return &grant, nil
}

func (g *DefaultGrants) Drop(key DefaultGrantKey) (bool, error) {
	return true, nil
}
