package postgresql

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"strings"
)

type Database struct {
	Name             string
	Owner            string
	Template         string
	Encoding         string
	Collation        string
	LcCtype          string
	TablespaceName   string
	ConnectionLimit  int
	IsTemplate       bool
	AllowConnections bool
}

func DefaultDatabase() Database {
	return Database{
		ConnectionLimit:  -1,
		AllowConnections: true,
	}
}

func (d Database) Create(db *sql.DB, info DbInfo) (err error) {
	err = nil

	if d.Owner != "" && !info.IsSuperuser {
		tempMembership := TempRoleMembership{
			Role:        d.Owner,
			CurrentUser: info.CurrentUser,
		}
		var grant *TempGrant
		grant, err = tempMembership.Grant(db)
		if grant != nil {
			defer func() {
				err = grant.Revoke(db)
			}()
		}
	}

	sq := d.generateCreateSql(info.SupportedFeatures)
	fmt.Printf("Creating database %q, assigning owner to service user %q\n", d.Name, d.Owner)
	if _, err := db.Exec(sq); err != nil {
		return fmt.Errorf("error creating database %q: %w", d.Name, err)
	}

	// err can be set by defer
	return
}

func (d Database) generateCreateSql(features Features) string {
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
		fmt.Fprint(b, " ALLOW_CONNECTIONS ", d.AllowConnections)
	}

	fmt.Fprint(b, " CONNECTION LIMIT ", d.ConnectionLimit)

	if features.IsSupported(FeatureDBIsTemplate) {
		fmt.Fprint(b, " IS_TEMPLATE ", d.IsTemplate)
	}

	return b.String()
}

func (d Database) Exists(db *sql.DB) (bool, error) {
	check := Database{Name: d.Name}
	if err := check.Read(db); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (d *Database) Read(db *sql.DB) error {
	var owner string
	row := db.QueryRow( `SELECT pg_catalog.pg_get_userbyid(d.datdba) from pg_database d WHERE datname=$1`, d.Name)
	if err := row.Scan(&owner); err != nil {
		return err
	}
	d.Owner = owner
	return nil
}

func (d Database) Update(db *sql.DB) error {
	return nil
}

func (d Database) Drop(db *sql.DB) error {
	return nil
}
