package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// PageRequest is the expected request body to identify a 
// page to be returned
type PageRequest struct {
	RequestHash string `json:"hash"`
	PageToken string `json:"token"`
}

// handlePageRetrieval is invoked after the initial authorization and validation checks are completed
func handlePageRetrieval(w http.ResponseWriter, req *http.Request, config *cacheConfig) {

	// Validate the content type requested
	reqSupportableTypes, allSupportedTypes := getRequestSupportedTypes(req)
	if len(reqSupportableTypes) == 0 {
		returnError(w, fmt.Sprintf("Supported content types are: %s", strings.Join(allSupportedTypes, ", ")), http.StatusUnsupportedMediaType)
		return
	}

	// Get the details of the requested page
	var p PageRequest
	err := json.NewDecoder(req.Body).Decode(&p)
	if err != nil {
		returnError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Retrieve page from cache
	info := &pageInfo{
		hash: p.RequestHash,
		token: p.PageToken,
		types: reqSupportableTypes,
		gzip: false,
	}
	b, err := getPage(info, config)
	if err != nil {
		returnError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return page
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}
