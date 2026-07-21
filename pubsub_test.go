package main

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	messages "viveSyncBroker/pb"
)

func TestPubsubSubscribeUnsubscribeAndClients(t *testing.T) {
	ps := NewPubsub(nil)
	a := newMemoryConn("client-a", nil)
	b := newMemoryConn("client-b", nil)

	ps.Subscribe(PubSubTopicBasic, a)
	ps.Subscribe(PubSubTopicBasic, a)
	ps.Subscribe(PubSubTopicBasic, b)

	clients := ps.GetClients()
	if len(clients) != 2 {
		t.Fatalf("clients = %#v", clients)
	}

	ps.Unsubscribe(PubSubTopicBasic, a.RemoteAddr().String())
	clients = ps.GetClients()
	if len(clients) != 1 || clients[0] != b.RemoteAddr().String() {
		t.Fatalf("clients after unsubscribe = %#v", clients)
	}

	ps.Unsubscribe("missing", "whatever")
}

func TestPubsubPublishUnicastAndClose(t *testing.T) {
	ps := NewPubsub(nil)
	a := newMemoryConn("client-a", nil)
	b := newMemoryConn("client-b", nil)
	ps.Subscribe(PubSubTopicBasic, a)
	ps.Subscribe(PubSubTopicBasic, b)

	msg := &messages.Command{Timestamp: timestamppb.Now()}
	ps.Publish(PubSubTopicBasic, msg)

	select {
	case <-ps.HasMessages:
	case <-time.After(time.Second):
		t.Fatal("expected HasMessages signal")
	}

	for _, conn := range []*memoryConn{a, b} {
		value, ok := ps.subs[PubSubTopicBasic].Load(conn.RemoteAddr().String())
		if !ok {
			t.Fatalf("missing client %s", conn.RemoteAddr())
		}
		select {
		case got := <-value.(*RemoteClient).Chan:
			if got != msg {
				t.Fatalf("publish mismatch for %s", conn.RemoteAddr())
			}
		default:
			t.Fatalf("client %s did not receive publish", conn.RemoteAddr())
		}
	}

	unicast := &messages.Command{Command: messages.CommandType_MsgCommand}
	ps.Unicast(a.RemoteAddr().String(), unicast)
	value, _ := ps.subs[PubSubTopicBasic].Load(a.RemoteAddr().String())
	select {
	case got := <-value.(*RemoteClient).Chan:
		if got != unicast {
			t.Fatalf("unicast message mismatch: %#v", got)
		}
	default:
		t.Fatal("expected unicast message")
	}

	ps.Close()
	value, _ = ps.subs[PubSubTopicBasic].Load(a.RemoteAddr().String())
	if _, ok := <-value.(*RemoteClient).Chan; ok {
		t.Fatal("channel should be closed after Close")
	}

	ps.Publish(PubSubTopicBasic, msg)
	ps.Unicast(a.RemoteAddr().String(), msg)
}
