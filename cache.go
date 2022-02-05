package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	lz4 "github.com/pierrec/lz4"
)

type pageInfo struct {
	hash  string
	token string
	types []string
	gzip  bool
}

// createCacheFileName creates the subfolder for the request if it doesn't exist
// and returns the name of the file
func (b *baseHandler) createCacheFileName(info *pageInfo) (string, error) {
	err := os.MkdirAll(fmt.Sprintf("%v/%v", b.config.root, info.hash), 0744)
	if err != nil {
		return "", err
	}
	return b.getCacheFileName(info), nil
}

// getCacheFileName returns a unique filename for a page
func (b *baseHandler) getCacheFileName(info *pageInfo) string {
	data := append(b.config.salt, info.hash...)
	data = append(data, info.token...)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%v/%v/%x", b.config.root, info.hash, hash[:])
}

// zipData applies default gzip to the supplied byte slice
func (b *baseHandler) zipData(data []byte, token string) ([]byte, error) {
	b.Debug("Page %v: Compressing", token)

	d := []byte{}
	r := bytes.NewReader(data)
	buf := bytes.NewBuffer(d)
	zw := lz4.NewWriter(buf)

	_, err := io.Copy(zw, r)
	if err != nil {
		b.Error("Page %v: LZ4 write error - %v", token, err)
		return nil, err
	}
	zw.Close()

	b.Debug("Page %v: LZ4", token)
	return buf.Bytes(), nil
}

// unzipData applies gzip decompression to the supplied byte slice
func (b *baseHandler) unzipData(data []byte, token string) ([]byte, error) {
	b.Debug("Page %v: Uncompressing", token)

	r := bytes.NewReader(data)
	zr := lz4.NewReader(r)

	data, err := ioutil.ReadAll(zr)
	if err != nil {
		b.Error("Page %v: Zip read error - %v", token, err)
		return nil, fmt.Errorf("internal failure handling page (4)")
	}

	b.Debug("Page %v: Uncompressed", token)
	return data, nil
}

// writePage creates an encrypted page from the slice
func (b *baseHandler) writePage(data []byte, info *pageInfo) error {
	b.Debug("Page %v: Writing", info.token)
	defer b.Debug("Page %v: Completed", info.token)

	// Apply gzip if specified
	if b.config.zip {
		var err error
		data, err = b.zipData(data, info.token)
		if err != nil {
			return err
		}
	}

	// If a key is provided, assume the page is to be encrypted
	if len(b.config.key) > 0 {
		b.Debug("Page %v: Encrypting", info.token)

		c, err := aes.NewCipher(b.config.key)
		if err != nil {
			b.Error("Page %v: Error creating Cipher - %v", info.token, err)
			return fmt.Errorf("invalid page data")
		}

		gcm, err := cipher.NewGCM(c)
		if err != nil {
			b.Error("Page %v: Error creating GCM - %v", info.token, err)
			return fmt.Errorf("internal failure creating page (1)")
		}

		nonce := make([]byte, gcm.NonceSize())
		if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
			b.Error("Page %v: Error creating Nonce - %v", info.token, err)
			return fmt.Errorf("internal failure creating page (2)")
		}

		data = gcm.Seal(nonce, nonce, data, nil)

		b.Debug("Page %v: Completed encryption", info.token)
	}

	b.Debug("Page %v: Writing to Disk", info.token)
	fileName, err := b.createCacheFileName(info)
	if err != nil {
		b.Error("Page %v: Error writing to disk - %v", info.token, err)
		return err
	}
	err = ioutil.WriteFile(fileName, data, 0644)
	b.Debug("Page %v: Writing to Disk Completed", info.token)
	if err != nil {
		b.Error("Page %v: Error writing to disk - %v", info.token, err)
	}

	return err
}

// retrievePage returns decrypted byte slice
func (b *baseHandler) retrievePage(info *pageInfo) (page []byte, err error) {
	b.Info("Page %v: Retrieving", info.token)
	defer b.Info("Page %v: Completed retrieval", info.token)

	pageLocation := b.getCacheFileName(info)

	b.Debug("Page %v: Reading from Disk", info.token)
	page, err = ioutil.ReadFile(pageLocation)
	b.Debug("Page %v: Reading from Disk completed", info.token)
	if err != nil {
		b.Error("Page %v: Error reading from disk - %v", info.token, err)
		return nil, fmt.Errorf("invalid request or page token")
	}

	// If a key is provided, assume the page is encrypted
	if len(b.config.key) > 0 {
		b.Debug("Page %v: Decrypting", info.token)

		c, err := aes.NewCipher(b.config.key)
		if err != nil {
			b.Error("Page %v: Error creating Cipher - %v", info.token, err)
			return nil, fmt.Errorf("invalid page data")
		}

		gcm, err := cipher.NewGCM(c)
		if err != nil {
			b.Error("Page %v: Error creating GCM - %v", info.token, err)
			return nil, fmt.Errorf("internal failure handling page (1)")
		}

		nonceSize := gcm.NonceSize()
		if len(page) < nonceSize {
			b.Error("Page %v: Data inconsistency", info.token)
			return nil, fmt.Errorf("internal failure handling page (2)")
		}

		nonce, data := page[:nonceSize], page[nonceSize:]
		page, err = gcm.Open(nil, nonce, data, nil)
		if err != nil {
			b.Error("Page %v: Error decrypting - %v", info.token, err)
			return nil, fmt.Errorf("internal failure handling page (3)")
		}

		b.Debug("Page %v: Decrypted", info.token)
	}

	// unzip if requested
	if !info.gzip {
		if b.config.zip {
			page, err = b.unzipData(page, info.token)
			if err != nil {
				return nil, err
			}
		}
	}

	return
}
