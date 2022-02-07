package main

import (
	"bytes"
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
	hash           string
	token          string
	types          []string
	useCompression bool
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

// compressData applies lz4 compression to the supplied byte slice
func (b *baseHandler) compressData(data []byte, token string) ([]byte, error) {
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

// uncompressData applies decompression to the supplied byte slice
func (b *baseHandler) uncompressData(compressedData []byte, token string) ([]byte, error) {
	b.Debug("Page %v: Uncompressing", token)

	readAll := func(r io.Reader, initialSize int) error {
		// Reuse b.data if exists and useful size
		if b.data == nil || cap(b.data) < initialSize {
			b.data = make([]byte, 0, initialSize)
		} else {
			b.data = b.data[:0]
		}
		for {
			if len(b.data) == cap(b.data) {
				// Add more capacity (let append pick how much).
				b.data = append(b.data, 0)[:len(b.data)]
			}
			n, err := r.Read(b.data[len(b.data):cap(b.data)])
			b.data = b.data[:len(b.data)+n]
			if err != nil {
				if err == io.EOF {
					err = nil
				}
				return err
			}
		}
	}

	r := bytes.NewReader(compressedData)
	zr := lz4.NewReader(r)

	// Scaling of 5 as estimate of compression ratio to minimise reallocs
	err := readAll(zr, 5*len(compressedData))
	if err != nil {
		b.Error("Page %v: Zip read error - %v", token, err)
		return nil, fmt.Errorf("internal failure handling page (4)")
	}

	b.Debug("Page %v: Uncompressed", token)
	return b.data, nil
}

// writePage creates an encrypted page from the slice
func (b *baseHandler) writePage(data []byte, info *pageInfo) error {
	b.Debug("Page %v: Writing", info.token)
	defer b.Debug("Page %v: Completed", info.token)

	// Apply compression if specified
	if b.config.useCompression {
		var err error
		data, err = b.compressData(data, info.token)
		if err != nil {
			return err
		}
	}

	// If a key is provided, assume the page is to be encrypted
	if b.config.cipher != nil {
		b.Debug("Page %v: Encrypting", info.token)

		gcm, err := cipher.NewGCM(b.config.cipher)
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
	if b.config.cipher != nil {
		b.Debug("Page %v: Decrypting", info.token)

		gcm, err := cipher.NewGCM(b.config.cipher)
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

	// uncompress if requested
	if !info.useCompression {
		if b.config.useCompression {
			page, err = b.uncompressData(page, info.token)
			if err != nil {
				return nil, err
			}
		}
	}

	return
}
