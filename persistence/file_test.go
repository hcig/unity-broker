package persistence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestFileHandlerCreateFilenameAndAddEntry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PERSIST_FOLDER", dir)

	h := NewFileHandler()
	h.prefix = "study"

	name := h.createFilename()
	if !strings.HasPrefix(name, filepath.Join(dir, "study_")) || !strings.HasSuffix(name, ".txt") {
		t.Fatalf("unexpected filename: %s", name)
	}

}

func TestFileHandlerCreateFilenameWithParticipantAndTrial(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PERSIST_FOLDER", dir)

	h := NewFileHandler()
	h.prefix = "study"
	h.participantNum = 4
	h.passNum = 2

	name := h.createFilename()
	if !strings.Contains(name, "study_4_2_") {
		t.Fatalf("unexpected filename: %s", name)
	}
}

func TestFileHandlerNotImplementedMethods(t *testing.T) {
	h := NewFileHandler()
	if err := h.AddParticipantData(map[string]any{"a": 1}); err == nil {
		t.Fatal("expected AddParticipantData to be not implemented")
	}
	if err := h.SaveQuestionnaire("kind", []byte("x")); err == nil {
		t.Fatal("expected SaveQuestionnaire to be not implemented")
	}
	if err := h.AddTrialData(map[string]any{"a": 1}); err == nil {
		t.Fatal("expected AddTrialData to be not implemented")
	}
}

func TestFileHandlerInitErrorWhenFolderMissing(t *testing.T) {
	t.Setenv("PERSIST_FOLDER", filepath.Join(t.TempDir(), "missing"))
	h := NewFileHandler()
	if err := h.Init(); err == nil {
		t.Fatal("expected Init to fail for a missing persistence folder")
	}
}

func TestFileHandlerInitAndAddEntry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PERSIST_FOLDER", dir)

	h := NewFileHandler()
	if err := h.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := h.AddEntry("id-1", &timestamppb.Timestamp{}); err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	if err := h.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestFileHandlerLastTrialInvalidFileContinues(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PERSIST_FOLDER", dir)
	if err := os.WriteFile(filepath.Join(dir, "1_bad.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	h := NewFileHandler()
	h.participantNum = 1
	got, err := h.LastTrial()
	if err != nil {
		t.Fatalf("LastTrial: %v", err)
	}
	if got != 0 {
		t.Fatalf("LastTrial = %d, want 0", got)
	}
}

func TestFileHandlerLastParticipantInvalidName(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PERSIST_FOLDER", dir)
	if err := os.WriteFile(filepath.Join(dir, "bad.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	h := NewFileHandler()
	if _, err := h.LastParticipant(); err == nil {
		t.Fatal("expected LastParticipant to fail for an invalid participant name")
	}
}

func TestFileHandlerLastParticipantAndTrial(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PERSIST_FOLDER", dir)

	for _, file := range []string{
		"1_pre_20240101.txt",
		"2_pre_20240101.txt",
		"2_1_20240101.txt",
		"2_3_20240101.txt",
	} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte("x"), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	h := NewFileHandler()
	got, err := h.LastParticipant()
	if err != nil {
		t.Fatalf("LastParticipant: %v", err)
	}
	if got != 1 {
		t.Fatalf("LastParticipant = %d, want 1", got)
	}

	h.participantNum = 2
	got, err = h.LastTrial()
	if err != nil {
		t.Fatalf("LastTrial: %v", err)
	}
	if got != 1 {
		t.Fatalf("LastTrial = %d, want 1", got)
	}
}

func TestFileHandlerRestartSettersAndAddEntry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PERSIST_FOLDER", dir)

	h := NewFileHandler()
	h.writeChan = make(chan proto.Message, 1)
	h.prefix = "study"
	if err := h.openFile(); err != nil {
		t.Fatalf("openFile: %v", err)
	}
	t.Cleanup(func() { _ = h.Close() })

	if err := h.SetPrefix("next"); err != nil {
		t.Fatalf("SetPrefix: %v", err)
	}
	if !strings.Contains(filepath.Base(h.fileHandle.Name()), "next") {
		t.Fatalf("file name not updated after SetPrefix: %s", h.fileHandle.Name())
	}
	if err := h.SetParticipant(5); err != nil {
		t.Fatalf("SetParticipant: %v", err)
	}
	if err := h.SetTrial(3); err != nil {
		t.Fatalf("SetTrial: %v", err)
	}
	if err := h.AddEntry("id", &timestamppb.Timestamp{}); err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	select {
	case <-h.writeChan:
	default:
		t.Fatal("expected AddEntry to queue a message")
	}
}
