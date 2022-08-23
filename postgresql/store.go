package postgresql

import "database/sql"

type Store struct {
	Databases             *Databases
	Roles                 *Roles
	RoleMembers           *RoleMembers
	RoleDefaultPrivileges *DefaultGrants
	SchemaPrivileges      *SchemaPrivileges
}

func NewStore(db *sql.DB, connUrl string) Store {
	return Store{
		Databases:             &Databases{Db: db},
		Roles:                 &Roles{Db: db},
		RoleMembers:           &RoleMembers{Db: db},
		RoleDefaultPrivileges: &DefaultGrants{BaseConnectionUrl: connUrl},
		SchemaPrivileges:      &SchemaPrivileges{BaseConnectionUrl: connUrl},
	}
}
