package persistence

import "testing"

func TestBlackholeHandlerNoops(t *testing.T) {
	h := NewBlackholeHandler()
	if err := h.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := h.SetPrefix("study"); err != nil {
		t.Fatalf("SetPrefix: %v", err)
	}
	if err := h.SetParticipant(1); err != nil {
		t.Fatalf("SetParticipant: %v", err)
	}
	if err := h.AddParticipantData(map[string]any{"x": 1}); err != nil {
		t.Fatalf("AddParticipantData: %v", err)
	}
	if err := h.SaveQuestionnaire("demo", []byte("body")); err != nil {
		t.Fatalf("SaveQuestionnaire: %v", err)
	}
	if err := h.SetTrial(2); err != nil {
		t.Fatalf("SetTrial: %v", err)
	}
	if err := h.AddTrialData(map[string]any{"y": 2}); err != nil {
		t.Fatalf("AddTrialData: %v", err)
	}
	if err := h.AddEntry("id", nil); err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	if got, err := h.LastParticipant(); err != nil || got != 0 {
		t.Fatalf("LastParticipant = %d, %v", got, err)
	}
	if got, err := h.LastTrial(); err != nil || got != 0 {
		t.Fatalf("LastTrial = %d, %v", got, err)
	}
	if err := h.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
