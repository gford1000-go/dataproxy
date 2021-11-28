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

// firstPageInfo provides the details needed to return the first page of results
type firstPageInfo struct {
	requestHash string
	token       string
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

	// Attempt to open the file
	file, err := os.Open(p.CSVFileName)
	if err != nil {
		m.Error("%v", err)
		returnError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Hash should be generated from the request; here is it just a UUID
	hash := NewUUID()

	// token to the first page of data
	firstPageToken := NewUUID()

	// Asynchronously generate the page data in the cache
	m.Debug("Starting page generation - hash: %v, first page: %v", hash, firstPageToken)
	go m.cacheData(hash, firstPageToken, &p, file)

	// Create initial response, which is empty and points to the first page
	m.Debug("Creating empty first page")
	b := m.createPageBytes(firstPageToken, p.Columns, [][]string{})

	// Return the first page
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

// cacheData reads records from the file, creating cache pages until EOF is reached
func (m *existingFileRequestHandler) cacheData(hash, firstPageToken string, req *ExistingRequest, file *os.File) error {
	// Ensure the file is always closed
	defer file.Close()

	// Current page is initially the first page
	curPageToken := firstPageToken

	// CSV based file
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
				return err
			}

			records = append(records, record)
		}

		// Having read 1 more record than a page should have, we
		// know that another page is required, so create its token
		nextPageToken := NewUUID()

		// Asynchronous write now that we have the data
		go m.createPage(hash, curPageToken, nextPageToken, req.Columns, records[0:req.RecordsPerPage])

		// Reset for next page
		curPageToken = nextPageToken
		records = [][]string{records[req.RecordsPerPage]}
	}

	// Final page - identified by an empty token
	err := m.createPage(hash, curPageToken, "", req.Columns, records)
	if err != nil {
		return err
	}

	return nil
}
