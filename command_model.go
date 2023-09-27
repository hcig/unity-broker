package main

import (
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log"
	"net"
	"viveSyncBroker/pb/broker/messages"
)

// ToProtobufEnpoint converts a *net.UDPAddr to a protobuf *messages.IpEndpoint
func ToProtobufEnpoint(source *net.UDPAddr) *messages.IpEndpoint {
	ep := &messages.IpEndpoint{
		Port: uint32(source.Port),
	}
	if len(source.IP) == 4 {
		// 4 Bytes = IPv4
		ep.IpTypes = &messages.IpEndpoint_V4Ip{
			V4Ip: binary.LittleEndian.Uint32(source.IP),
		}
	} else {
		// 16 Bytes = IPv6; UINT64 is the max usable, so converting to a slice of 2 uint64
		const SIZEOF_INT64 = 8
		data := make([]uint64, len(source.IP)/SIZEOF_INT64)
		for i := range data {
			// assuming little endian
			data[i] = binary.LittleEndian.Uint64(source.IP[i*SIZEOF_INT64 : (i+1)*SIZEOF_INT64])
		}
		ep.IpTypes = &messages.IpEndpoint_V6Ip{
			V6Ip: &messages.IpEndpoint_V6Type{V6IpAddress: data},
		}
	}
	return ep
}

func FromProtobufEndpoint(ep *messages.IpEndpoint) *net.UDPAddr {
	addr := &net.UDPAddr{
		Port: int(ep.Port),
	}
	switch ep.IpTypes.(type) {
	case *messages.IpEndpoint_V4Ip:
		addr.IP = make([]byte, 4)
		binary.LittleEndian.PutUint32(addr.IP, ep.GetV4Ip())
		break
	case *messages.IpEndpoint_V6Ip:
		addr.IP = make([]byte, 16)
		// Upper 64 bytes
		binary.LittleEndian.PutUint64(addr.IP, ep.GetV6Ip().V6IpAddress[0])
		// Lower 64 bytes
		binary.LittleEndian.PutUint64(addr.IP[8:], ep.GetV6Ip().V6IpAddress[1])
		break
	}
	return addr
}

// ParseCommand unpacks a json string command to a Command.
func ParseCommand(cmd []byte, source *net.UDPAddr) (*messages.Command, error) {
	result := &messages.Command{}
	if err := proto.Unmarshal(cmd, result); err != nil {
		return nil, err
	}
	result.Source = ToProtobufEnpoint(source)
	return result, nil
}

// UpdateTimestamp replaces the command's Command.Timestamp to the current time and moves the original timestamp as
// "orig_timestamp" field in the payload.
func UpdateTimestamp(c *messages.Command) {
	now := timestamppb.Now()
	c.Payload.OrigTimestamp = c.Timestamp
	c.Timestamp = now
}

// ToBytes converts a Command to a byte slice.
func ToBytes(c *messages.Command) []byte {
	result, err := proto.Marshal(c)
	if err != nil {
		protoErr, err := proto.Marshal(&messages.CommandError{
			Message:   err.Error(),
			Timestamp: timestamppb.Now(),
			Reason:    string(result),
		})
		if err != nil {
			log.Fatal(err)
		}
		return protoErr
	}
	return result
}

// CommandHandler defines a registry and execution regulator for command name handlers.
type CommandHandler struct {
	nm       *NetworkMgr
	handlers map[messages.CommandType]func(*messages.Command, *CommandHandler) error
}

// NewCommandHandler creates a new CommandHandler.
func NewCommandHandler(nm *NetworkMgr) *CommandHandler {
	ch := &CommandHandler{}
	ch.nm = nm
	ch.handlers = make(map[messages.CommandType]func(*messages.Command, *CommandHandler) error)
	return ch
}

// Register adds a handler for a command name.
func (ch *CommandHandler) Register(name string, fn func(*messages.Command, *CommandHandler) error) *CommandHandler {
	ch.handlers[messages.CommandType(messages.CommandType_value[name])] = fn
	return ch
}

// Handle executes a Command for a specific command name handler. If no handler for the Command is registered, an error
// is being returned.
func (ch *CommandHandler) Handle(command *messages.Command) error {
	handler, found := ch.handlers[command.Command]
	if !found {
		return fmt.Errorf("could not find handler for '%s'", command.Command.String())
	}
	return handler(command, ch)
}

// Broadcast publishes a Command to the PubSubTopicBasic topic.
func (ch *CommandHandler) Broadcast(com *messages.Command) {
	ch.nm.Pubsub.Publish(PubSubTopicBasic, ToBytes(com))
}

// Respond sends a Command to the Command's source.
func (ch *CommandHandler) Respond(com *messages.Command) {
	ch.nm.Pubsub.Unicast(FromProtobufEndpoint(com.Source), ToBytes(com))
}

// Persist adds a Command to the persistence queue.
func (ch *CommandHandler) Persist(com *messages.Command) {
	ch.nm.Persist.AddEntry(com.Source.String(), ToBytes(com))
}
