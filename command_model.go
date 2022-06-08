package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
)

type Command struct {
	Source    *net.UDPAddr           `json:"-"`
	Command   *string                `json:"command"`
	Timestamp *time.Time             `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

func ParseCommand(cmd []byte, source *net.UDPAddr) (*Command, error) {
	result := &Command{}
	if err := json.Unmarshal(cmd, result); err != nil {
		return nil, err
	}
	result.Source = source
	return result, nil
}

func (c *Command) UpdateTimestamp() {
	now := time.Now()
	c.Payload["orig_timestamp"] = c.Timestamp
	c.Timestamp = &now
}

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

type CommandHandler struct {
	nm       *NetworkMgr
	handlers map[string]func(*Command, *CommandHandler) error
}

func NewCommandHandler(nm *NetworkMgr) *CommandHandler {
	ch := &CommandHandler{}
	ch.nm = nm
	ch.handlers = make(map[string]func(*Command, *CommandHandler) error)
	return ch
}

func (ch *CommandHandler) Register(name string, fn func(*Command, *CommandHandler) error) *CommandHandler {
	ch.handlers[name] = fn
	return ch
}

func (ch *CommandHandler) Handle(command *Command) error {
	handler, found := ch.handlers[*command.Command]
	if !found {
		return fmt.Errorf("could not find handler for '%s'", *command.Command)
	}
	return handler(command, ch)
}

func (ch *CommandHandler) Broadcast(com *Command) {
	ch.nm.Pubsub.Publish(PubSubTopicBasic, com.ToBytes())
}

func (ch *CommandHandler) Respond(com *Command) {
	log.Printf("Sending: msg to %v\n", com.Source.String())
	ch.nm.Pubsub.Unicast(com.Source, com.ToBytes())
}
