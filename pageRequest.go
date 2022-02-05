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
	PageToken   string `json:"token"`
}

func NewPageRequestHandlerFactory() HandlerFactory {
	return &pageRequestHandlerFactory{}
}

type pageRequestHandlerFactory struct {
}

func (f *pageRequestHandlerFactory) New(pattern string, config *cacheConfig, requestID string) Handler {
	h := &pageRequestHandler{}
	h.method = http.MethodPost
	h.config = config
	h.handler = h.handlePageRetrieval
	h.pattern = pattern
	h.requestID = requestID

	return h
}

type pageRequestHandler struct {
	baseHandler
}

// handlePageRetrieval is invoked after the initial authorization and validation checks are completed
func (p *pageRequestHandler) handlePageRetrieval(w http.ResponseWriter, req *http.Request) {

	// Validate the content type requested
	reqSupportableTypes, allSupportedTypes := getRequestSupportedTypes(req)
	if len(reqSupportableTypes) == 0 {
		returnError(w, fmt.Sprintf("Supported content types are: %s", strings.Join(allSupportedTypes, ", ")), http.StatusUnsupportedMediaType)
		return
	}

	// Get the details of the requested page
	var pg PageRequest
	err := json.NewDecoder(req.Body).Decode(&pg)
	if err != nil {
		returnError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Retrieve page from cache
	info := &pageInfo{
		hash:           pg.RequestHash,
		token:          pg.PageToken,
		types:          reqSupportableTypes,
		useCompression: false,
	}
	b, err := p.getPage(info)
	if err != nil {
		returnError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return page
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}
