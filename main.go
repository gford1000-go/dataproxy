package main

import (
	"crypto/aes"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gford1000-go/logger"
)

// Handler describes the interface to handle a request
type Handler interface {
	Process(w http.ResponseWriter, req *http.Request)
	Info(format string, a ...interface{})
	Debug(format string, a ...interface{})
	Error(format string, a ...interface{})
	Warn(format string, a ...interface{})
}

// HandlerFactory provides an instance creation method for a Handler
type HandlerFactory interface {
	New(pattern string, config *cacheConfig, requestID string) Handler
}

// postHandler creates a request handler that ensures consistent authorization and validation behaviour for POST requests
func postHandler(pattern string, config *cacheConfig, factory HandlerFactory) func(w http.ResponseWriter, req *http.Request) {

	return func(w http.ResponseWriter, req *http.Request) {
		// Create a new handler instance for each request, with a unique identifier
		handler := factory.New(pattern, config, NewUUID())
		handler.Process(w, req)
	}
}

// alive verifies the server is running
func alive(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "up\n")
}

// setupCloseHandler captures CTRL-C events
func setupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal")
		os.Exit(0)
	}()
}

func main() {

	port := flag.Int("port", 8080, "Port on which to listen")
	root := flag.String("cache", "/tmp", "Location of cache")
	encryptionKey := flag.String("key", "", "AES key for cache")
	salt := flag.String("salt", "", "Salt for cache filenames")
	logName := flag.String("log", "/tmp/dataproxy.log", "Log file name")
	useCompression := flag.Bool("zip", false, "If present, then cache files are compressed prior to saving")

	flag.Parse()

	config := &serverConfig{
		port: *port,
		log:  *logName,
		cache: &cacheConfig{
			root:           *root,
			salt:           []byte(*salt),
			useCompression: *useCompression,
		},
	}

	key := []byte(*encryptionKey)
	if len(key) > 0 {
		c, err := aes.NewCipher(key)
		if err != nil {
			panic(fmt.Sprintf("Error creating Cipher - %v", err))
		}
		config.cache.cipher = c
	}

	setupCloseHandler()

	log, _ := logger.NewFileLogger(config.log, logger.All, "DataProxy ")
	log(logger.Info, "", "Starting on port %v", config.port)

	http.HandleFunc("/alive", alive)
	http.HandleFunc("/page", postHandler("/page", config.cache, NewPageRequestHandlerFactory()))
	http.HandleFunc("/create", postHandler("/create", config.cache, NewMockCreatRequestHandlerFactory()))
	http.HandleFunc("/existing", postHandler("/existing", config.cache, NewExistingRequestHandlerFactory()))
	http.ListenAndServe(fmt.Sprintf(":%v", config.port), nil)
}
