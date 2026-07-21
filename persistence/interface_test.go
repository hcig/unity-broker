package persistence

import (
	"os"
	"strings"
	"testing"
)

func TestParseMode(t *testing.T) {
	cases := map[string]Mode{
		"file":       File,
		"file-xyz":   File,
		"timescale":  Db,
		"timescale1": Db,
		"off":        Off,
		"unknown":    Off,
	}
	for input, want := range cases {
		if got := ParseMode(input); got != want {
			t.Fatalf("ParseMode(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFactoryUsesBlackholeByDefault(t *testing.T) {
	t.Setenv("PERSIST_MODE", "unknown")
	t.Setenv("PERSIST_PREFIX", "custom")

	hdl := Factory()
	bh, ok := hdl.(*BlackholeHandler)
	if !ok {
		t.Fatalf("Factory returned %T, want BlackholeHandler", hdl)
	}
	if bh.prefix != "custom" {
		t.Fatalf("prefix = %q", bh.prefix)
	}
}

func TestFactoryFallsBackToStudyPrefix(t *testing.T) {
	t.Setenv("PERSIST_MODE", "off")
	original, found := os.LookupEnv("PERSIST_PREFIX")
	if err := os.Unsetenv("PERSIST_PREFIX"); err != nil {
		t.Fatalf("Unsetenv: %v", err)
	}
	t.Cleanup(func() {
		if found {
			_ = os.Setenv("PERSIST_PREFIX", original)
		}
	})

	hdl := Factory()
	if _, ok := hdl.(*BlackholeHandler); !ok {
		t.Fatalf("Factory returned %T", hdl)
	}
}

func TestFactoryFileModeInitializesAndPersists(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PERSIST_MODE", "file")
	t.Setenv("PERSIST_FOLDER", dir)
	t.Setenv("PERSIST_PREFIX", "study")

	hdl := Factory()
	fh, ok := hdl.(*FileHandler)
	if !ok {
		t.Fatalf("Factory returned %T, want FileHandler", hdl)
	}
	if fh.writeChan == nil || fh.fileHandle == nil {
		t.Fatal("file handler was not initialized")
	}

	if err := fh.AddEntry("id", nil); err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	data, err := os.ReadFile(fh.fileHandle.Name())
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "null") {
		t.Fatalf("expected persisted output, got %q", string(data))
	}
}
