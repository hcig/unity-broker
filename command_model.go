package main

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Command represents a generic command. Each command is described by a command name, timestamp and a command-name
// specific payload. Also, each Command includes the source client's address.
type Command struct {
	Source    *net.UDPAddr           `json:"-"`
	Command   *string                `json:"command"`
	Timestamp *time.Time             `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

// ParseCommand unpacks a json string command to a Command.
func ParseCommand(cmd []byte, source *net.UDPAddr) (*Command, error) {
	result := &Command{}
	if err := json.Unmarshal(cmd, result); err != nil {
		return nil, err
	}
	result.Source = source
	return result, nil
}

// UpdateTimestamp replaces the command's Command.Timestamp to the current time and moves the original timestamp as
// "orig_timestamp" field in the payload.
func (c *Command) UpdateTimestamp() {
	now := time.Now()
	c.Payload["orig_timestamp"] = c.Timestamp
	c.Timestamp = &now
}

// ToBytes converts a Command to a byte slice.
func (c *Command) ToBytes() []byte {
	result, err := json.Marshal(c)
	if err != nil {
		return []byte(fmt.Sprintf(
			`{"command":"error","timestamp":"%s","payload":"%v"}`,
			time.Now().Format("2006-01-02T15:04:05.999Z"),
			*c,
		))
	}
	return result
}

// CommandHandler defines a registry and execution regulator for command name handlers.
type CommandHandler struct {
	nm       *NetworkMgr
	handlers map[string]func(*Command, *CommandHandler) error
}

// NewCommandHandler creates a new CommandHandler.
func NewCommandHandler(nm *NetworkMgr) *CommandHandler {
	ch := &CommandHandler{}
	ch.nm = nm
	ch.handlers = make(map[string]func(*Command, *CommandHandler) error)
	return ch
}

// Register adds a handler for a command name.
func (ch *CommandHandler) Register(name string, fn func(*Command, *CommandHandler) error) *CommandHandler {
	ch.handlers[name] = fn
	return ch
}

// Handle executes a Command for a specific command name handler. If no handler for the Command is registered, an error
// is being returned.
func (ch *CommandHandler) Handle(command *Command) error {
	handler, found := ch.handlers[*command.Command]
	if !found {
		return fmt.Errorf("could not find handler for '%s'", *command.Command)
	}
	return handler(command, ch)
}

// Broadcast publishes a Command to the PubSubTopicBasic topic.
func (ch *CommandHandler) Broadcast(com *Command) {
	ch.nm.Pubsub.Publish(PubSubTopicBasic, com.ToBytes())
}

// Respond sends a Command to the Command's source.
func (ch *CommandHandler) Respond(com *Command) {
	ch.nm.Pubsub.Unicast(com.Source, com.ToBytes())
}

// Persist adds a Command to the persistence queue.
func (ch *CommandHandler) Persist(com *Command) {
	ch.nm.Persist.AddEntry(com.Source.String(), com.ToBytes())
}
