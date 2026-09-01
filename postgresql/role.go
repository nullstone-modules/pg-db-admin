package postgresql

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"github.com/nullstone-io/go-rest-api"
	"log"
	"strings"
)

var ErrRoleNotFound = errors.New("role not found")

type Role struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	// Do not error if trying to create a role that already exists
	// Instead, read the existing, set the password, and return
	UseExisting bool `json:"useExisting"`
	// SkipPasswordUpdate informs Create to skip updating the role's password if the role already exists
	SkipPasswordUpdate bool `json:"-"`

	MemberOf   []string       `json:"memberOf"`
	Attributes RoleAttributes `json:"attributes"`
}

func (r Role) Key() string {
	return r.Name
}

type RoleAttributes struct {
	CreateDb   bool `json:"createDb"`
	CreateRole bool `json:"createRole"`
	// Replication allows the role to initiate streaming replication and to create/drop replication slots
	// This is required for logical replication consumers (e.g. Datastream)
	Replication bool `json:"replication"`
}

// alterSql produces the `ALTER ROLE ... WITH <attrs>` fragments necessary to add missing attributes to an existing role
// Attributes are only ever added, never removed:
// a role is frequently shared with other tooling and dropping an attribute we didn't set would be destructive
func (a RoleAttributes) alterSql(existing RoleAttributes, features Features) []string {
	attrs := make([]string, 0)
	if a.CreateRole && !existing.CreateRole {
		attrs = append(attrs, "CREATEROLE")
	}
	if a.CreateDb && !existing.CreateDb {
		attrs = append(attrs, "CREATEDB")
	}
	if a.Replication && !existing.Replication && features.IsSupported(FeatureReplication) {
		attrs = append(attrs, "REPLICATION")
	}
	return attrs
}

var _ rest.DataAccess[string, Role] = &Roles{}

type Roles struct {
	DbOpener DbOpener
}

func (r *Roles) Create(role Role) (*Role, error) {
	if role.UseExisting {
		if existing, err := r.Read(role.Name); err != nil {
			return nil, err
		} else if existing != nil {
			log.Printf("[Create] Role %q already exists, updating...\n", role.Name)
			return r.Update(role.Name, role)
		}
	}

	db, err := r.DbOpener.OpenDatabase("")
	if err != nil {
		return nil, err
	}

	info, err := CalcDbConnectionInfo(db)
	if err != nil {
		return nil, fmt.Errorf("error analyzing database: %w", err)
	}

	fmt.Printf("Creating role %q\n", role.Name)
	if _, err := db.Exec(r.generateCreateSql(role, info.SupportedFeatures)); err != nil {
		return nil, fmt.Errorf("error creating user %q: %w", role.Name, err)
	}
	return &role, nil
}

func (r *Roles) Read(key string) (*Role, error) {
	db, err := r.DbOpener.OpenDatabase("")
	if err != nil {
		return nil, err
	}

	var name string
	var attrs RoleAttributes
	row := db.QueryRow(`SELECT rolname, rolcreatedb, rolcreaterole, rolreplication from pg_roles WHERE rolname = $1`, key)
	if err := row.Scan(&name, &attrs.CreateDb, &attrs.CreateRole, &attrs.Replication); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &Role{Name: name, Attributes: attrs}, nil
}

func (r *Roles) Update(key string, role Role) (*Role, error) {
	db, err := r.DbOpener.OpenDatabase("")
	if err != nil {
		return nil, err
	}

	if err := r.updateAttributes(db, key, role); err != nil {
		return nil, err
	}

	if role.Password == "" {
		return &role, nil
	}
	if role.SkipPasswordUpdate {
		log.Printf("Skipping password update for %q\n", role.Name)
		role.Password = ""
		return &role, nil
	}
	if verifyLogin(r.DbOpener.ConnectionUrl(), role.Name, role.Password) {
		log.Printf("Password for %q already valid, skipping update\n", role.Name)
		return &role, nil
	}
	if err := setPassword(db, role.Name, role.Password); err != nil {
		return nil, err
	}
	return &role, nil
}

type EnsurePasswordResult struct {
	Name    string `json:"name"`
	Changed bool   `json:"changed"`
}

// EnsurePassword alters the role's password only when a login attempt with it fails.
// It never creates the role; a missing role returns ErrRoleNotFound.
// Callers run it on every apply to heal roles created passwordless or reset outside their control.
func (r *Roles) EnsurePassword(name, password string) (*EnsurePasswordResult, error) {
	if name == "" || password == "" {
		return nil, fmt.Errorf("name and password are required")
	}
	if existing, err := r.Read(name); err != nil {
		return nil, err
	} else if existing == nil {
		return nil, fmt.Errorf("role %q: %w", name, ErrRoleNotFound)
	}

	if verifyLogin(r.DbOpener.ConnectionUrl(), name, password) {
		log.Printf("Password for %q already valid\n", name)
		return &EnsurePasswordResult{Name: name, Changed: false}, nil
	}

	db, err := r.DbOpener.OpenDatabase("")
	if err != nil {
		return nil, err
	}
	if err := setPassword(db, name, password); err != nil {
		return nil, err
	}
	return &EnsurePasswordResult{Name: name, Changed: true}, nil
}

// verifyLogin reports whether the role authenticates with the given password.
// Only a confirmed login skips the subsequent ALTER; any failure (bad password, pg_hba, network)
// falls through to setting the password, so behavior is never worse than altering unconditionally.
// The probe opens its own connection so a failed attempt never lands in the store's connection cache.
func verifyLogin(connUrl, name, password string) bool {
	probeUrl, err := UrlWithUserinfo(connUrl, name, password)
	if err != nil {
		return false
	}
	db, err := OpenDatabase(probeUrl, "")
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			log.Printf("Login check for %q failed (SQLSTATE %s)\n", name, pqErr.Code)
		} else {
			log.Printf("Login check for %q failed: %s\n", name, err)
		}
		return false
	}
	db.Close()
	return true
}

func setPassword(db *sql.DB, name, password string) error {
	log.Printf("Setting password for %q\n", name)
	updateSql := fmt.Sprintf(`ALTER ROLE %s WITH PASSWORD %s`, pq.QuoteIdentifier(name), pq.QuoteLiteral(password))
	if _, err := db.Exec(updateSql); err != nil {
		return fmt.Errorf("error setting password: %w", err)
	}
	log.Printf("Password set for %q\n", name)
	return nil
}

func (r *Roles) Drop(key string) (bool, error) {
	return true, nil
}

// updateAttributes adds any missing role attributes (e.g. REPLICATION) to an existing role
func (r *Roles) updateAttributes(db *sql.DB, key string, role Role) error {
	name := role.Name
	if name == "" {
		name = key
	}

	existing, err := r.Read(name)
	if err != nil {
		return err
	} else if existing == nil {
		return fmt.Errorf("cannot update role %q: role does not exist", name)
	}

	info, err := CalcDbConnectionInfo(db)
	if err != nil {
		return fmt.Errorf("error analyzing database: %w", err)
	}

	attrs := role.Attributes.alterSql(existing.Attributes, info.SupportedFeatures)
	if len(attrs) < 1 {
		return nil
	}

	log.Printf("Adding attributes [%s] to role %q\n", strings.Join(attrs, ", "), name)
	updateSql := fmt.Sprintf(`ALTER ROLE %s WITH %s`, pq.QuoteIdentifier(name), strings.Join(attrs, " "))
	if _, err := db.Exec(updateSql); err != nil {
		return fmt.Errorf("error altering role %q with [%s]: %w", name, strings.Join(attrs, ", "), err)
	}
	return nil
}

func (*Roles) generateCreateSql(role Role, features Features) string {
	b := bytes.NewBufferString("CREATE ROLE ")
	fmt.Fprint(b, pq.QuoteIdentifier(role.Name), " WITH LOGIN")
	if role.Attributes.CreateRole {
		fmt.Fprint(b, " CREATEROLE")
	}
	if role.Attributes.CreateDb {
		fmt.Fprint(b, " CREATEDB")
	}
	if role.Attributes.Replication && features.IsSupported(FeatureReplication) {
		fmt.Fprint(b, " REPLICATION")
	}
	if len(role.MemberOf) > 0 {
		safeRoleNames := make([]string, 0)
		for _, m := range role.MemberOf {
			safeRoleNames = append(safeRoleNames, pq.QuoteIdentifier(m))
		}
		fmt.Fprintf(b, " IN ROLE %s", strings.Join(safeRoleNames, ","))
	}
	if role.Password != "" {
		fmt.Fprint(b, " PASSWORD ")
		fmt.Fprint(b, pq.QuoteLiteral(role.Password))
	}
	return b.String()
}
