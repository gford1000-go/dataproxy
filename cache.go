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
func (b *baseHandler) getCacheFileName(info *pageInfo) string {
	data := append(b.config.salt, info.hash...)
	data = append(data, info.token...)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%v/%x", b.config.root, hash[:])
}

// writePage creates an encrypted page from the slice
func (b *baseHandler) writePage(data []byte, info *pageInfo) error {
	b.Log(fmt.Sprintf("Page %v: Writing", info.token))		
	defer b.Log(fmt.Sprintf("Page %v: Completed", info.token))		

	// Always apply gzip
	b.Log(fmt.Sprintf("Page %v: GZipping", info.token))		

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)

	_, err := zw.Write(data)
	if err != nil {
		b.Log(fmt.Sprintf("Page %v: GZip write error - %v", info.token, err))		
		return err
	}
	zw.Close()

	data = buf.Bytes()

	b.Log(fmt.Sprintf("Page %v: GZipped", info.token))		

	// If a key is provided, assume the page is to be encrypted
	if len(b.config.key) > 0 {
		b.Log(fmt.Sprintf("Page %v: Encrypting", info.token))		

		c, err := aes.NewCipher(b.config.key)
		if err != nil {
			b.Log(fmt.Sprintf("Page %v: Error creating Cipher - %v", info.token, err))		
			return fmt.Errorf("Invalid page data")
		}

		gcm, err := cipher.NewGCM(c)
		if err != nil {
			b.Log(fmt.Sprintf("Page %v: Error creating GCM - %v", info.token, err))		
			return fmt.Errorf("Internal failure creating page (1)")
		}

		nonce := make([]byte, gcm.NonceSize())
		if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
			b.Log(fmt.Sprintf("Page %v: Error creating Nonce - %v", info.token, err))		
			return fmt.Errorf("Internal failure creating page (2)")
		}

		data = gcm.Seal(nonce, nonce, data, nil)

		b.Log(fmt.Sprintf("Page %v: Completed encryption", info.token))		
	}

	b.Log(fmt.Sprintf("Page %v: Writing to Disk", info.token))		
	err = ioutil.WriteFile(b.getCacheFileName(info), data, 0644)
	b.Log(fmt.Sprintf("Page %v: Writing to Disk Completed", info.token))		
	if err != nil {
		b.Log(fmt.Sprintf("Page %v: Error writing to disk - %v", info.token, err))		
	}

	return err
}

// retrievePage returns decrypted byte slice
func (b *baseHandler) retrievePage(info *pageInfo) (page []byte, err error) {
	b.Log(fmt.Sprintf("Page %v: Retrieving", info.token))		
	defer b.Log(fmt.Sprintf("Page %v: Completed retrieval", info.token))		

	pageLocation := b.getCacheFileName(info)

	b.Log(fmt.Sprintf("Page %v: Reading from Disk", info.token))		
	page, err = ioutil.ReadFile(pageLocation)
	b.Log(fmt.Sprintf("Page %v: Reading from Disk completed", info.token))		
	if err != nil {
		b.Log(fmt.Sprintf("Page %v: Error reading from disk - %v", info.token, err))		
		return nil, fmt.Errorf("Invalid request or page token")
	}

	// If a key is provided, assume the page is encrypted
	if len(b.config.key) > 0 {
		b.Log(fmt.Sprintf("Page %v: Decrypting", info.token))		

		c, err := aes.NewCipher(b.config.key)
		if err != nil {
			b.Log(fmt.Sprintf("Page %v: Error creating Cipher - %v", info.token, err))		
			return nil, fmt.Errorf("Invalid page data")
		}

		gcm, err := cipher.NewGCM(c)
		if err != nil {
			b.Log(fmt.Sprintf("Page %v: Error creating GCM - %v", info.token, err))		
			return nil, fmt.Errorf("Internal failure handling page (1)")
		}

		nonceSize := gcm.NonceSize()
		if len(page) < nonceSize {
			b.Log(fmt.Sprintf("Page %v: Data inconsistency", info.token))		
			return nil, fmt.Errorf("Internal failure handling page (2)")
		}

		nonce, data := page[:nonceSize], page[nonceSize:]
		page, err = gcm.Open(nil, nonce, data, nil)
		if err != nil {
			b.Log(fmt.Sprintf("Page %v: Error decrypting - %v", info.token, err))		
			return nil, fmt.Errorf("Internal failure handling page (3)")
		}

		b.Log(fmt.Sprintf("Page %v: Decrypted", info.token))		
	}

	// unzip if requested
	if !info.gzip {
		b.Log(fmt.Sprintf("Page %v: Ungzipping", info.token))		

		buf := bytes.NewBuffer(page)

		zr, err := gzip.NewReader(buf)
		defer zr.Close()

		if err != nil {
			b.Log(fmt.Sprintf("Page %v: Zip buffer error - %v", info.token, err))		
			return nil, fmt.Errorf("Internal failure handling page (4)")
		}

		page, err = ioutil.ReadAll(zr)
		if err != nil {
			b.Log(fmt.Sprintf("Page %v: Zip read error - %v", info.token, err))		
			return nil, fmt.Errorf("Internal failure handling page (4)")
		}

		b.Log(fmt.Sprintf("Page %v: Ungzipped", info.token))		
	}

	return
}
