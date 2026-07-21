package main

import (
	"bytes"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
	messages "viveSyncBroker/pb"
	"viveSyncBroker/persistence"
)

type stubAddr string

func (a stubAddr) Network() string { return "stub" }
func (a stubAddr) String() string  { return string(a) }

type memoryConn struct {
	addr   net.Addr
	reader *bytes.Reader
	writer bytes.Buffer
	closed bool
	mu     sync.Mutex
}

func newMemoryConn(addr string, script []byte) *memoryConn {
	return &memoryConn{
		addr:   stubAddr(addr),
		reader: bytes.NewReader(script),
	}
}

func (c *memoryConn) Read(p []byte) (int, error) {
	if c.reader == nil {
		return 0, io.EOF
	}
	return c.reader.Read(p)
}

func (c *memoryConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writer.Write(p)
}

func (c *memoryConn) Close() error {
	c.closed = true
	return nil
}

func (c *memoryConn) LocalAddr() net.Addr  { return stubAddr("local") }
func (c *memoryConn) RemoteAddr() net.Addr { return c.addr }
func (c *memoryConn) SetDeadline(t time.Time) error {
	return nil
}
func (c *memoryConn) SetReadDeadline(t time.Time) error {
	return nil
}
func (c *memoryConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (c *memoryConn) WrittenBytes() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]byte(nil), c.writer.Bytes()...)
}

type fakeHandler struct {
	persistence.Handler
	prefix string

	lastParticipant      int
	lastParticipantErr   error
	setParticipantCalls  []int
	addParticipantData   []any
	lastTrial            int
	lastTrialErr         error
	setTrialCalls        []int
	addTrialData         []any
	questionnaires       map[string][]byte
	entries              []persistEntry
	setPrefixErr         error
	setParticipantErr    error
	addParticipantErr    error
	saveQuestionnaireErr error
	setTrialErr          error
	addTrialErr          error
	addEntryErr          error
}

type persistEntry struct {
	id  string
	msg proto.Message
}

func (f *fakeHandler) SetPrefix(prefix string) error {
	f.prefix = prefix
	return f.setPrefixErr
}

func (f *fakeHandler) LastParticipant() (int, error) { return f.lastParticipant, f.lastParticipantErr }
func (f *fakeHandler) SetParticipant(participant int) error {
	f.setParticipantCalls = append(f.setParticipantCalls, participant)
	return f.setParticipantErr
}
func (f *fakeHandler) AddParticipantData(data any) error {
	f.addParticipantData = append(f.addParticipantData, data)
	return f.addParticipantErr
}
func (f *fakeHandler) SaveQuestionnaire(subscale string, data []byte) error {
	if f.questionnaires == nil {
		f.questionnaires = make(map[string][]byte)
	}
	f.questionnaires[subscale] = append([]byte(nil), data...)
	return f.saveQuestionnaireErr
}
func (f *fakeHandler) LastTrial() (int, error) { return f.lastTrial, f.lastTrialErr }
func (f *fakeHandler) SetTrial(pass int) error {
	f.setTrialCalls = append(f.setTrialCalls, pass)
	return f.setTrialErr
}
func (f *fakeHandler) AddTrialData(data any) error {
	f.addTrialData = append(f.addTrialData, data)
	return f.addTrialErr
}
func (f *fakeHandler) AddEntry(id string, msg proto.Message) error {
	f.entries = append(f.entries, persistEntry{id: id, msg: msg})
	return f.addEntryErr
}
func (f *fakeHandler) Init() error  { return nil }
func (f *fakeHandler) Close() error { return nil }

func newCommand(source string, command messages.CommandType) *messages.Command {
	return &messages.Command{
		Source:  source,
		Command: command,
		Payload: &messages.Payload{},
	}
}

func withNetMgr(nm *NetworkMgr) func() {
	prev := netmgr
	netmgr = nm
	return func() {
		netmgr = prev
	}
}

func channelValue[T any](ch <-chan T) (T, bool) {
	var zero T
	select {
	case v := <-ch:
		return v, true
	default:
		return zero, false
	}
}

func must(err error) {
	if err != nil && !errors.Is(err, io.EOF) {
		panic(err)
	}
}
