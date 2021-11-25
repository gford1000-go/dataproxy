package main

import (
	"errors"
)

// getJSONPage returns JSON
func getJSONPage(b []byte, info *pageInfo) (page []byte, err error) {
	return b, nil
}

// returnProcessingMap defines how a request will be handled, based on
// the first context type from the requestor that matched those that
// the server can provide
var returnProcessingMap map[string]func(b []byte, info *pageInfo) (page []byte, err error)

func init() {
	returnProcessingMap = map[string]func(b []byte, info *pageInfo) (page []byte, err error){
		"application/json": getJSONPage,
	}
}

// getPage identifies the handling function based on type
func (p *pageRequestHandler) getPage(info *pageInfo) (page []byte, err error) {

	b, err := p.retrievePage(info)
	if err != nil {
		return nil, err
	}

	if f, ok := returnProcessingMap[info.types[0]]; ok {
		return f(b, info)
	}

	return nil, errors.New("unexpected error handling page")
}
