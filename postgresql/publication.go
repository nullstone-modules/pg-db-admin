package postgresql

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.com/nullstone-io/go-rest-api"
	"log"
	"sort"
	"strings"
)

// Publication defines a postgres publication -- the set of tables that a logical replication consumer receives
//
// A publication selects its tables in one of three ways:
//   - AllTables: `FOR ALL TABLES` -- every table in every schema, including tables created later
//   - Schemas:   `FOR TABLES IN SCHEMA <schema>` -- every table in the schema, including tables created later (postgres 15+)
//   - Tables:    `FOR TABLE <schema>.<table>` -- only the listed tables
//
// Schemas and Tables can be combined; AllTables cannot be combined with either.
type Publication struct {
	Name     string `json:"name"`
	Database string `json:"database"`

	AllTables bool               `json:"allTables"`
	Schemas   []string           `json:"schemas"`
	Tables    []PublicationTable `json:"tables"`

	// Do not error if trying to create a publication that already exists
	// Instead, reconcile the existing publication's membership and return
	UseExisting bool `json:"useExisting"`
}

func (p Publication) Key() PublicationKey {
	return PublicationKey{
		Database: p.Database,
		Name:     p.Name,
	}
}

type PublicationKey struct {
	Database string
	Name     string
}

type PublicationTable struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
}

func (t PublicationTable) String() string {
	if t.Schema == "" {
		return t.Name
	}
	return fmt.Sprintf("%s.%s", t.Schema, t.Name)
}

func (t PublicationTable) quoted() string {
	if t.Schema == "" {
		return pq.QuoteIdentifier(t.Name)
	}
	return fmt.Sprintf("%s.%s", pq.QuoteIdentifier(t.Schema), pq.QuoteIdentifier(t.Name))
}

var _ rest.DataAccess[PublicationKey, Publication] = &Publications{}

type Publications struct {
	DbOpener DbOpener
}

func (p *Publications) Create(obj Publication) (*Publication, error) {
	if obj.UseExisting {
		if existing, err := p.Read(obj.Key()); err != nil {
			return nil, err
		} else if existing != nil {
			log.Printf("[Create] Publication %q already exists, updating...\n", obj.Name)
			return p.Update(obj.Key(), obj)
		}
	}

	db, err := p.DbOpener.OpenDatabase(obj.Database)
	if err != nil {
		return nil, err
	}

	info, err := CalcDbConnectionInfo(db)
	if err != nil {
		return nil, fmt.Errorf("error analyzing database: %w", err)
	}
	if err := obj.verifySupported(info.SupportedFeatures); err != nil {
		return nil, err
	}

	log.Printf("Creating publication %q in database %q\n", obj.Name, obj.Database)
	if _, err := db.Exec(p.generateCreateSql(obj)); err != nil {
		return nil, fmt.Errorf("error creating publication %q: %w", obj.Name, err)
	}
	return &obj, nil
}

func (p *Publications) Read(key PublicationKey) (*Publication, error) {
	db, err := p.DbOpener.OpenDatabase(key.Database)
	if err != nil {
		return nil, err
	}

	obj := Publication{Name: key.Name, Database: key.Database}
	row := db.QueryRow(`SELECT puballtables FROM pg_publication WHERE pubname = $1`, key.Name)
	if err := row.Scan(&obj.AllTables); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	info, err := CalcDbConnectionInfo(db)
	if err != nil {
		return nil, fmt.Errorf("error analyzing database: %w", err)
	}

	if info.SupportedFeatures.IsSupported(FeaturePublicationSchema) {
		if obj.Schemas, err = p.readSchemas(db, key.Name); err != nil {
			return nil, err
		}
	}
	if obj.Tables, err = p.readTables(db, key.Name); err != nil {
		return nil, err
	}
	return &obj, nil
}

// Update reconciles the publication's membership using `ALTER PUBLICATION ... ADD|DROP`
// The publication is never dropped and recreated: recreating invalidates any active subscription
func (p *Publications) Update(key PublicationKey, obj Publication) (*Publication, error) {
	obj.Name, obj.Database = key.Name, key.Database

	db, err := p.DbOpener.OpenDatabase(obj.Database)
	if err != nil {
		return nil, err
	}

	info, err := CalcDbConnectionInfo(db)
	if err != nil {
		return nil, fmt.Errorf("error analyzing database: %w", err)
	}
	if err := obj.verifySupported(info.SupportedFeatures); err != nil {
		return nil, err
	}

	existing, err := p.Read(key)
	if err != nil {
		return nil, err
	} else if existing == nil {
		return p.Create(obj)
	}

	if existing.AllTables != obj.AllTables {
		// Postgres has no way to convert a `FOR ALL TABLES` publication into a table/schema list (or back)
		// Recreating would invalidate an active subscription, so this needs an operator decision
		return nil, fmt.Errorf("publication %q in database %q is %s, but is configured to be %s; "+
			"postgres cannot alter this in place -- drop the publication (and any subscription reading it) to change it",
			obj.Name, obj.Database, describeAllTables(existing.AllTables), describeAllTables(obj.AllTables))
	}

	for _, sq := range p.generateReconcileSql(obj, *existing) {
		log.Printf("Reconciling publication %q: %s\n", obj.Name, sq)
		if _, err := db.Exec(sq); err != nil {
			return nil, fmt.Errorf("error altering publication %q: %w", obj.Name, err)
		}
	}
	return &obj, nil
}

func (p *Publications) Drop(key PublicationKey) (bool, error) {
	db, err := p.DbOpener.OpenDatabase(key.Database)
	if err != nil {
		return false, err
	}

	log.Printf("Dropping publication %q in database %q\n", key.Name, key.Database)
	sq := fmt.Sprintf(`DROP PUBLICATION IF EXISTS %s`, pq.QuoteIdentifier(key.Name))
	if _, err := db.Exec(sq); err != nil {
		return false, fmt.Errorf("error dropping publication %q: %w", key.Name, err)
	}
	return true, nil
}

func (p *Publications) readSchemas(db *sql.DB, name string) ([]string, error) {
	sq := `SELECT n.nspname
FROM pg_publication_namespace pn
	JOIN pg_namespace n ON n.oid = pn.pnnspid
	JOIN pg_publication p ON p.oid = pn.pnpubid
WHERE p.pubname = $1`
	rows, err := db.Query(sq, name)
	if err != nil {
		return nil, fmt.Errorf("error reading schemas for publication %q: %w", name, err)
	}
	defer rows.Close()

	schemas := make([]string, 0)
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, err
		}
		schemas = append(schemas, schema)
	}
	return schemas, rows.Err()
}

// readTables reads the tables explicitly added to the publication
// This intentionally reads pg_publication_rel instead of the pg_publication_tables view:
// the view expands `FOR ALL TABLES` and `FOR TABLES IN SCHEMA` into their current members,
// which would make every schema-based publication look like it needs its tables dropped
func (p *Publications) readTables(db *sql.DB, name string) ([]PublicationTable, error) {
	sq := `SELECT n.nspname, c.relname
FROM pg_publication_rel pr
	JOIN pg_class c ON c.oid = pr.prrelid
	JOIN pg_namespace n ON n.oid = c.relnamespace
	JOIN pg_publication p ON p.oid = pr.prpubid
WHERE p.pubname = $1`
	rows, err := db.Query(sq, name)
	if err != nil {
		return nil, fmt.Errorf("error reading tables for publication %q: %w", name, err)
	}
	defer rows.Close()

	tables := make([]PublicationTable, 0)
	for rows.Next() {
		var table PublicationTable
		if err := rows.Scan(&table.Schema, &table.Name); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, rows.Err()
}

func (p Publication) verifySupported(features Features) error {
	if !features.IsSupported(FeaturePublication) {
		return fmt.Errorf("publications require postgres 10 or later")
	}
	if len(p.Schemas) > 0 && !features.IsSupported(FeaturePublicationSchema) {
		return fmt.Errorf("publishing an entire schema (`FOR TABLES IN SCHEMA`) requires postgres 15 or later; "+
			"list the tables explicitly instead (publication %q)", p.Name)
	}
	if p.AllTables && (len(p.Schemas) > 0 || len(p.Tables) > 0) {
		return fmt.Errorf("publication %q cannot set allTables together with schemas or tables", p.Name)
	}
	return nil
}

func (*Publications) generateCreateSql(p Publication) string {
	b := bytes.NewBufferString("CREATE PUBLICATION ")
	fmt.Fprint(b, pq.QuoteIdentifier(p.Name))

	switch {
	case p.AllTables:
		fmt.Fprint(b, " FOR ALL TABLES")
	case len(p.Tables) > 0 || len(p.Schemas) > 0:
		// Tables are emitted before schemas to match the syntax documented by postgres
		objects := make([]string, 0)
		if len(p.Tables) > 0 {
			quoted := make([]string, 0, len(p.Tables))
			for _, t := range p.Tables {
				quoted = append(quoted, t.quoted())
			}
			objects = append(objects, fmt.Sprintf("TABLE %s", strings.Join(quoted, ", ")))
		}
		if len(p.Schemas) > 0 {
			quoted := make([]string, 0, len(p.Schemas))
			for _, s := range p.Schemas {
				quoted = append(quoted, pq.QuoteIdentifier(s))
			}
			objects = append(objects, fmt.Sprintf("TABLES IN SCHEMA %s", strings.Join(quoted, ", ")))
		}
		fmt.Fprintf(b, " FOR %s", strings.Join(objects, ", "))
	}

	return b.String()
}

func (*Publications) generateReconcileSql(desired, existing Publication) []string {
	quotedName := pq.QuoteIdentifier(desired.Name)
	stmts := make([]string, 0)

	addSchemas, dropSchemas := diffStrings(desired.Schemas, existing.Schemas)
	for _, s := range addSchemas {
		stmts = append(stmts, fmt.Sprintf(`ALTER PUBLICATION %s ADD TABLES IN SCHEMA %s`, quotedName, pq.QuoteIdentifier(s)))
	}
	for _, s := range dropSchemas {
		stmts = append(stmts, fmt.Sprintf(`ALTER PUBLICATION %s DROP TABLES IN SCHEMA %s`, quotedName, pq.QuoteIdentifier(s)))
	}

	desiredTables, existingTables := map[string]PublicationTable{}, map[string]PublicationTable{}
	for _, t := range desired.Tables {
		desiredTables[t.String()] = t
	}
	for _, t := range existing.Tables {
		existingTables[t.String()] = t
	}
	addTables, dropTables := diffStrings(keys(desiredTables), keys(existingTables))
	for _, t := range addTables {
		stmts = append(stmts, fmt.Sprintf(`ALTER PUBLICATION %s ADD TABLE %s`, quotedName, desiredTables[t].quoted()))
	}
	for _, t := range dropTables {
		stmts = append(stmts, fmt.Sprintf(`ALTER PUBLICATION %s DROP TABLE %s`, quotedName, existingTables[t].quoted()))
	}

	return stmts
}

func describeAllTables(allTables bool) string {
	if allTables {
		return "`FOR ALL TABLES`"
	}
	return "a list of schemas/tables"
}

// diffStrings returns the values that need to be added to (and dropped from) existing to match desired
func diffStrings(desired, existing []string) (add []string, drop []string) {
	desiredSet, existingSet := map[string]bool{}, map[string]bool{}
	for _, cur := range desired {
		desiredSet[cur] = true
	}
	for _, cur := range existing {
		existingSet[cur] = true
	}

	add, drop = make([]string, 0), make([]string, 0)
	for cur := range desiredSet {
		if !existingSet[cur] {
			add = append(add, cur)
		}
	}
	for cur := range existingSet {
		if !desiredSet[cur] {
			drop = append(drop, cur)
		}
	}
	// Sorted to keep the emitted sql deterministic
	sort.Strings(add)
	sort.Strings(drop)
	return add, drop
}

func keys[T any](m map[string]T) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}
