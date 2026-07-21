package persistence

import (
	"strings"
	"testing"
)

func TestJsonEventValueAndScan(t *testing.T) {
	want := JsonEvent{Id: "id-1", Message: "hello"}
	raw, err := want.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	var got JsonEvent
	if err := got.Scan(raw.([]byte)); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got != want {
		t.Fatalf("decoded = %#v, want %#v", got, want)
	}
	if err := got.Scan("bad"); err == nil {
		t.Fatal("expected scan error for invalid input type")
	}
}

func TestTsConnectionFormattingAndTableName(t *testing.T) {
	conn := &TsConnection{
		Host:     "localhost",
		Port:     5432,
		DBName:   "db",
		User:     "user",
		Password: "pass",
	}
	if got := conn.AsRConnection(); !strings.Contains(got, "dbConnect") || !strings.Contains(got, "localhost") {
		t.Fatalf("AsRConnection = %q", got)
	}
	if got := conn.AsDbUri(); !strings.HasPrefix(got, "postgres://user:pass@localhost:5432/db") {
		t.Fatalf("AsDbUri = %q", got)
	}

	h := &TimescaleHandler{}
	h.prefix = "study"
	if got := h.tbl("trial"); got != "study_trials" {
		t.Fatalf("tbl = %q", got)
	}
	if got := h.GetTrialQuery(); !strings.Contains(got, "study_trials") {
		t.Fatalf("GetTrialQuery = %q", got)
	}
}
