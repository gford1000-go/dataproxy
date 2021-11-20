package main 

import (
	"encoding/json"
	"net/http"
)

type errResponse struct {
	Error string `json:"error"`
}

// returnError ensures a standard JSON object is returned with the error details
func returnError(w http.ResponseWriter, errMsg string, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	e := errResponse{Error: errMsg }
	json.NewEncoder(w).Encode(e)
}
