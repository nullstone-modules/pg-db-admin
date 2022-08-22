package api

import (
	"database/sql"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"net/http"
)

type Databases struct {
	DbBroker DbBroker
}

type databasePayload struct {
	postgresql.Database
	UseExisting bool `json:"useExisting"`
}

func (d Databases) Create(w http.ResponseWriter, req *http.Request) {
	payload, ok := DecodeBody[databasePayload](w, req)
	if !ok {
		return
	}

	op(w, req, d.DbBroker, func(db *sql.DB) (*postgresql.Database, error) {
		dbInfo, err := postgresql.CalcDbConnectionInfo(db)
		if err != nil {
			return nil, fmt.Errorf("error introspecting postgres cluster: %s", err)
		}

		database := payload.Database
		if payload.UseExisting {
			return &database, database.Ensure(db, *dbInfo)
		}
		return &database, database.Create(db, *dbInfo)
	})
}

func (d Databases) Get(w http.ResponseWriter, req *http.Request) {
	op(w, req, d.DbBroker, func(db *sql.DB) (*postgresql.Database, error) {
		vars := mux.Vars(req)
		data := postgresql.Database{Name: vars["name"]}
		if err := data.Read(db); err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}
		return &data, nil
	})
}

func (d Databases) Update(w http.ResponseWriter, req *http.Request) {
	payload, ok := DecodeBody[databasePayload](w, req)
	if !ok {
		return
	}

	op(w, req, d.DbBroker, func(db *sql.DB) (*postgresql.Database, error) {
		vars := mux.Vars(req)
		data := payload.Database
		data.Name = vars["name"]
		if ok, err := data.Exists(db); err != nil {
			return nil, err
		} else if !ok {
			return nil, nil
		}
		return &data, data.Update(db)
	})
}

func (d Databases) Delete(w http.ResponseWriter, req *http.Request) {
	payload, ok := DecodeBody[struct {
		SkipDestroy bool `json:"skipDestroy"`
	}](w, req)
	if !ok {
		return
	}

	if payload.SkipDestroy {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	op(w, req, d.DbBroker, func(db *sql.DB) (*postgresql.Database, error) {
		vars := mux.Vars(req)
		data := postgresql.Database{Name: vars["name"]}
		if err := data.Drop(db); err != nil {
			return nil, err
		}
		return &data, data.Read(db)
	})
}
