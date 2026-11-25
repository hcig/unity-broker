package main

import (
	"compress/gzip"
	"errors"
	"testing"
)

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestGzHandlerPackWriteErrorBranch(t *testing.T) {
	gh := &GzHandler{}
	gh.Setup()
	gh.gw = gzip.NewWriter(failingWriter{err: errors.New("write failed")})
	if _, err := gh.Pack([]byte("payload")); err == nil {
		t.Fatal("expected Pack to fail when the underlying writer fails")
	}
}

func TestGzHandlerPackAndUnpack(t *testing.T) {
	var gh GzHandler
	gh.Setup()

	data := []byte("hello gzip")
	packed, err := gh.Pack(data)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if len(packed) == 0 {
		t.Fatal("expected compressed output")
	}

	unpacked, err := gh.Unpack(packed)
	if err != nil {
		t.Fatalf("Unpack: %v", err)
	}
	if string(unpacked) != string(data) {
		t.Fatalf("unpacked = %q, want %q", unpacked, data)
	}
}

func TestGzHandlerUnpackInvalidData(t *testing.T) {
	var gh GzHandler
	if _, err := gh.Unpack([]byte("not gzip")); err == nil {
		t.Fatal("expected error for invalid gzip data")
	}
}
