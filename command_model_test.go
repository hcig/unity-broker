package main

import (
	"testing"
	"time"

	messages "viveSyncBroker/pb"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestParseCommandRoundTrip(t *testing.T) {
	want := &messages.Command{
		Command:   messages.CommandType_SetCommand,
		Timestamp: timestamppb.New(time.Unix(10, 0)),
		Payload: &messages.Payload{
			PayloadTypes: &messages.Payload_Set{
				Set: &messages.Payload_SetPayload{Data: map[string]string{"k": "v"}},
			},
		},
	}

	raw, err := proto.Marshal(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := ParseCommand(raw, "source-1")
	if err != nil {
		t.Fatalf("ParseCommand returned error: %v", err)
	}

	if got.Source != "source-1" {
		t.Fatalf("source = %q, want %q", got.Source, "source-1")
	}
	if got.Command != want.Command {
		t.Fatalf("command = %v, want %v", got.Command, want.Command)
	}
	if got.Payload.GetSet().GetData()["k"] != "v" {
		t.Fatalf("payload not restored correctly: %#v", got.Payload)
	}
}

func TestParseCommandInvalidBytes(t *testing.T) {
	if _, err := ParseCommand([]byte("not-a-command"), "source"); err == nil {
		t.Fatal("expected error for invalid payload")
	}
}

func TestUpdateTimestampMovesOriginalTimestamp(t *testing.T) {
	orig := timestamppb.New(time.Unix(100, 0))
	com := &messages.Command{
		Timestamp: orig,
		Payload:   &messages.Payload{},
	}

	UpdateTimestamp(com)

	if com.Payload.OrigTimestamp != orig {
		t.Fatalf("orig timestamp not preserved")
	}
	if com.Timestamp == nil {
		t.Fatal("timestamp not updated")
	}
	if !com.Timestamp.AsTime().After(orig.AsTime()) && !com.Timestamp.AsTime().Equal(orig.AsTime()) {
		t.Fatalf("timestamp not set to a current-ish value: %v", com.Timestamp)
	}
}

func TestCommandHandlerRegisterHandleBroadcastPersist(t *testing.T) {
	nm := &NetworkMgr{Pubsub: NewPubsub(nil), Persist: &fakeHandler{}}
	ch := NewCommandHandler(nm)

	handled := false
	ch.Register("EchoCommand", func(c *messages.Command, _ *CommandHandler) error {
		handled = true
		c.Payload.Response = []string{"ok"}
		return nil
	})

	if err := ch.Handle(newCommand("source-1", messages.CommandType_EchoCommand)); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if !handled {
		t.Fatal("registered handler was not called")
	}

	msg := newCommand("source-1", messages.CommandType_MsgCommand)
	client := newMemoryConn("client-a", nil)
	nm.Pubsub.Subscribe(PubSubTopicBasic, client)
	ch.Broadcast(msg)

	entry := nm.Pubsub.subs[PubSubTopicBasic]
	value, ok := entry.Load(client.RemoteAddr().String())
	if !ok {
		t.Fatal("client not subscribed")
	}
	rc := value.(*RemoteClient)
	select {
	case got := <-rc.Chan:
		if got != msg {
			t.Fatal("broadcast did not route the original message")
		}
	default:
		t.Fatal("expected broadcast message")
	}

	ch.Persist(msg)
	if len(nm.Persist.(*fakeHandler).entries) != 1 {
		t.Fatalf("persist entry count = %d, want 1", len(nm.Persist.(*fakeHandler).entries))
	}
}

func TestCommandHandlerHandleMissing(t *testing.T) {
	ch := NewCommandHandler(&NetworkMgr{})
	err := ch.Handle(newCommand("source", messages.CommandType_GetCommand))
	if err == nil {
		t.Fatal("expected error for missing handler")
	}
}

func TestParseCommandSerializedFromProtoBytes(t *testing.T) {
	want := newCommand("source", messages.CommandType_DisconnectCommand)
	raw, err := proto.Marshal(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := ParseCommand(raw, "override")
	if err != nil {
		t.Fatalf("ParseCommand failed on proto bytes: %v", err)
	}
	if got.Source != "override" {
		t.Fatalf("source = %q, want override", got.Source)
	}
}
