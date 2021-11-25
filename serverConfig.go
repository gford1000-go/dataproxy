package main

type cacheConfig struct {
	root string
	salt []byte
	key  []byte
	zip  bool
}

type serverConfig struct {
	port  int
	log   string
	cache *cacheConfig
}
