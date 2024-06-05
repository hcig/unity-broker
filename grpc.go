package main

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"reflect"
	"unsafe"
	"viveSyncBroker/pb/proto"
)

// BrokerServer is the central gRPC broker server
type BrokerServer struct {
	nm *NetworkMgr
}

func NewBrokerServer(nm *NetworkMgr) *BrokerServer {
	return &BrokerServer{
		nm: nm,
	}
}

func (s *BrokerServer) SendCommand(ctx context.Context, cmd *messages.Command) (*messages.Ack, error) {
	printContextInternals(ctx, false)
	fmt.Printf("[Server] received 'SEND': %v \n", cmd)
	return &messages.Ack{
		Source:  cmd.Source,
		Command: cmd.Command,
	}, nil
}

func (s *BrokerServer) RequestCommand(context.Context, *messages.Command) (*messages.Command, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RequestCommand not implemented")
}

func (s *BrokerServer) StreamUpdates(server messages.Broker_StreamUpdatesServer) error {
	for {
		cmd, err := server.Recv()
		if err != nil {
			return err
		}
		b, err := json.Marshal(cmd)
		if err != nil {
			return err
		}
		if err = s.nm.Persist.AddEntry(cmd.Source, b); err != nil {
			return err
		}
	}
}

func printContextInternals(ctx interface{}, inner bool) {
	contextValues := reflect.ValueOf(ctx)
	contextKeys := reflect.TypeOf(ctx)

	if !inner {
		fmt.Printf("\nFields for %s.%s\n", contextKeys.PkgPath(), contextKeys.Name())
	}

	if contextKeys.Kind() == reflect.Struct {
		for i := 0; i < contextValues.NumField(); i++ {
			reflectValue := contextValues.Field(i)
			reflectValue = reflect.NewAt(reflectValue.Type(), unsafe.Pointer(reflectValue.UnsafeAddr())).Elem()

			reflectField := contextKeys.Field(i)
			if reflectField.Name == "Context" {
				printContextInternals(reflectValue.Interface(), true)
			} else {
				fmt.Printf("field name: %+v\n", reflectField.Name)
				fmt.Printf("value: %+v\n", reflectValue.Interface())
			}
		}
	} else {
		fmt.Printf("context is empty (int)\n")
	}
}
