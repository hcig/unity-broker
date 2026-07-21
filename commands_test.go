package main

import (
	"errors"
	"testing"

	"google.golang.org/protobuf/types/known/timestamppb"
	messages "viveSyncBroker/pb"
)

func TestEchoDisconnectSetUpdateMsgCommands(t *testing.T) {
	fake := &fakeHandler{}
	nm := &NetworkMgr{
		Pubsub:       NewPubsub(nil),
		Persist:      fake,
		BrokerServer: NewBrokerServer(nil),
	}
	restore := withNetMgr(nm)
	defer restore()

	client := newMemoryConn("client-1", nil)
	nm.Pubsub.Subscribe(PubSubTopicBasic, client)

	msg := &messages.Command{
		Source:    client.RemoteAddr().String(),
		Command:   messages.CommandType_EchoCommand,
		Timestamp: timestamppb.Now(),
		Payload:   &messages.Payload{},
	}

	if err := EchoCommand(msg); err != nil {
		t.Fatalf("EchoCommand: %v", err)
	}
	if len(fake.entries) != 1 {
		t.Fatalf("EchoCommand persisted %d entries", len(fake.entries))
	}
	if fake.entries[0].id != client.RemoteAddr().String() {
		t.Fatalf("persist id = %q", fake.entries[0].id)
	}
	if msg.Payload.OrigTimestamp == nil {
		t.Fatal("EchoCommand did not move timestamp")
	}
	if got := readPubsubMessage(t, nm, client); got.Command != messages.CommandType_EchoCommand {
		t.Fatalf("broadcast command = %v", got.Command)
	}

	fake.addEntryErr = errors.New("boom")
	if err := SetCommand(msg); err == nil {
		t.Fatal("SetCommand should surface persistence error")
	}
	fake.addEntryErr = nil
	if err := SetCommand(msg); err != nil {
		t.Fatalf("SetCommand: %v", err)
	}
	if err := UpdateCommand(msg); err != nil {
		t.Fatalf("UpdateCommand: %v", err)
	}
	if err := MsgCommand(msg); err != nil {
		t.Fatalf("MsgCommand: %v", err)
	}
	if len(fake.entries) < 4 {
		t.Fatalf("expected multiple persistence entries, got %d", len(fake.entries))
	}

	if err := DisconnectCommand(msg); err != nil {
		t.Fatalf("DisconnectCommand: %v", err)
	}
	if _, ok := nm.Pubsub.subs[PubSubTopicBasic].Load(client.RemoteAddr().String()); ok {
		t.Fatal("DisconnectCommand did not unsubscribe client")
	}
}

func TestRegisterCommands(t *testing.T) {
	nm := &NetworkMgr{Pubsub: NewPubsub(nil), Persist: &fakeHandler{}, BrokerServer: NewBrokerServer(nil)}
	restore := withNetMgr(nm)
	defer restore()

	RegisterCommands()
	if len(nm.BrokerServer.handlers) != 6 {
		t.Fatalf("registered handlers = %d", len(nm.BrokerServer.handlers))
	}
}

func TestGetCommandHelpAndClients(t *testing.T) {
	fake := &fakeHandler{}
	nm := &NetworkMgr{
		Pubsub:       NewPubsub(nil),
		Persist:      fake,
		BrokerServer: NewBrokerServer(nil),
	}
	restore := withNetMgr(nm)
	defer restore()

	nm.BrokerServer.Register(messages.CommandType_EchoCommand, func(*messages.Command) error { return nil })
	nm.BrokerServer.Register(messages.CommandType_GetCommand, func(*messages.Command) error { return nil })
	client := newMemoryConn("client-2", nil)
	nm.Pubsub.Subscribe(PubSubTopicBasic, client)

	help := &messages.Command{
		Source: client.RemoteAddr().String(),
		Payload: &messages.Payload{
			PayloadTypes: &messages.Payload_Get{
				Get: &messages.Payload_GetPayload{Data: []string{"help"}},
			},
		},
	}
	if err := GetCommand(help); err != nil {
		t.Fatalf("GetCommand(help): %v", err)
	}
	if got := readPubsubMessage(t, nm, client); len(got.Payload.GetResponse()) == 0 {
		t.Fatal("help response was not set")
	}

	clients := &messages.Command{
		Source: client.RemoteAddr().String(),
		Payload: &messages.Payload{
			PayloadTypes: &messages.Payload_Get{
				Get: &messages.Payload_GetPayload{Data: []string{"clients"}},
			},
		},
	}
	if err := GetCommand(clients); err != nil {
		t.Fatalf("GetCommand(clients): %v", err)
	}
	got := readPubsubMessage(t, nm, client)
	if len(got.Payload.GetResponse()) != 1 || got.Payload.GetResponse()[0] != client.RemoteAddr().String() {
		t.Fatalf("clients response = %#v", got.Payload.GetResponse())
	}
}

func TestGetCommandUnknownAndNilPayload(t *testing.T) {
	nm := &NetworkMgr{Pubsub: NewPubsub(nil), Persist: &fakeHandler{}, BrokerServer: NewBrokerServer(nil)}
	restore := withNetMgr(nm)
	defer restore()

	if err := GetCommand(&messages.Command{Source: "src", Payload: &messages.Payload{PayloadTypes: &messages.Payload_Get{Get: &messages.Payload_GetPayload{Data: []string{"unknown"}}}}}); err != nil {
		t.Fatalf("GetCommand unknown returned error: %v", err)
	}
}

func readPubsubMessage(t *testing.T, nm *NetworkMgr, client *memoryConn) *messages.Command {
	t.Helper()
	value, ok := nm.Pubsub.subs[PubSubTopicBasic].Load(client.RemoteAddr().String())
	if !ok {
		t.Fatal("client not found in subscription map")
	}
	rc := value.(*RemoteClient)
	select {
	case got := <-rc.Chan:
		msg, ok := got.(*messages.Command)
		if !ok {
			t.Fatalf("unexpected message type %T", got)
		}
		return msg
	default:
		t.Fatal("expected pubsub message")
	}
	return nil
}
