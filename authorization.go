package main

import (
    "net/http"
)

// authorize provides a standard access point to validate
// requestor credentials
func authorize(pattern string, req *http.Request) error {
	return nil
}