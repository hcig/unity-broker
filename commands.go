package main

import "fmt"

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
}

func EchoCommand(com *Command, ch *CommandHandler) error {
	com.UpdateTimestamp()
	ch.Broadcast(com)
	return nil
}

func ShutdownCommand(com *Command, ch *CommandHandler) error {
	return ch.nm.Close()
}

func DisconnectCommand(com *Command, ch *CommandHandler) error {
	ch.nm.Pubsub.Unsubscribe(PubSubTopicBasic, com.Source)
	return nil
}

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
			clients := make([]string, 0, len(ch.nm.clients))
			for c := range ch.nm.clients {
				clients = append(clients, c.String())
			}
			com.Payload["response"] = clients
			ch.Respond(com)
			break
		}
	}
	return nil
}

func SetCommand(com *Command, ch *CommandHandler) error {

	return nil
}

func UpdateCommand(com *Command, ch *CommandHandler) error {
	ch.Persist(com)
	ch.Broadcast(com)
	return nil
}
