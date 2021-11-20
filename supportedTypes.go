package main

import (
    "net/http"
	"strings"
)

// getSupportedContentTypes lists the supported format of the 
// cached pages.  Gives scope to support for other context types
func getSupportedContentTypes() []string {
	return []string{ "application/json" }
}

// getRequestSupportedTypes examines the request headers to determine
// what the client is prepared to process, and compares that to the 
// types that the server can provide.
func getRequestSupportedTypes(req *http.Request) ([]string, []string) {
	var requestedTypes []string = []string{}
	for _, t := range req.Header.Values("Content-Type") {

		for _, st := range getSupportedContentTypes() {
			t = strings.ToLower(strings.TrimSpace(t))
			if t == strings.ToLower(st) {
				requestedTypes = append(requestedTypes, t)
			}
		}
	}

	return requestedTypes, getSupportedContentTypes()
}
