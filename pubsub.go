package main

import (
	"log"
	"net"
	"sync"
)

const (
	PubSubTopicBasic = "basic"
)

type UdpClient struct {
	Addr *net.UDPAddr
	Chan chan []byte
}

type Pubsub struct {
	nm     *NetworkMgr
	mu     sync.RWMutex
	subs   map[string]map[string]*UdpClient
	closed bool
}

func NewPubsub(nm *NetworkMgr) *Pubsub {
	ps := &Pubsub{}
	ps.nm = nm
	ps.subs = make(map[string]map[string]*UdpClient)
	return ps
}

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

func (ps *Pubsub) Publish(topic string, msg []byte) {
	ps.PublishWithOptions(topic, msg, PlainMode)
}

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

func (ps *Pubsub) Unicast(addr *net.UDPAddr, msg []byte) {
	ps.UnicastWithOptions(addr, msg, PlainMode)
}

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
