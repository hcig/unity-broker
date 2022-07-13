package main

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"sync"
)

// GzHandler represents a middleware to gzip the messages for transmission
type GzHandler struct {
	writeBuffer *bytes.Buffer
	mu          sync.RWMutex
	gw          *gzip.Writer
	gr          *gzip.Reader
}

// Setup creates the buffer and a gzip.Writer to it.
func (gh *GzHandler) Setup() {
	gh.writeBuffer = &bytes.Buffer{}
	gh.gw = gzip.NewWriter(gh.writeBuffer)
}

// cleanup empties the buffers and unlocks the write-mutex
func (gh *GzHandler) cleanup() {
	gh.writeBuffer.Reset()
	gh.gw.Reset(gh.writeBuffer)
	gh.mu.Unlock()
}

// Pack gzips a message and returns the resulting byte slice
func (gh *GzHandler) Pack(data []byte) ([]byte, error) {
	gh.mu.Lock()
	defer gh.cleanup()
	_, err := gh.gw.Write(data)
	if err != nil {
		return nil, err
	}
	if err = gh.gw.Flush(); err != nil {
		return nil, err
	}
	return gh.writeBuffer.Bytes(), nil
}

// Unpack unzips a message using gzip and returns the plain message as byte slice.
func (gh *GzHandler) Unpack(data []byte) (result []byte, err error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(reader)
}
