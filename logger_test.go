package main

import (
	"log"
	"os"
	"strings"
	"testing"
)

func TestSetupLoggerWritesToConfiguredFile(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/app.log"
	oldOutput := log.Writer()
	t.Cleanup(func() { log.SetOutput(oldOutput) })

	t.Setenv("LOG_MESSAGES", "true")
	t.Setenv("LOG_FILE_NAME", path)
	setupLogger()
	log.Println("hello logger")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(data), "hello logger") {
		t.Fatalf("log file does not contain message: %q", string(data))
	}
}

func TestSetupLoggerFallsBackToDevNullWhenDisabled(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/app.log"
	oldOutput := log.Writer()
	t.Cleanup(func() { log.SetOutput(oldOutput) })

	t.Setenv("LOG_MESSAGES", "false")
	t.Setenv("LOG_FILE_NAME", path)
	setupLogger()
	log.Println("should disappear")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("expected no log output in configured file, got %q", string(data))
	}
}
