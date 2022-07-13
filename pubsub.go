package main

import (
	"log"
	"net"
	"sync"
)

const (
	PubSubTopicBasic = "basic"
)

// UdpClient describes a client by its address and a channel for its mesages
type UdpClient struct {
	Addr *net.UDPAddr
	Chan chan []byte
}

// Pubsub describes a publish/subscribe broker with different topics to subscribe on.
type Pubsub struct {
	nm     *NetworkMgr
	mu     sync.RWMutex
	subs   map[string]map[string]*UdpClient
	closed bool
}

// NewPubsub creates a new Pubsub.
func NewPubsub(nm *NetworkMgr) *Pubsub {
	ps := &Pubsub{}
	ps.nm = nm
	ps.subs = make(map[string]map[string]*UdpClient)
	return ps
}

// Subscribe a client to a topic.
func (ps *Pubsub) Subscribe(topic string, addr *net.UDPAddr) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.subs[topic] == nil {
		ps.subs[topic] = make(map[string]*UdpClient)
	}
	s := addr.String()
	if ps.subs[topic][s] == nil {
		ps.subs[topic][s] = &UdpClient{
			Addr: addr,
			Chan: make(chan []byte),
		}
	}
}

// Unsubscribe a client from a topic.
func (ps *Pubsub) Unsubscribe(topic string, addr *net.UDPAddr) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.subs[topic] == nil {
		return
	}
	s := addr.String()
	if ps.subs[topic][s] != nil {
		delete(ps.subs[topic], s)
	}
}

// Publish a message to a topic.
func (ps *Pubsub) Publish(topic string, msg []byte) {
	ps.PublishWithOptions(topic, msg, PlainMode)
}

// PublishWithOptions publishes s message to a topic with a config if encryption should be used.
func (ps *Pubsub) PublishWithOptions(topic string, msg []byte, plain bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.closed {
		return
	}
	if !plain {
		data, err := ps.nm.gz.Pack(msg)
		if err != nil {
			log.Printf("ERROR compressing data: %v", err)
		}
		msg = data
	}
	for _, client := range ps.subs[topic] {
		go func(ch chan []byte) {
			ch <- msg
		}(client.Chan)
	}
}

// Unicast sends a message to a client
func (ps *Pubsub) Unicast(addr *net.UDPAddr, msg []byte) {
	ps.UnicastWithOptions(addr, msg, PlainMode)
}

// UnicastWithOptions sends a message to a client with a config if encryption should be used.
func (ps *Pubsub) UnicastWithOptions(addr *net.UDPAddr, msg []byte, plain bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.closed {
		return
	}
	if !plain {
		data, err := ps.nm.gz.Pack(msg)
		if err != nil {
			log.Printf("ERROR compressing data: %v", err)
		}
		msg = data
	}
	ps.subs[PubSubTopicBasic][addr.String()].Chan <- msg
}

// Close unsubscribes all clients from all topics.
func (ps *Pubsub) Close() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if !ps.closed {
		ps.closed = true
		for _, clients := range ps.subs {
			for _, client := range clients {
				close(client.Chan)
			}
		}
	}
}
