package main

import (
	"fmt"
	"viveSyncBroker/pb"
)

// RegisterCommands is the central point to register commands.
func RegisterCommands() {
	// Echo cmd: Update timestamp and add original to the payload
	netmgr.BrokerServer.Register(messages.CommandType_EchoCommand, EchoCommand)
	// Disconnect from the broker
	netmgr.BrokerServer.Register(messages.CommandType_DisconnectCommand, DisconnectCommand)
	// Request broker information and general values
	netmgr.BrokerServer.Register(messages.CommandType_GetCommand, GetCommand)
	// Set broker information and general values
	netmgr.BrokerServer.Register(messages.CommandType_SetCommand, SetCommand)
	// Set broker information and general values
	netmgr.BrokerServer.Register(messages.CommandType_UpdateCommand, UpdateCommand)
	// Send a message to every listening component
	netmgr.BrokerServer.Register(messages.CommandType_MsgCommand, MsgCommand)
}

// EchoCommand is the Command for "echo".
func EchoCommand(com *messages.Command) error {
	err := netmgr.Persist.AddEntry(com.Source, com)
	if err != nil {
		return err
	}
	UpdateTimestamp(com)
	netmgr.Broadcast(com)
	return nil
}

// DisconnectCommand is the Command for "disconnect".
func DisconnectCommand(com *messages.Command) error {
	netmgr.Pubsub.Unsubscribe(PubSubTopicBasic, com.Source)
	return nil
}

// GetCommand is the Command for "get".
func GetCommand(com *messages.Command) error {
	UpdateTimestamp(com)
	fmt.Printf("%v\n", com.Payload.GetGet().GetData())
	for _, param := range com.Payload.GetGet().GetData() {
		switch param {
		case "help":
			help := make([]string, 0, len(netmgr.BrokerServer.handlers))
			for c := range netmgr.BrokerServer.handlers {
				help = append(help, c.String())
			}
			com.Payload.Response = help
			netmgr.Pubsub.Unicast(com.Source, com)
			break
		case "clients":
			com.Payload.Response = netmgr.Pubsub.GetClients()
			netmgr.Pubsub.Unicast(com.Source, com)
			break
		}
	}
	return nil
}

// SetCommand is the Command for "set"
func SetCommand(com *messages.Command) error {
	err := netmgr.Persist.AddEntry(com.Source, com)
	if err != nil {
		return err
	}
	netmgr.Broadcast(com)
	return nil
}

// UpdateCommand is the Command for "update".
func UpdateCommand(com *messages.Command) error {
	err := netmgr.Persist.AddEntry(com.Source, com)
	if err != nil {
		return err
	}
	netmgr.Broadcast(com)
	return nil
}

// MsgCommand is the Command for "send".
func MsgCommand(com *messages.Command) error {
	err := netmgr.Persist.AddEntry(com.Source, com)
	if err != nil {
		return err
	}
	netmgr.Broadcast(com)
	return nil
}
