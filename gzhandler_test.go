package main

import (
	"testing"
)

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
