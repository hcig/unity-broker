package main

import (
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
)

var (
	PlainMode = true
)

type NetworkMgr struct {
	conn     *net.UDPConn
	clients  map[*net.UDPAddr]chan string
	Pubsub   *Pubsub
	Commands *CommandHandler
	Persist  *PersistenceHandler
	gz       *GzHandler
	wg       *sync.WaitGroup
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
	nm.gz = new(GzHandler)
	nm.gz.Setup()
	nm.wg = new(sync.WaitGroup)
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
	nm.wg.Add(1)
	go nm.Listen()
	go nm.Publish()
	nm.wg.Wait()
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
		log.Printf("Read from Client: %s\n", data)
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
				_, err := nm.conn.WriteToUDP(<-client.Chan, client.Addr)
				if err != nil {
					log.Printf("Error sending to UDP Client %s: %v", client.Addr, err)
				}
			}
		}
	}
}

func (nm *NetworkMgr) Close() error {
	defer nm.wg.Done()
	nm.Pubsub.Close()
	return nm.conn.Close()
}
