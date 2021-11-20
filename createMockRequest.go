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

// handleCreatePages is invoked after the initial authorization and validation checks are completed,
// and creates pages of random data as defined by the request
func handleCreatePages(w http.ResponseWriter, req *http.Request, config *cacheConfig) {

	// Get the details of the requested page
	var p MockCreateRequest
	err := json.NewDecoder(req.Body).Decode(&p)
	if err != nil {
		returnError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate the data in the cache
	resp, err := createMockData(&p, config)
	if err != nil {
		returnError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return details of created pages
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func createMockData(req *MockCreateRequest, config *cacheConfig) (*MockCreateResponse, error) {

	// Hash should be generated from the request; here is it just a UUID
	hash := NewUUID()

	// Only create a single page of data for now; token is a UUID
	token := NewUUID()

	resp := &MockCreateResponse{
		RequestHash: hash,
		PageTokens: []string{ token },
	}

	var remainingRecords int = req.RecordCount

	_, err := createPage(hash, token, req.Columns, remainingRecords, req.RecordsPerPage, config)

	if err != nil {
		return nil, err
	}

	fmt.Println(resp)
	return resp, nil
}

// createPage creates a single page, generating the remaining records up to the page size
func createPage(hash, token string, cols []MockColumn, remainingRecords, pageRecordCount int, config *cacheConfig) (int, error) {
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

	err := writePage(buf.Bytes(), info, config)
	if err != nil {
		return 0, err
	}

	return 0, nil
}


