package postgresql

type Store struct {
	Databases        *Databases
	Roles            *Roles
	RoleMembers      *RoleMembers
	DefaultGrants    *DefaultGrants
	SchemaPrivileges *SchemaPrivileges
}

func NewStore(connUrl string) *Store {
	return &Store{
		Databases:        &Databases{BaseConnectionUrl: connUrl},
		Roles:            &Roles{BaseConnectionUrl: connUrl},
		RoleMembers:      &RoleMembers{BaseConnectionUrl: connUrl},
		DefaultGrants:    &DefaultGrants{BaseConnectionUrl: connUrl},
		SchemaPrivileges: &SchemaPrivileges{BaseConnectionUrl: connUrl},
	}
}
