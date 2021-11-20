package main

import (
	"bytes"
	"compress/gzip"
    "crypto/aes"
    "crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
)

type pageInfo struct {
	hash string 
	token string
	types []string
	gzip bool
}

// getCacheFileName returns a unique filename for a page
func getCacheFileName(config *cacheConfig, info *pageInfo) string {
	data := append(config.salt, info.hash...)
	data = append(data, info.token...)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%v/%x", config.root, hash[:])
}

// writePage creates an encrypted page from the slice
func writePage(data []byte, info *pageInfo, config *cacheConfig) error {
	logger := GetLogger()
	logger.Println(fmt.Sprintf("Writing Page %v", info.token))		
	defer logger.Println(fmt.Sprintf("Completed Page %v", info.token))		

	// Always apply gzip
	logger.Println(fmt.Sprintf("GZipping Page %v", info.token))		

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)

	_, err := zw.Write(data)
	if err != nil {
		logger.Println(fmt.Sprintf("GZip write erro for Page %v - %v", info.token, err))		
		return err
	}
	zw.Close()

	data = buf.Bytes()

	logger.Println(fmt.Sprintf("GZipped Page %v", info.token))		

	// If a key is provided, assume the page is to be encrypted
	if len(config.key) > 0 {
		logger.Println(fmt.Sprintf("Encrypting Page %v", info.token))		

		c, err := aes.NewCipher(config.key)
		if err != nil {
			logger.Println(fmt.Sprintf("Error creating Cipher for Page %v - %v", info.token, err))		
			return fmt.Errorf("Invalid page data")
		}

		gcm, err := cipher.NewGCM(c)
		if err != nil {
			logger.Println(fmt.Sprintf("Error creating GCM for Page %v - %v", info.token, err))		
			return fmt.Errorf("Internal failure creating page (1)")
		}

		nonce := make([]byte, gcm.NonceSize())
		if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
			logger.Println(fmt.Sprintf("Error creating Nonce for Page %v - %v", info.token, err))		
			return fmt.Errorf("Internal failure creating page (2)")
		}

		data = gcm.Seal(nonce, nonce, data, nil)

		logger.Println(fmt.Sprintf("Completed encryption for Page %v", info.token))		
	}

	err = ioutil.WriteFile(getCacheFileName(config, info), data, 0644)
	if err != nil {
		logger.Println(fmt.Sprintf("Error writing to disk for Page %v - %v", info.token, err))		
	}

	return err
}

// retrievePage returns decrypted byte slice
func retrievePage(info *pageInfo, config *cacheConfig) (page []byte, err error) {
	logger := GetLogger()
	logger.Println(fmt.Sprintf("Retrieving Page %v", info.token))		
	defer logger.Println(fmt.Sprintf("Completed Page %v", info.token))		

	pageLocation := getCacheFileName(config, info)

	page, err = ioutil.ReadFile(pageLocation)
	if err != nil {
		logger.Println(fmt.Sprintf("Error reading from disk for Page %v - %v", info.token, err))		
		return nil, fmt.Errorf("Invalid request or page token")
	}

	// If a key is provided, assume the page is encrypted
	if len(config.key) > 0 {
		logger.Println(fmt.Sprintf("Decrypting Page %v", info.token))		

		c, err := aes.NewCipher(config.key)
		if err != nil {
			logger.Println(fmt.Sprintf("Error creating Cipher for Page %v - %v", info.token, err))		
			return nil, fmt.Errorf("Invalid page data")
		}

		gcm, err := cipher.NewGCM(c)
		if err != nil {
			logger.Println(fmt.Sprintf("Error creating GCM for Page %v - %v", info.token, err))		
			return nil, fmt.Errorf("Internal failure handling page (1)")
		}

		nonceSize := gcm.NonceSize()
		if len(page) < nonceSize {
			logger.Println(fmt.Sprintf("Data inconsistency for Page %v", info.token))		
			return nil, fmt.Errorf("Internal failure handling page (2)")
		}

		nonce, data := page[:nonceSize], page[nonceSize:]
		page, err = gcm.Open(nil, nonce, data, nil)
		if err != nil {
			logger.Println(fmt.Sprintf("Error decrypting Page %v - %v", info.token, err))		
			return nil, fmt.Errorf("Internal failure handling page (3)")
		}

		logger.Println(fmt.Sprintf("Decrypted Page %v", info.token))		
	}

	// unzip if requested
	if !info.gzip {
		logger.Println(fmt.Sprintf("Ungzipping Page %v", info.token))		

		buf := bytes.NewBuffer(page)

		zr, err := gzip.NewReader(buf)
		defer zr.Close()

		if err != nil {
			logger.Println(fmt.Sprintf("Zip buffer error for Page %v - %v", info.token, err))		
			return nil, fmt.Errorf("Internal failure handling page (4)")
		}

		page, err = ioutil.ReadAll(zr)
		if err != nil {
			logger.Println(fmt.Sprintf("Zip read error for Page %v - %v", info.token, err))		
			return nil, fmt.Errorf("Internal failure handling page (4)")
		}

		logger.Println(fmt.Sprintf("Ungzipped Page %v", info.token))		
	}

	return
}
