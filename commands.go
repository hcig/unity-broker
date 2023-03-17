package main

import (
	"fmt"
	"os"
)

// RegisterCommands is the central point to register commands.
func RegisterCommands() {
	// Echo cmd: Update timestamp and add original to the payload
	netmgr.Commands.Register("echo", EchoCommand)
	// Shutdown cmd: Shutdown the broker - FIXME to be removed ^^
	netmgr.Commands.Register("shutdown", ShutdownCommand)
	// Disconnect from the broker
	netmgr.Commands.Register("disconnect", DisconnectCommand)
	// Request broker information and general values
	netmgr.Commands.Register("get", GetCommand)
	// Set broker information and general values
	netmgr.Commands.Register("set", SetCommand)
	// Set broker information and general values
	netmgr.Commands.Register("update", UpdateCommand)
	// Send a message to every listening component
	netmgr.Commands.Register("msg", MsgCommand)
}

// EchoCommand is the Command for "echo".
func EchoCommand(com *Command, ch *CommandHandler) error {
	com.UpdateTimestamp()
	ch.Broadcast(com)
	return nil
}

// ShutdownCommand is the Command for "shutdown"
func ShutdownCommand(com *Command, ch *CommandHandler) error {
	// os.Exit sends a syscall.SIGINT on exit, that gets worked with in the shutdown routine
	os.Exit(1)
	return nil
}

// DisconnectCommand is the Command for "disconnect".
func DisconnectCommand(com *Command, ch *CommandHandler) error {
	ch.nm.Pubsub.Unsubscribe(PubSubTopicBasic, com.Source)
	return nil
}

// GetCommand is the Command for "get".
func GetCommand(com *Command, ch *CommandHandler) error {
	com.UpdateTimestamp()
	fmt.Printf("%v\n", com.Payload["params"])
	for _, param := range com.Payload["params"].([]interface{}) {
		switch param.(string) {
		case "help":
			help := make([]string, 0, len(netmgr.Commands.handlers))
			for c := range netmgr.Commands.handlers {
				help = append(help, c)
			}
			com.Payload["response"] = help
			ch.Respond(com)
			break
		case "clients":
			com.Payload["response"] = ch.nm.Pubsub.GetClients()
			ch.Respond(com)
			break
		}
	}
	return nil
}

// SetCommand is the Command for "set"
func SetCommand(com *Command, ch *CommandHandler) error {
	ch.Persist(com)
	ch.Broadcast(com)
	return nil
}

// UpdateCommand is the Command for "update".
func UpdateCommand(com *Command, ch *CommandHandler) error {
	ch.Persist(com)
	ch.Broadcast(com)
	return nil
}

// MsgCommand is the Command for "send".
func MsgCommand(com *Command, ch *CommandHandler) error {
	ch.Persist(com)
	ch.Broadcast(com)
	return nil
}
