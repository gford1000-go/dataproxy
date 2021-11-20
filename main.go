package main

import (
	"flag"
    "fmt"
    "net/http"
)

// postHandler creates a request handler that ensures consistent authorization and validation behaviour for POST requests
func postHandler(pattern string, config *cacheConfig, reqHandler func(w http.ResponseWriter, req *http.Request, config *cacheConfig)) func(w http.ResponseWriter, req *http.Request) {

	return func(w http.ResponseWriter, req *http.Request) {
		logger := GetLogger()
		requestID := NewUUID()
		logger.Printf(fmt.Sprintf("Processing %v (%v) ", pattern, requestID))
		defer logger.Printf(fmt.Sprintf("Completed %v (%v) ", pattern, requestID))

		defer func() {
			if r := recover(); r != nil {
				logger.Printf(fmt.Sprintf("Processing error %v (%v) ", r, requestID))
				returnError(w, "Request error", http.StatusBadRequest)
			}
		}()

		// Accept only POST requests
		if req.Method != http.MethodPost {
			returnError(w, "Only POST methods are accepted", http.StatusMethodNotAllowed)
			return	
		}

		// Always authorize the requestor
		if err := authorize(pattern, req); err != nil {
			returnError(w, err.Error(), http.StatusUnauthorized)
			return	
		}

		// Invoke the delegated handler
		reqHandler(w, req, config)
	}
}


// alive verifies the server is running
func alive(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "up\n")
}

func main() {

	port := flag.Int("port", 8080, "Port on which to listen")
	root := flag.String("cache", "/tmp", "Location of cache")
	encryptionKey := flag.String("key", "", "AES key for cache")
	salt := flag.String("salt", "", "Salt for cache filenames")
	logName := flag.String("log", "/tmp/dataproxy.log", "Log file name")

	flag.Parse()

	config := &serverConfig{
		port: *port,
		log: *logName,
		cache: &cacheConfig{
			root: *root,
			salt: []byte(*salt),
			key: []byte(*encryptionKey),
		},
	}

	logger := NewLogger(config.log)
	logger.Println(fmt.Sprintf("Starting on port %v", config.port))

	http.HandleFunc("/alive", alive)
	http.HandleFunc("/page", postHandler("/page", config.cache, handlePageRetrieval))
	http.HandleFunc("/request", postHandler("/request", config.cache, handleCreatePages))
	http.ListenAndServe(fmt.Sprintf(":%v", config.port), nil)
}