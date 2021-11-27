package main

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"os"
)

// Column specifies a column of data in the file
type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ExistingRequest specifies the caching of a specific CSV file at the given location, split
// into pages according to the specified number of records per page
type ExistingRequest struct {
	CSVFileName    string   `json:"file_name"`
	Columns        []Column `json:"columns"`
	RecordsPerPage int      `json:"records_per_page"`
}

// ExistingResponse provides the details to be able to recover any of the pages
// created as a result of a MockCreateRequest being sent to the server
type ExistingResponse struct {
	RequestHash string   `json:"hash"`
	PageTokens  []string `json:"tokens"`
}

// NewExistingRequestHandlerFactory returns a factory instance that manufactures Handlers
// which can cache data from an existing file.
func NewExistingRequestHandlerFactory() HandlerFactory {
	return &existingFileRequestHandlerFactory{}
}

type existingFileRequestHandlerFactory struct {
}

func (f *existingFileRequestHandlerFactory) New(pattern string, config *cacheConfig, requestID string) Handler {
	h := &existingFileRequestHandler{}
	h.method = http.MethodPost
	h.config = config
	h.handler = h.handleCreatePages
	h.pattern = pattern
	h.requestID = requestID

	return h
}

type existingFileRequestHandler struct {
	writeHandler
}

// handleCreatePages is invoked after the initial authorization and validation checks are completed,
// and creates caches pages of the specified file.
func (m *existingFileRequestHandler) handleCreatePages(w http.ResponseWriter, req *http.Request) {

	// Get the details of the file to be cached
	var p ExistingRequest
	err := json.NewDecoder(req.Body).Decode(&p)
	if err != nil {
		m.Error("%v", err)
		returnError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate the data in the cache
	resp, err := m.cacheData(&p)
	if err != nil {
		returnError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return details of created pages
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (m *existingFileRequestHandler) cacheData(req *ExistingRequest) (*ExistingResponse, error) {

	// Hash should be generated from the request; here is it just a UUID
	hash := NewUUID()

	// Only create a single page of data for now; token is a UUID
	curPageToken := NewUUID()

	resp := &ExistingResponse{
		RequestHash: hash,
		PageTokens:  []string{curPageToken},
	}

	file, err := os.Open(req.CSVFileName)
	if err != nil {
		m.Error("%v", err)
		return nil, err
	}
	defer file.Close()

	csvReader := csv.NewReader(file)

	records := [][]string{}

endOfFile:
	for {

		for len(records) <= req.RecordsPerPage {

			record, err := csvReader.Read()
			if err == io.EOF {
				break endOfFile
			}

			if err != nil {
				m.Error("error reading file record: %v", err)
				return nil, err
			}

			records = append(records, record)
		}

		nextPageToken := NewUUID()

		// Asynchronous write now that we have the data
		go m.createPage(hash, curPageToken, nextPageToken, req.Columns, records[0:req.RecordsPerPage])

		resp.PageTokens = append(resp.PageTokens, nextPageToken)
		curPageToken = nextPageToken
		records = [][]string{records[req.RecordsPerPage]}
	}

	err = m.createPage(hash, curPageToken, "", req.Columns, records)
	if err != nil {
		return nil, err
	}

	m.Debug("Hash: %v, Pages: %v", resp.RequestHash, resp.PageTokens)
	return resp, nil
}
