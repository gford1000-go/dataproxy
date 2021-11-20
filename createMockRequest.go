package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type MockColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
	MaxLength int `json:"max_length"`
}

type MockCreateRequest struct {
	RecordCount int `json:"max_records"`
	Columns []MockColumn `json:"columns"`
	RecordsPerPage int `json:"records_per_page"`
}

type MockCreateResponse struct {
	RequestHash string `json:"hash"`
	PageTokens []string `json:"tokens"`
}

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
	h.logger = GetLogger()
	h.pattern = pattern
	h.requestID = requestID

	return h
}

type mockCreatRequestHandler struct {
	baseHandler
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
	token := NewUUID()

	resp := &MockCreateResponse{
		RequestHash: hash,
		PageTokens: []string{ token },
	}

	var remainingRecords int = req.RecordCount

	_, err := m.createPage(hash, token, req.Columns, remainingRecords, req.RecordsPerPage)

	if err != nil {
		return nil, err
	}

	fmt.Println(resp)
	return resp, nil
}

// createPage creates a single page, generating the remaining records up to the page size
func (m *mockCreatRequestHandler) createPage(hash, token string, cols []MockColumn, remainingRecords, pageRecordCount int) (int, error) {
	type Record struct {
		Cells []string `json:"cells"`
	}

	type Column struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Position int `json:"position"`
	}

	type Header struct {
		Columns []Column `json:"columns"`
	}

	type Data struct {
		Header Header `json:"header"`
		Records []Record `json:"records"`
	}

	type Meta struct {

	}

	type ResultSet struct {
		Meta Meta `json:"meta"`
		Data Data `json:"data"`
	}

	// Create mock data
	var page ResultSet = ResultSet{
		Data: Data{
			Header: Header{
				Columns: []Column{
					Column{
						Name: "colX",
						Type: "string",
						Position: 1,
					},
				},
			},
			Records: []Record{
				Record{
					Cells: []string{
						"Hello",
					},
				},
			},
		},
	}


	info := &pageInfo{
		hash: hash,
		token: token,
	}

	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(page)

	err := m.writePage(buf.Bytes(), info)
	if err != nil {
		return 0, err
	}

	return 0, nil
}


