package main

import (
	"google.golang.org/protobuf/proto"
	"net"
	"sync"
)

const (
	PubSubTopicBasic = "basic"
)

// RemoteClient describes a client by its address and a channel for its mesages
type RemoteClient struct {
	Client net.Conn
	Chan   chan proto.Message
}

// Pubsub describes a publish/subscribe broker with different topics to subscribe on.
type Pubsub struct {
	nm          *NetworkMgr
	mu          sync.Mutex
	subs        map[string]*sync.Map
	closed      bool
	HasMessages chan bool
}

// NewPubsub creates a new Pubsub.
func NewPubsub(nm *NetworkMgr) *Pubsub {
	ps := &Pubsub{}
	ps.nm = nm
	ps.subs = make(map[string]*sync.Map)
	ps.HasMessages = make(chan bool, 16)
	return ps
}

// Subscribe a client to a topic.
func (ps *Pubsub) Subscribe(topic string, client net.Conn) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.subs[topic] == nil {
		ps.subs[topic] = &sync.Map{}
	}
	s := client.RemoteAddr().String()
	if _, ok := ps.subs[topic].Load(s); !ok {
		ps.subs[topic].Store(s, &RemoteClient{
			Client: client,
			Chan:   make(chan proto.Message, 8),
		})
	}
}

// Unsubscribe a client from a topic.
func (ps *Pubsub) Unsubscribe(topic string, client string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.subs[topic] == nil {
		return
	}
	_, _ = ps.subs[topic].LoadAndDelete(client)
}

// Publish a message to a topic.
func (ps *Pubsub) Publish(topic string, msg proto.Message) {
	ps.PublishWithOptions(topic, msg, PlainMode)
}

// PublishWithOptions publishes s message to a topic with a config if encryption should be used.
func (ps *Pubsub) PublishWithOptions(topic string, msg proto.Message, plain bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.closed {
		return
	}
	ps.subs[topic].Range(func(k interface{}, client interface{}) bool {
		client.(*RemoteClient).Chan <- msg
		ps.HasMessages <- true
		return true
	})
}

// Unicast sends a message to a client
func (ps *Pubsub) Unicast(client string, msg proto.Message) {
	ps.UnicastWithOptions(client, msg, PlainMode)
}

// UnicastWithOptions sends a message to a client with a config if encryption should be used.
func (ps *Pubsub) UnicastWithOptions(clientName string, msg proto.Message, plain bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.closed {
		return
	}
	client, _ := ps.subs[PubSubTopicBasic].Load(clientName)
	client.(*RemoteClient).Chan <- msg
	ps.HasMessages <- true
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
				close(c.(*RemoteClient).Chan)
				return true
			})
		}
	}
}
