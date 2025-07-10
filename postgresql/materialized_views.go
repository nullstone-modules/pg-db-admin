package postgresql

import (
	"database/sql"
	"errors"
	"github.com/lib/pq"
	"github.com/nullstone-io/go-rest-api"
	"log"
)

type MaterializedViewsGrant struct {
	Database string             `json:"database"`
	Role     string             `json:"role"`
	Views    []MaterializedView `json:"views"`
}

func (g MaterializedViewsGrant) Key() MaterializedViewsGrantKey {
	return MaterializedViewsGrantKey{
		Database: g.Database,
		Role:     g.Role,
	}
}

type MaterializedViewsGrantKey struct {
	Database string
	Role     string
}

var (
	_ rest.DataAccess[MaterializedViewsGrantKey, MaterializedViewsGrant] = &MaterializedViews{}
)

type MaterializedViews struct {
	DbOpener DbOpener
}

func (m MaterializedViews) Create(obj MaterializedViewsGrant) (*MaterializedViewsGrant, error) {
	return m.Update(obj.Key(), obj)
}

func (m MaterializedViews) Read(key MaterializedViewsGrantKey) (*MaterializedViewsGrant, error) {
	db, err := m.DbOpener.OpenDatabase(key.Database)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	views, err := m.getViews(db, key)
	if err != nil {
		return nil, err
	}

	return &MaterializedViewsGrant{
		Database: key.Database,
		Role:     key.Role,
		Views:    filterToViewsWithRefresh(views),
	}, nil
}

func (m MaterializedViews) Update(key MaterializedViewsGrantKey, obj MaterializedViewsGrant) (*MaterializedViewsGrant, error) {
	db, err := m.DbOpener.OpenDatabase(key.Database)
	if err != nil {
		return nil, err
	}

	existingViews, err := m.getViews(db, key)
	if err != nil {
		return nil, err
	}
	getExistingView := func(match MaterializedView) *MaterializedView {
		for _, existing := range existingViews {
			if match.IsSameObject(existing) {
				return &existing
			}
		}
		return nil
	}

	toGrant := obj.Views
	if len(obj.Views) == 1 && obj.Views[0].Name == "*" {
		toGrant = make([]MaterializedView, 0)
		for _, existing := range existingViews {
			toGrant = append(toGrant, MaterializedView{
				Schema:     existing.Schema,
				Name:       existing.Name,
				HasRefresh: true,
			})
		}
	}

	errs := make([]error, 0)
	for _, view := range toGrant {
		existing := getExistingView(view)
		if existing == nil {
			log.Printf("Cannot grant refresh on %s.%s, does not exist", view.Schema, view.Name)
		} else if !existing.HasRefresh {
			_, err := db.Exec("GRANT MAINTAIN ON " + pq.QuoteIdentifier(view.Schema) + "." + pq.QuoteIdentifier(view.Name) + " TO " + pq.QuoteIdentifier(key.Role))
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return &obj, nil
}

func (m MaterializedViews) Drop(key MaterializedViewsGrantKey) (bool, error) {
	return true, nil
}

func (m MaterializedViews) getViews(db *sql.DB, key MaterializedViewsGrantKey) ([]MaterializedView, error) {
	q := `
SELECT schemaname, matviewname,
	has_table_privilege($1, quote_ident(schemaname) || '.' || quote_ident(matviewname), 'MAINTAIN') AS has_refresh
FROM pg_matviews
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')`
	rows, err := db.Query(q, key.Role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]MaterializedView, 0)
	for rows.Next() {
		var mv MaterializedView
		if err := rows.Scan(&mv.Schema, &mv.Name, &mv.HasRefresh); err != nil {
			return nil, err
		}
		results = append(results, mv)
	}
	return results, nil
}

type MaterializedView struct {
	Schema     string `json:"schema"`
	Name       string `json:"name"`
	HasRefresh bool   `json:"-"`
}

func (m MaterializedView) IsSameObject(other MaterializedView) bool {
	schema := m.Schema
	if schema == "" {
		schema = "public"
	}
	otherSchema := other.Schema
	if otherSchema == "" {
		otherSchema = "public"
	}
	return schema == otherSchema && m.Name == other.Name
}

func filterToViewsWithRefresh(views []MaterializedView) []MaterializedView {
	hasAll := true
	result := make([]MaterializedView, 0)
	for _, view := range views {
		if !view.HasRefresh {
			hasAll = false
		} else {
			result = append(result, view)
		}
	}
	if hasAll {
		return []MaterializedView{{Name: "*"}}
	}
	return result
}
