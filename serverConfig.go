package main

type cacheConfig struct {
	root string
	salt []byte
	key []byte
}

type serverConfig struct {
	port int
	log string
	cache *cacheConfig
}
