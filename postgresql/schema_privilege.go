package postgresql

import (
	"fmt"
	"github.com/lib/pq"
	"github.com/nullstone-io/go-rest-api"
	"strings"
)

// SchemaPrivilege grants to Role on Database
//
//	CREATE|USAGE on public schema
//	CREATE|CONNECT|TEMPORARY on database
type SchemaPrivilege struct {
	Role     string `json:"role"`
	Database string `json:"database"`
}

func (p SchemaPrivilege) Key() SchemaPrivilegeKey {
	return SchemaPrivilegeKey{
		Role:     p.Role,
		Database: p.Database,
	}
}

type SchemaPrivilegeKey struct {
	Role     string
	Database string
}

var _ rest.DataAccess[SchemaPrivilegeKey, SchemaPrivilege] = &SchemaPrivileges{}

type SchemaPrivileges struct {
	DbOpener DbOpener
}

func (r *SchemaPrivileges) Create(obj SchemaPrivilege) (*SchemaPrivilege, error) {
	return r.Update(obj.Key(), obj)
}

func (r *SchemaPrivileges) Read(key SchemaPrivilegeKey) (*SchemaPrivilege, error) {
	db, err := r.DbOpener.OpenDatabase(key.Database)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// TODO: Introspect
	obj := SchemaPrivilege{
		Role:     key.Role,
		Database: key.Database,
	}
	return &obj, nil
}

func (r *SchemaPrivileges) Update(key SchemaPrivilegeKey, obj SchemaPrivilege) (*SchemaPrivilege, error) {
	db, err := r.DbOpener.OpenDatabase(obj.Database)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	sq := strings.Join([]string{
		// CREATE | USAGE
		fmt.Sprintf(`GRANT ALL PRIVILEGES ON SCHEMA public TO %s;`, pq.QuoteIdentifier(obj.Role)),
		// CREATE | CONNECT | TEMPORARY | TEMP
		fmt.Sprintf(`GRANT ALL PRIVILEGES ON DATABASE %s TO %s;`, pq.QuoteIdentifier(obj.Database), pq.QuoteIdentifier(obj.Role)),
	}, " ")
	_, err = db.Exec(sq)
	return &obj, err
}

func (r *SchemaPrivileges) Drop(key SchemaPrivilegeKey) (bool, error) {
	return true, nil
}
