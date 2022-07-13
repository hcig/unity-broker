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
	mu     sync.Mutex
	subs   map[string]*sync.Map
	closed bool
}

// NewPubsub creates a new Pubsub.
func NewPubsub(nm *NetworkMgr) *Pubsub {
	ps := &Pubsub{}
	ps.nm = nm
	ps.subs = make(map[string]*sync.Map)
	return ps
}

// Subscribe a client to a topic.
func (ps *Pubsub) Subscribe(topic string, addr *net.UDPAddr) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.subs[topic] == nil {
		ps.subs[topic] = &sync.Map{}
	}
	s := addr.String()
	if _, ok := ps.subs[topic].Load(s); !ok {
		ps.subs[topic].Store(s, &UdpClient{
			Addr: addr,
			Chan: make(chan []byte),
		})
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
	_, _ = ps.subs[topic].LoadAndDelete(s)
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
	ps.subs[topic].Range(func(k interface{}, client interface{}) bool {
		client.(*UdpClient).Chan <- msg
		return true
	})
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
	client, _ := ps.subs[PubSubTopicBasic].Load(addr.String())
	client.(*UdpClient).Chan <- msg
}

func (ps *Pubsub) GetClients() []string {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	var clients []string
	ps.subs[PubSubTopicBasic].Range(func(k interface{}, c interface{}) bool {
		clients = append(clients, k.(string))
		return true
	})
	return clients
}

// Close unsubscribes all clients from all topics.
func (ps *Pubsub) Close() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if !ps.closed {
		ps.closed = true
		for _, clients := range ps.subs {
			clients.Range(func(k interface{}, c interface{}) bool {
				close(c.(*UdpClient).Chan)
				return true
			})
		}
	}
}
