package postgresql

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/go-multierror/multierror"
	"github.com/lib/pq"
	"github.com/nullstone-io/go-rest-api"
	"log"
	"strings"
)

type Database struct {
	Name               string `json:"name"`
	Owner              string `json:"owner"`
	Template           string `json:"template"`
	Encoding           string `json:"encoding"`
	Collation          string `json:"collation"`
	LcCtype            string `json:"lcCtype"`
	TablespaceName     string `json:"tablespaceName"`
	ConnectionLimit    int    `json:"connectionLimit"`
	IsTemplate         bool   `json:"isTemplate"`
	DisableConnections bool   `json:"disableConnections"`
}

var _ rest.DataAccess[string, Database] = &Databases{}

type Databases struct {
	Db *sql.DB
}

func (d *Databases) ParseKey(val string) (string, error) {
	return val, nil
}

func (d *Databases) Read(key string) (*Database, error) {
	var owner string
	row := d.Db.QueryRow(`SELECT pg_catalog.pg_get_userbyid(d.datdba) from pg_database d WHERE datname=$1`, key)
	if err := row.Scan(&owner); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &Database{Name: key, Owner: owner}, nil
}

func (d *Databases) Exists(obj Database) (bool, error) {
	existing, err := d.Read(obj.Name)
	return existing != nil, err
}

func (d *Databases) Create(obj Database) (*Database, error) {
	info, err := CalcDbConnectionInfo(d.Db)
	if err != nil {
		return nil, fmt.Errorf("error analyzing existing databases: %w", err)
	}

	var grant Revoker = NoopRevoker{}
	if obj.Owner != "" && !info.IsSuperuser {
		var err error
		grant, err = GrantRoleMembership(d.Db, obj.Owner, info.CurrentUser)
		if err != nil {
			return nil, fmt.Errorf("error granting temporary membership: %w", err)
		}
	}

	sq := d.generateCreateSql(obj, info.SupportedFeatures)
	log.Printf("Creating database %q, assigning owner to service user %q\n", obj.Name, obj.Owner)
	errs := make([]error, 0)
	if _, err := d.Db.Exec(sq); err != nil {
		errs = append(errs, fmt.Errorf("error creating database %q: %w", obj.Name, err))
	}
	if err := grant.Revoke(d.Db); err != nil {
		errs = append(errs, fmt.Errorf("error revoking temporary membership: %w", err))
	}
	if len(errs) > 0 {
		return nil, multierror.New(errs)
	}
	return &obj, nil
}

func (d *Databases) Ensure(obj Database) (*Database, error) {
	if existing, err := d.Read(obj.Name); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}
	return d.Create(obj)
}

func (d *Databases) Update(key string, obj Database) (*Database, error) {
	return d.Read(key)
}

func (d *Databases) Drop(key string) (bool, error) {
	return true, nil
}

func (*Databases) generateCreateSql(d Database, features Features) string {
	b := bytes.NewBufferString("CREATE DATABASE ")
	fmt.Fprint(b, pq.QuoteIdentifier(d.Name))

	if d.Owner != "" {
		fmt.Fprint(b, " OWNER ", pq.QuoteIdentifier(d.Owner))
	}

	switch template := d.Template; {
	case strings.ToUpper(template) == "DEFAULT":
		fmt.Fprint(b, " TEMPLATE DEFAULT")
	case template != "":
		fmt.Fprint(b, " TEMPLATE ", pq.QuoteIdentifier(template))
	}

	switch encoding := d.Encoding; {
	case strings.ToUpper(encoding) == "DEFAULT":
		fmt.Fprint(b, " ENCODING DEFAULT")
	case encoding != "":
		fmt.Fprint(b, " ENCODING ", pq.QuoteLiteral(encoding))
	}

	// Don't specify LC_COLLATE if user didn't specify it
	// This will use the default one (usually the one defined in the template database)
	switch collation := d.Collation; {
	case strings.ToUpper(collation) == "DEFAULT":
		fmt.Fprint(b, " LC_COLLATE DEFAULT")
	case collation != "":
		fmt.Fprint(b, " LC_COLLATE ", pq.QuoteLiteral(collation))
	}

	// Don't specify LC_CTYPE if user didn't specify it
	// This will use the default one (usually the one defined in the template database)
	switch lcCtype := d.LcCtype; {
	case strings.ToUpper(lcCtype) == "DEFAULT":
		fmt.Fprint(b, " LC_CTYPE DEFAULT")
	case lcCtype != "":
		fmt.Fprint(b, " LC_CTYPE ", pq.QuoteLiteral(lcCtype))
	}

	switch tablespace := d.TablespaceName; {
	case strings.ToUpper(tablespace) == "DEFAULT":
		fmt.Fprint(b, " TABLESPACE DEFAULT")
	case tablespace != "":
		fmt.Fprint(b, " TABLESPACE ", pq.QuoteIdentifier(tablespace))
	}

	if features.IsSupported(FeatureDBAllowConnections) {
		fmt.Fprintf(b, " ALLOW_CONNECTIONS %t", !d.DisableConnections)
	}

	if d.ConnectionLimit > 0 {
		fmt.Fprint(b, " CONNECTION LIMIT ", d.ConnectionLimit)
	}

	if features.IsSupported(FeatureDBIsTemplate) {
		fmt.Fprint(b, " IS_TEMPLATE ", d.IsTemplate)
	}

	return b.String()
}
