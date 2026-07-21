package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"viveSyncBroker/pb"
)

// BrokerServer is the central gRPC broker server
type BrokerServer struct {
	nm       *NetworkMgr
	handlers map[messages.CommandType]func(com *messages.Command) error
}

func NewBrokerServer(nm *NetworkMgr) *BrokerServer {
	return &BrokerServer{
		nm:       nm,
		handlers: make(map[messages.CommandType]func(com *messages.Command) error),
	}
}

func (s *BrokerServer) Serve() {
	defer s.nm.Close()
	// Broker Server main loop
	for {
		conn, err := s.nm.conn.Accept()
		if err != nil {
			fmt.Printf("Error receiving: %v\n", err)
		}
		if conn != nil {
			go s.nm.HandleClient(conn)
		}
	}
}

func (s *BrokerServer) ReceiveCommand(cmd *messages.Command) (*messages.Ack, error) {
	hdl, found := s.handlers[cmd.Command]
	if !found {
		fmt.Printf("Could not find command handler for %s\n", cmd.Command)
		return nil, nil
	}
	go func() {
		err := hdl(cmd)
		if err != nil {
			fmt.Printf("Error on handler handler for %s: %v\n", cmd.Command, err)
		}
	}()
	return &messages.Ack{
		Source:  cmd.Source,
		Command: cmd.Command,
	}, nil
}

func (s *BrokerServer) RequestCommand(context.Context, *messages.Command) (*messages.Command, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RequestCommand not implemented")
}

func (s *BrokerServer) Register(t messages.CommandType, cmd func(com *messages.Command) error) {
	s.handlers[t] = cmd
}
