package postgresql

import (
	"bytes"
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"strconv"
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

func (d Database) Create(conn *pgx.Conn, info DbInfo) (err error) {
	err = nil

	if d.Owner != "" && !info.IsSuperuser {
		tempMembership := TempRoleMembership{
			Role:        d.Owner,
			CurrentUser: info.CurrentUser,
		}
		var grant *TempGrant
		grant, err = tempMembership.Grant(conn)
		if grant != nil {
			defer func() {
				err = grant.Revoke(conn)
			}()
		}
	}

	sql, args := d.generateCreateSql(info.SupportedFeatures)
	fmt.Printf("Creating database %q, assigning owner to service user %q\n", d.Name, d.Owner)
	if _, err := conn.Exec(context.Background(), sql, args...); err != nil {
		return fmt.Errorf("error creating database %q: %w", d.Name, err)
	}

	// err can be set by defer
	return
}

func (d Database) generateCreateSql(features Features) (string, []interface{}) {
	args := make([]interface{}, 0)

	b := bytes.NewBufferString("CREATE DATABASE ")
	nameIdentifier := pgx.Identifier{d.Name}
	fmt.Fprint(b, nameIdentifier.Sanitize())

	if d.Owner != "" {
		fmt.Fprint(b, " OWNER $", strconv.Itoa(len(args)+1))
		args = append(args, d.Owner)
	}

	switch template := d.Template; {
	case strings.ToUpper(template) == "DEFAULT":
		fmt.Fprint(b, " TEMPLATE DEFAULT")
	case template != "":
		fmt.Fprint(b, " TEMPLATE ")
		fmt.Fprint(b, "$", strconv.Itoa(len(args)+1))
		args = append(args, template)
	}

	switch encoding := d.Encoding; {
	case strings.ToUpper(encoding) == "DEFAULT":
		fmt.Fprint(b, " ENCODING DEFAULT")
	case encoding != "":
		fmt.Fprint(b, " ENCODING ")
		fmt.Fprint(b, "$", strconv.Itoa(len(args)+1))
		args = append(args, encoding)
	}

	// Don't specify LC_COLLATE if user didn't specify it
	// This will use the default one (usually the one defined in the template database)
	switch collation := d.Collation; {
	case strings.ToUpper(collation) == "DEFAULT":
		fmt.Fprint(b, " LC_COLLATE DEFAULT")
	case collation != "":
		fmt.Fprint(b, " LC_COLLATE ")
		fmt.Fprint(b, "$", strconv.Itoa(len(args)+1))
		args = append(args, collation)
	}

	// Don't specify LC_CTYPE if user didn't specify it
	// This will use the default one (usually the one defined in the template database)
	switch lcCtype := d.LcCtype; {
	case strings.ToUpper(lcCtype) == "DEFAULT":
		fmt.Fprint(b, " LC_CTYPE DEFAULT")
	case lcCtype != "":
		fmt.Fprint(b, " LC_CTYPE ")
		fmt.Fprint(b, "$", strconv.Itoa(len(args)+1))
		args = append(args, lcCtype)
	}

	switch tablespace := d.TablespaceName; {
	case strings.ToUpper(tablespace) == "DEFAULT":
		fmt.Fprint(b, " TABLESPACE DEFAULT")
	case tablespace != "":
		fmt.Fprint(b, " TABLESPACE ")
		fmt.Fprint(b, "$", strconv.Itoa(len(args)+1))
		args = append(args, tablespace)
	}

	if features.IsSupported(FeatureDBAllowConnections) {
		fmt.Fprint(b, " ALLOW_CONNECTIONS ", d.AllowConnections)
	}

	fmt.Fprint(b, " CONNECTION LIMIT ", d.ConnectionLimit)

	if features.IsSupported(FeatureDBIsTemplate) {
		fmt.Fprint(b, " IS_TEMPLATE ", d.IsTemplate)
	}

	return b.String(), args
}

func (d Database) Update() error {
	return nil
}

func (d Database) Drop() error {
	return nil
}
