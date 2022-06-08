package main

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"sync"
)

type GzHandler struct {
	writeBuffer *bytes.Buffer
	mu          sync.RWMutex
	gw          *gzip.Writer
	gr          *gzip.Reader
}

func (gh *GzHandler) Setup() {
	gh.writeBuffer = &bytes.Buffer{}
	gh.gw = gzip.NewWriter(gh.writeBuffer)
}

func (gh *GzHandler) cleanup() {
	gh.writeBuffer.Reset()
	gh.gw.Reset(gh.writeBuffer)
	gh.mu.Unlock()
}

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

func (gh *GzHandler) Unpack(data []byte) (result []byte, err error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(reader)
}
