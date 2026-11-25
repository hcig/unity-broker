package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"testing"
	"time"
)

// serveListener is a net.Listener stub for testing BrokerServer.Serve.
type serveListener struct {
	conns chan net.Conn
}

func (l *serveListener) Accept() (net.Conn, error) {
	c, ok := <-l.conns
	if !ok {
		return nil, errors.New("serve listener closed")
	}
	return c, nil
}

func (l *serveListener) Close() error   { return nil }
func (l *serveListener) Addr() net.Addr { return stubAddr("serve-listener") }

func TestBrokerServerServeRegistersClients(t *testing.T) {
	listener := &serveListener{conns: make(chan net.Conn, 1)}
	nm := &NetworkMgr{
		Pubsub:  NewPubsub(nil),
		conn:    listener,
		clients: make(map[string]net.Conn),
	}
	bs := NewBrokerServer(nm)

	// net.Pipe end that is never written to: the client's ListenClient stays
	// parked in Read, so the connection remains registered.
	client, _ := net.Pipe()
	listener.conns <- client
	go bs.Serve()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, ok := nm.clients["pipe"]; ok {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("Serve did not register the accepted client")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	defer l.Close()
	return fmt.Sprint(l.Addr().(*net.TCPAddr).Port)
}

func dialPort(port string) bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func TestNetworkMgrConnectStartsListeners(t *testing.T) {
	t.Setenv("BROKER_PORT", freePort(t))
	t.Setenv("REST_PORT", freePort(t))
	t.Setenv("PERSIST_MODE", "off")

	nm := NewNetworkMgr()
	restore := withNetMgr(nm)
	defer restore()

	go nm.Connect()

	deadline := time.Now().Add(3 * time.Second)
	for {
		if dialPort(os.Getenv("BROKER_PORT")) && dialPort(os.Getenv("REST_PORT")) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("Connect did not start the broker or REST listeners")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestNetworkMgrConnectSSLSecondBindFails(t *testing.T) {
	t.Setenv("BROKER_SSL", "true")
	t.Setenv("BROKER_PORT", freePort(t))
	t.Setenv("REST_PORT", freePort(t))

	nm := NewNetworkMgr()
	restore := withNetMgr(nm)
	defer restore()

	go nm.Connect()
	deadline := time.Now().Add(3 * time.Second)
	for !dialPort(os.Getenv("BROKER_PORT")) {
		if time.Now().After(deadline) {
			t.Fatal("first Connect did not start the TLS listener")
		}
		time.Sleep(10 * time.Millisecond)
	}
	// The broker port is now bound; a second Connect must fail on bind.
	if err := nm.Connect(); err == nil {
		t.Fatal("expected second Connect to fail on the already-bound TLS port")
	}
}
