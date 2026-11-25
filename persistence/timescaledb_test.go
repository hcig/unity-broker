package persistence

import (
	"net"
	"strconv"
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

func TestTimescaleHandlerConstructionAndSetPrefix(t *testing.T) {
	h := NewTimescaleHandler()
	if h == nil {
		t.Fatal("NewTimescaleHandler returned nil")
	}
	if err := h.SetPrefix("study"); err != nil {
		t.Fatalf("SetPrefix: %v", err)
	}
	if h.prefix != "study" {
		t.Fatalf("prefix = %q", h.prefix)
	}
}

func TestTimescaleHandlerConnectAndInitWithoutDatabase(t *testing.T) {
	// Reserve a port and close it immediately so the database is unreachable.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()

	prev := TSConfig
	t.Cleanup(func() { TSConfig = prev })

	t.Setenv("PERSIST_TIMESCALE_HOST", "127.0.0.1")
	t.Setenv("PERSIST_TIMESCALE_PORT", strconv.Itoa(port))
	t.Setenv("PERSIST_TIMESCALE_DATABASE", "study")
	t.Setenv("PERSIST_TIMESCALE_USERNAME", "postgres")
	t.Setenv("PERSIST_TIMESCALE_PASSWORD", "pass")

	h := NewTimescaleHandler()
	// Connect with a global (non-database-specific) configuration.
	if err := h.connect(true); err != nil {
		t.Logf("connect(global): %v", err)
	}
	if TSConfig == nil || TSConfig.Host != "127.0.0.1" || TSConfig.DBName != "" {
		t.Fatalf("unexpected TSConfig: %#v", TSConfig)
	}

	// Init must fail because there is no reachable database.
	if err := h.Init(); err == nil {
		t.Fatal("expected Init to fail without a reachable database")
	}
}
