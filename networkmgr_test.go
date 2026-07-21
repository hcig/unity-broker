package main

import (
	"bytes"
	"errors"
	"net"
	"testing"
	"time"

	messages "viveSyncBroker/pb"

	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNewNetworkMgrHonorsPlainModeAndFactory(t *testing.T) {
	t.Setenv("PLAIN_MODE", "true")
	t.Setenv("PERSIST_MODE", "off")

	nm := NewNetworkMgr()
	if !PlainMode {
		t.Fatal("expected plain mode to be enabled")
	}
	if nm.Pubsub == nil || nm.Persist == nil || nm.ShutdownCompleted == nil {
		t.Fatal("network manager was not initialized")
	}
}

func TestListenClientReceivesCommandAndSendsAck(t *testing.T) {
	fake := &fakeHandler{}
	nm := &NetworkMgr{
		Pubsub:       NewPubsub(nil),
		Persist:      fake,
		BrokerServer: NewBrokerServer(nil),
	}
	restore := withNetMgr(nm)
	defer restore()

	handled := make(chan struct{}, 1)
	nm.BrokerServer.Register(messages.CommandType_EchoCommand, func(cmd *messages.Command) error {
		handled <- struct{}{}
		return nil
	})
	var cmdBuf bytes.Buffer
	if _, err := protodelim.MarshalTo(&cmdBuf, &messages.Command{
		Command:   messages.CommandType_EchoCommand,
		Timestamp: timestamppb.Now(),
		Payload:   &messages.Payload{},
	}); err != nil {
		t.Fatalf("marshal: %v", err)
	}

	conn := newMemoryConn("client-1", cmdBuf.Bytes())
	done := make(chan struct{})
	go func() {
		nm.ListenClient(conn)
		close(done)
	}()

	select {
	case <-handled:
	case <-time.After(time.Second):
		t.Fatal("command handler was not called")
	}

	conn.reader = bytes.NewReader(nil)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("ListenClient did not exit on EOF")
	}
	if len(conn.WrittenBytes()) == 0 {
		t.Fatal("expected ack to be written back to the client")
	}
}

func TestNetworkMgrBroadcastAndPublish(t *testing.T) {
	nm := &NetworkMgr{Pubsub: NewPubsub(nil)}
	restore := withNetMgr(nm)
	defer restore()

	conn := newMemoryConn("client-1", nil)
	nm.Pubsub.Subscribe(PubSubTopicBasic, conn)

	msg := &messages.Command{Timestamp: timestamppb.Now()}
	done := make(chan struct{})
	go func() {
		nm.Publish()
		close(done)
	}()

	nm.Broadcast(msg)
	select {
	case <-nm.Pubsub.HasMessages:
	case <-time.After(time.Second):
		t.Fatal("expected publish signal")
	}

	value, _ := nm.Pubsub.subs[PubSubTopicBasic].Load(conn.RemoteAddr().String())
	select {
	case got := <-value.(*RemoteClient).Chan:
		if got != msg {
			t.Fatal("broadcast did not reach client")
		}
	case <-time.After(time.Second):
		t.Fatal("client did not receive broadcast")
	}

	nm.Pubsub.Close()
	nm.Pubsub.HasMessages <- true
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish did not exit after close")
	}
}

type closingListener struct {
	closed bool
}

func (c *closingListener) Accept() (net.Conn, error) { return nil, errors.New("closed") }
func (c *closingListener) Close() error              { c.closed = true; return nil }
func (c *closingListener) Addr() net.Addr            { return stubAddr("listener") }

func TestHandleClientAndClose(t *testing.T) {
	conn := newMemoryConn("client-1", nil)
	nm := &NetworkMgr{
		Pubsub:            NewPubsub(nil),
		BrokerServer:      NewBrokerServer(nil),
		ShutdownCompleted: make(chan bool, 1),
		conn:              &closingListener{},
		clients:           make(map[string]net.Conn),
	}
	restore := withNetMgr(nm)
	defer restore()

	nm.HandleClient(conn)
	time.Sleep(20 * time.Millisecond)
	if _, ok := nm.clients[conn.RemoteAddr().String()]; !ok {
		t.Log("HandleClient did not record connection")
	}

	nm.Close()
	select {
	case <-nm.ShutdownCompleted:
	case <-time.After(time.Second):
		t.Fatal("Close did not signal shutdown")
	}
}

type errorWriteConn struct{ memoryConn }

func (e *errorWriteConn) Write(p []byte) (int, error) { return 0, errors.New("write fail") }

func TestSendClientErrorBranch(t *testing.T) {
	nm := &NetworkMgr{}
	nm.SendClient(&errorWriteConn{}, &messages.Command{})
}
