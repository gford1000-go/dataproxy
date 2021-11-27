package main

import (
	"bytes"
	"encoding/json"
)

// writeHandler extends baseHandler to provide standard support for writing
// pages to the cache
type writeHandler struct {
	baseHandler
}

// createPage creates a single page, generating the remaining records up to the page size
func (m *writeHandler) createPage(hash, pageToken, nextPageToken string, cols []Column, records [][]string) error {

	type Column struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Position int    `json:"position"`
	}

	type Header struct {
		Columns []Column `json:"columns"`
	}

	type Data struct {
		Header  Header     `json:"header"`
		Records [][]string `json:"records"`
	}

	type Meta struct {
		NextToken string `json:"next"`
	}

	type ResultSet struct {
		Meta Meta `json:"meta"`
		Data Data `json:"data"`
	}

	pageCols := []Column{}
	for offset, col := range cols {
		pageCols = append(pageCols, Column{
			Name:     col.Name,
			Type:     col.Type,
			Position: offset,
		})
	}

	// Create page of data
	var page ResultSet = ResultSet{
		Meta: Meta{NextToken: nextPageToken},
		Data: Data{
			Header:  Header{Columns: pageCols},
			Records: records,
		},
	}

	info := &pageInfo{
		hash:  hash,
		token: pageToken,
	}

	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(page)

	err := m.writePage(buf.Bytes(), info)
	if err != nil {
		return err
	}

	return nil
}
