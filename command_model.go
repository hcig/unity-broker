package main

import (
	"fmt"
	"viveSyncBroker/pb"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ParseCommand unpacks a json string command to a Command.
func ParseCommand(cmd []byte, source string) (*messages.Command, error) {
	result := &messages.Command{}
	if err := proto.Unmarshal(cmd, result); err != nil {
		return nil, err
	}
	result.Source = source
	return result, nil
}

// UpdateTimestamp replaces the command's Command.Timestamp to the current time and moves the original timestamp as
// "orig_timestamp" field in the payload.
func UpdateTimestamp(c *messages.Command) {
	now := timestamppb.Now()
	c.Payload.OrigTimestamp = c.Timestamp
	c.Timestamp = now
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
	ch.nm.Pubsub.Publish(PubSubTopicBasic, com)
}

// Persist adds a Command to the persistence queue.
func (ch *CommandHandler) Persist(com *messages.Command) {
	ch.nm.Persist.AddEntry(com.Source, com)
}
