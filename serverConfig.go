package main

import "crypto/cipher"

type cacheConfig struct {
	root   string
	salt   []byte
	cipher cipher.Block
	zip    bool
}

type serverConfig struct {
	port  int
	log   string
	cache *cacheConfig
}
