package main

import (
	"context"
	"testing"
	"time"

	messages "viveSyncBroker/pb"
)

func TestBrokerServerRegisterReceiveAndMissingHandler(t *testing.T) {
	nm := &NetworkMgr{}
	srv := NewBrokerServer(nm)
	called := make(chan struct{}, 1)
	srv.Register(messages.CommandType_GetCommand, func(cmd *messages.Command) error {
		called <- struct{}{}
		if cmd.Source != "source" {
			t.Fatalf("source = %q", cmd.Source)
		}
		return nil
	})

	ack, err := srv.ReceiveCommand(&messages.Command{
		Source:  "source",
		Command: messages.CommandType_GetCommand,
	})
	if err != nil {
		t.Fatalf("ReceiveCommand returned error: %v", err)
	}
	if ack.Source != "source" || ack.Command != messages.CommandType_GetCommand {
		t.Fatalf("ack = %#v", ack)
	}

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("registered handler not called")
	}

	ack, err = srv.ReceiveCommand(&messages.Command{Source: "source", Command: messages.CommandType_MsgCommand})
	if err != nil || ack != nil {
		t.Fatalf("missing handler should return nil,nil, got %#v %v", ack, err)
	}
}

func TestBrokerServerRequestCommandAndReceiveCommandErrorBranch(t *testing.T) {
	srv := NewBrokerServer(&NetworkMgr{})
	_, err := srv.RequestCommand(context.Background(), nil)
	if err == nil {
		t.Fatal("expected unimplemented error")
	}
	if err.Error() == "" {
		t.Fatal("expected a meaningful error")
	}
}
