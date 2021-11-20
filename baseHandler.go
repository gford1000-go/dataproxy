package main

import (
	"fmt"
	"net/http"
)

// baseHandler drives the handling of requests
type baseHandler struct {
	config *cacheConfig
	handler func(w http.ResponseWriter, req *http.Request)
	logger *logger
	method string
	pattern string
	requestID string
}

// Log ensures that the requestID is always applied to the logs
func (b *baseHandler) Log(msg string) {
	b.logger.Println(fmt.Sprintf("%v %v", b.requestID, msg))
}

// Process generates the response for a request
func (b *baseHandler) Process(w http.ResponseWriter, req *http.Request) {

	b.Log(fmt.Sprintf("Processing %v", b.pattern))
	defer b.Log(fmt.Sprintf("Completed %v", b.pattern))

	defer func() {
		if r := recover(); r != nil {
			b.Log(fmt.Sprintf("Processing error %v", r))
			returnError(w, "Request error", http.StatusBadRequest)
		}
	}()

	// Accept only requests of specified type
	if req.Method != b.method {
		returnError(w, fmt.Sprintf("Only %v methods are accepted", b.method), http.StatusMethodNotAllowed)
		return
	}

	// Always authorize the requestor
	if err := b.authorize(req); err != nil {
		returnError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Invoke the delegated handler
	b.handler(w, req)

}

// authorize provides a standard access point to validate
// requestor credentials
func (b *baseHandler) authorize(req *http.Request) error {
	return nil
}
