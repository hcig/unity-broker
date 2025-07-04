package main

import (
	"bufio"
	"crypto/tls"
	"github.com/gorilla/mux"
	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	messages "viveSyncBroker/pb"
	"viveSyncBroker/persistence"
)

var (
	PlainMode = true
)

type NetworkMgr struct {
	conn              net.Listener
	Pubsub            *Pubsub
	BrokerServer      *BrokerServer
	Persist           persistence.Handler
	ShutdownCompleted chan bool
	clients           map[string]net.Conn
}

func NewNetworkMgr() *NetworkMgr {
	pm, err := strconv.ParseBool(os.Getenv("PLAIN_MODE"))
	if err != nil {
		pm = false
	}
	PlainMode = pm
	nm := &NetworkMgr{}
	nm.Pubsub = NewPubsub(nm)
	nm.Persist = persistence.Factory()
	nm.ShutdownCompleted = make(chan bool, 1)
	nm.clients = make(map[string]net.Conn)
	return nm
}

func (nm *NetworkMgr) Connect() error {
	var err error
	// gRPC Connection
	if os.Getenv("BROKER_SSL") == "true" {
		cer, err := tls.LoadX509KeyPair("certs/server.crt", "certs/server.key")
		if err != nil {
			return err
		}
		nm.conn, err = tls.Listen(
			"tcp4",
			"0.0.0.0:"+os.Getenv("BROKER_PORT"),
			&tls.Config{
				Certificates: []tls.Certificate{cer},
			},
		)
		if err != nil {
			return err
		}
	} else {
		nm.conn, err = net.Listen(
			"tcp4",
			"0.0.0.0:"+os.Getenv("BROKER_PORT"),
		)
		if err != nil {
			return err
		}
	}
	nm.BrokerServer = NewBrokerServer(nm)
	RegisterCommands()
	go nm.BrokerServer.Serve()
	go nm.Publish()
	log.Println("Protobuf/TCP: Listening on Port " + os.Getenv("BROKER_PORT"))

	router := mux.NewRouter()
	router.StrictSlash(true)
	router.HandleFunc("/", HomeHandler)
	router.HandleFunc("/participants", ParticipantsHandler)
	router.HandleFunc("/trials", TrialsHandler)
	router.HandleFunc("/override-gestures", nm.GesturesOverrideHandler)

	// R connection
	r := router.PathPrefix("/r").Subrouter()
	r.HandleFunc("/connection", RConnectionHandler)
	r.HandleFunc("/timeseries/{part}/{trial}", RTimeseriesQueryHandler)

	log.Println("REST: Listening on Port " + os.Getenv("REST_PORT"))
	return http.ListenAndServe("0.0.0.0:"+os.Getenv("REST_PORT"), router)
}

func (nm *NetworkMgr) HandleClient(conn net.Conn) {
	nm.clients[conn.RemoteAddr().String()] = conn
	go nm.ListenClient(conn)
}

func (nm *NetworkMgr) ListenClient(conn net.Conn) {
	reader := bufio.NewReader(conn)
	// If new client is joining, add and subscribe
	nm.Pubsub.Subscribe(PubSubTopicBasic, conn)
	cSrc := conn.RemoteAddr().String()
	for !nm.Pubsub.closed {
		cmd := &messages.Command{}
		err := protodelim.UnmarshalFrom(reader, cmd)
		if err == io.EOF {
			log.Println("Client closed conenction", err)
			nm.Pubsub.Unsubscribe(PubSubTopicBasic, cSrc)
			delete(nm.clients, cSrc)
			return
		}
		if err != nil {
			log.Println("Failed to read command:", err)
			continue
		}
		cmd.Source = cSrc
		ack, err := nm.BrokerServer.ReceiveCommand(cmd)
		if err != nil {
			log.Println("Failed to execute command:", err)
			continue
		}
		go nm.SendClient(conn, ack)
	}
}

func (nm *NetworkMgr) SendClient(conn net.Conn, message proto.Message) {
	_, err := protodelim.MarshalTo(conn, message)
	if err != nil {
		log.Println("Failed to marshal message:", err)
	}
}

// Broadcast publishes a Command to the PubSubTopicBasic topic.
func (nm *NetworkMgr) Broadcast(com *messages.Command) {
	nm.Pubsub.Publish(PubSubTopicBasic, com)
}

func (nm *NetworkMgr) Publish() {
	for !nm.Pubsub.closed {
		select {
		case <-nm.Pubsub.HasMessages:
			for _, clients := range nm.Pubsub.subs {
				clients.Range(func(k interface{}, c interface{}) bool {
					client := c.(*RemoteClient)
					select {
					case msg := <-client.Chan:
						_, err := protodelim.MarshalTo(client.Client, msg)
						if err != nil {
							log.Printf("Error sending to Client %s: %v", client.Client.RemoteAddr().String(), err)
						}
					default:
					}
					return true
				})
			}
		}
	}
}

func (nm *NetworkMgr) Close() {
	nm.Pubsub.Close()
	_ = nm.conn.Close()
	nm.ShutdownCompleted <- true
}
