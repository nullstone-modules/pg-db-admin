package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

func op[T any](w http.ResponseWriter, req *http.Request, dbBroker DbBroker, fn func(db *sql.DB) (*T, error)) {
	db, err := sql.Open("postgres", dbBroker.ConnectionUrl())
	if err != nil {
		http.Error(w, fmt.Sprintf("error connecting to db: %s", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	result, err := fn(db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else if result == nil {
		http.NotFound(w, req)
	} else if raw, err := json.Marshal(result); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		if req.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(raw)
		}
	}
}
