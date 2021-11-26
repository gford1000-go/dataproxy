package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"github.com/gford1000-go/logger"
)

// MockColumn defines how a column of data should be constructed, with boundaries
// for String, Int and Float64 types (which are only the types allowed)
type MockColumn struct {
	Column
	MaxLength       int     `json:"max_length"`
	IntLowerBound   int     `json:"int_lower_bound"`
	IntUpperBound   int     `json:"int_upper_bound"`
	FloatLowerBound float64 `json:"float_lower_bound"`
	FloatUpperBound float64 `json:"float_upper_bound"`
}

// MockCreateRequest specifies the generation of a mock set of records, split
// into pages according to the specified number of records per page
type MockCreateRequest struct {
	RecordCount    int          `json:"max_records"`
	Columns        []MockColumn `json:"columns"`
	RecordsPerPage int          `json:"records_per_page"`
}

// MockCreateResponse provides the details to be able to recover any of the pages
// created as a result of a MockCreateRequest being sent to the server
type MockCreateResponse struct {
	RequestHash string   `json:"hash"`
	PageTokens  []string `json:"tokens"`
}

// NewMockCreatRequestHandlerFactory returns a factory instance that manufactures Handlers
// which can generate mocked data to be added to the cache.
func NewMockCreatRequestHandlerFactory() HandlerFactory {
	return &mockCreatRequestHandlerFactory{}
}

type mockCreatRequestHandlerFactory struct {
}

func (f *mockCreatRequestHandlerFactory) New(pattern string, config *cacheConfig, requestID string) Handler {
	h := &mockCreatRequestHandler{}
	h.method = http.MethodPost
	h.config = config
	h.handler = h.handleCreatePages
	h.logger = logger.GetLogger()
	h.pattern = pattern
	h.requestID = requestID

	return h
}

type mockCreatRequestHandler struct {
	writeHandler
}

// handleCreatePages is invoked after the initial authorization and validation checks are completed,
// and creates pages of random data as defined by the request
func (m *mockCreatRequestHandler) handleCreatePages(w http.ResponseWriter, req *http.Request) {

	// Get the details of the requested page
	var p MockCreateRequest
	err := json.NewDecoder(req.Body).Decode(&p)
	if err != nil {
		returnError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate the data in the cache
	resp, err := m.createMockData(&p)
	if err != nil {
		returnError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return details of created pages
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (m *mockCreatRequestHandler) createMockData(req *MockCreateRequest) (*MockCreateResponse, error) {

	// Hash should be generated from the request; here is it just a UUID
	hash := NewUUID()

	// Only create a single page of data for now; token is a UUID
	curPageToken := NewUUID()

	cols := []Column{}
	for _, col := range req.Columns {
		cols = append(cols, Column{Name: col.Name, Type: col.Type})
	}

	resp := &MockCreateResponse{
		RequestHash: hash,
		PageTokens:  []string{curPageToken},
	}

	remainingRecords := req.RecordCount
	records := [][]string{}

	for remainingRecords > 0 {

		for len(records) < req.RecordsPerPage && remainingRecords > 0 {

			records = append(records, m.createRecord(req.Columns))

			remainingRecords--
		}

		if remainingRecords > 0 {
			nextPageToken := NewUUID()

			err := m.createPage(hash, curPageToken, nextPageToken, cols, records)
			if err != nil {
				return nil, err
			}

			resp.PageTokens = append(resp.PageTokens, nextPageToken)
			curPageToken = nextPageToken
			records = [][]string{}
		}

	}

	err := m.createPage(hash, curPageToken, "", cols, records)
	if err != nil {
		return nil, err
	}

	m.Debug("Hash: %v, Pages: %v", resp.RequestHash, resp.PageTokens)
	return resp, nil
}

func (m *mockCreatRequestHandler) createRandomString(maxLength int) string {
	available := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz012346789"

	length := rand.Intn(maxLength)

	ret := ""
	for i := 0; i < length; i++ {
		c := rand.Intn(len(available))
		ret = ret + available[c:c+1]
	}
	return ret
}

func (m *mockCreatRequestHandler) createRandomInt(lowerBound, upperBound int) int {
	if lowerBound == upperBound {
		return lowerBound
	}
	if lowerBound > upperBound {
		return m.createRandomInt(upperBound, lowerBound)
	}
	if lowerBound < 0 && upperBound < 0 {
		return -m.createRandomInt(-upperBound, -lowerBound)
	}
	return rand.Intn(upperBound-lowerBound) + lowerBound
}

func (m *mockCreatRequestHandler) createRandomFloat64(lowerBound, upperBound float64) float64 {
	if lowerBound == upperBound {
		return lowerBound
	}
	if lowerBound > upperBound {
		return m.createRandomFloat64(upperBound, lowerBound)
	}
	if lowerBound < 0 && upperBound < 0 {
		return -m.createRandomFloat64(-upperBound, -lowerBound)
	}
	return rand.Float64()*(upperBound-lowerBound) + lowerBound
}

func (m *mockCreatRequestHandler) createRecord(cols []MockColumn) []string {

	var record []string = []string{}

	for _, col := range cols {
		switch strings.ToLower(col.Type) {
		case "string":
			record = append(record, m.createRandomString(col.MaxLength))
		case "int":
			record = append(record, fmt.Sprintf("%v", m.createRandomInt(col.IntLowerBound, col.IntUpperBound)))
		case "float":
			record = append(record, fmt.Sprintf("%v", m.createRandomFloat64(col.FloatLowerBound, col.FloatUpperBound)))
		default:
			record = append(record, "Unsupported Type")
		}
	}

	return record
}
