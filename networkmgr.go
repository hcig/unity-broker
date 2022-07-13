package main

import (
	"io"
	"log"
	"net"
	"os"
	"strconv"
)

var (
	PlainMode = true
)

type NetworkMgr struct {
	conn              *net.UDPConn
	Pubsub            *Pubsub
	Commands          *CommandHandler
	Persist           *PersistenceHandler
	ShutdownCompleted chan bool
	gz                *GzHandler
}

func NewNetworkMgr() *NetworkMgr {
	pm, err := strconv.ParseBool(os.Getenv("PLAIN_MODE"))
	if err != nil {
		pm = false
	}
	PlainMode = pm
	nm := &NetworkMgr{}
	nm.Pubsub = NewPubsub(nm)
	nm.Commands = NewCommandHandler(nm)
	nm.Persist = NewPersistenceHandler()
	nm.ShutdownCompleted = make(chan bool, 1)
	nm.gz = new(GzHandler)
	nm.gz.Setup()
	return nm
}

func (nm *NetworkMgr) Connect() error {
	s, err := net.ResolveUDPAddr("udp4", ":"+os.Getenv("BROKER_PORT"))
	log.Println("Listening on Port " + os.Getenv("BROKER_PORT"))
	if err != nil {
		return err
	}
	nm.conn, err = net.ListenUDP("udp4", s)
	if err != nil {
		return err
	}
	go nm.Listen()
	go nm.Publish()
	return nil
}

func (nm *NetworkMgr) Listen() {
	var buffer []byte
	for !nm.Pubsub.closed {
		buffer = make([]byte, 1024)
		n, addr, err := nm.conn.ReadFromUDP(buffer)
		data := buffer[0 : n-1]
		if !PlainMode {
			data, err = nm.gz.Unpack(data)
			if err != nil && err != io.ErrUnexpectedEOF {
				// Drop datagrams that are not parseable
				log.Println(err)
				continue
			}
		}
		//log.Printf("Read from Client: %s\n", data)
		cmd, err := ParseCommand(data, addr)
		if err != nil {
			// Drop datagrams that are not parseable
			log.Println(err)
			continue
		}

		// If new client is joining, add and subscribe
		nm.Pubsub.Subscribe(PubSubTopicBasic, addr)

		if err = nm.Commands.Handle(cmd); err != nil {
			log.Fatalln(err)
		}
	}
}

func (nm *NetworkMgr) Publish() {
	for !nm.Pubsub.closed {
		for _, clients := range nm.Pubsub.subs {
			for _, client := range clients {
				select {
				case msg := <-client.Chan:
					log.Printf("Sending: msg to %v: %s\n", client.Addr.String(), msg)
					_, err := nm.conn.WriteToUDP(msg, client.Addr)
					if err != nil {
						log.Printf("Error sending to UDP Client %s: %v", client.Addr, err)
					}
				}
			}
		}
	}
}

func (nm *NetworkMgr) Close() error {
	nm.Pubsub.Close()
	return nm.conn.Close()
}
