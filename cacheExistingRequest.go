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

	// Channel allows the details of the first page to be returned
	c := make(chan *firstPageInfo, 1)

	// Asynchronously generate the page data in the cache
	m.Debug("Starting page generation")
	go m.cacheData(&p, file, c)

	// Wait for details of the first page to be returned
	m.Debug("Waiting for first page details")
	info := <-c
	m.Debug("Received first page details")

	// Retrieve the first page from the cache
	m.Debug("Retrieving first page")
	defer m.Debug("Retrieved first page")
	pi := &pageInfo{
		hash:  info.requestHash,
		token: info.token,
		types: []string{"application/json"},
		gzip:  false,
	}

	b, err := m.retrievePage(pi)
	if err != nil {
		defer m.Error("Error retrieving the first page")
		returnError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return the first page
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func (m *existingFileRequestHandler) cacheData(req *ExistingRequest, file *os.File, c chan *firstPageInfo) error {
	// Ensure the file is always closed
	defer file.Close()

	// Hash should be generated from the request; here is it just a UUID
	hash := NewUUID()

	// Only create a single page of data for now; token is a UUID
	curPageToken := NewUUID()

	csvReader := csv.NewReader(file)

	records := [][]string{}
	first := true

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

		nextPageToken := NewUUID()

		if first {
			// Synchronous write for the first page
			err := m.createPage(hash, curPageToken, "", req.Columns, records)
			if err != nil {
				return err
			}

			// Notify the first page details
			c <- &firstPageInfo{
				requestHash: hash,
				token:       curPageToken,
			}

			// End of first page processing
			first = false

		} else {
			// Asynchronous write now that we have the data
			go m.createPage(hash, curPageToken, nextPageToken, req.Columns, records[0:req.RecordsPerPage])
		}

		// Reset for next page
		curPageToken = nextPageToken
		records = [][]string{records[req.RecordsPerPage]}
	}

	// Final page
	err := m.createPage(hash, curPageToken, "", req.Columns, records)
	if err != nil {
		return err
	}

	// Final page might still have been the first page
	if first {
		c <- &firstPageInfo{
			requestHash: hash,
			token:       curPageToken,
		}
	}

	return nil
}
