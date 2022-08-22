package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func DecodeBody[T any](w http.ResponseWriter, req *http.Request) (T, bool) {
	decoder := json.NewDecoder(req.Body)
	defer req.Body.Close()

	var payload T
	if err := decoder.Decode(&payload); err != nil {
		http.Error(w, fmt.Sprintf("invalid payload: %s", err), http.StatusBadRequest)
		return payload, false
	}
	return payload, true
}
