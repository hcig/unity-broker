package main

import (
	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"viveSyncBroker/pb/proto"
	"viveSyncBroker/persistence"
)

var (
	PlainMode = true
)

type NetworkMgr struct {
	conn              net.Listener
	Pubsub            *Pubsub
	Commands          *CommandHandler
	Persist           persistence.Handler
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
	nm.Persist = persistence.Factory()
	nm.ShutdownCompleted = make(chan bool, 1)
	nm.gz = new(GzHandler)
	nm.gz.Setup()
	return nm
}

func (nm *NetworkMgr) Connect() error {
	var err error
	// gRPC Connection
	log.Println("gRPC: Listening on Port " + os.Getenv("BROKER_PORT"))
	nm.conn, err = net.Listen("tcp4", "0.0.0.0:"+os.Getenv("BROKER_PORT"))
	if err != nil {
		return err
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	messages.RegisterBrokerServer(grpcServer, NewBrokerServer(nm))
	err = grpcServer.Serve(nm.conn)

	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/participants", ParticipantsHandler)
	r.HandleFunc("/trials", TrialsHandler)

	return http.ListenAndServe("0.0.0.0:"+os.Getenv("REST_PORT"), r)
}

func (nm *NetworkMgr) ListenRest() {
	//	var buffer []byte
	for !nm.Pubsub.closed {
		/*
			buffer = make([]byte, 1024)
			n, addr, err := nm.conn.ReadFromUDP(buffer)
			data := make([]byte, 0, n)
			if n > 0 {
				data = buffer[0 : n-1]
			}
			if !PlainMode {
				data, err = nm.gz.Unpack(data)
				if err != nil && err != io.ErrUnexpectedEOF {
					// Drop datagrams that are not parseable
					log.Println(err)
					continue
				}
			}
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
		*/
	}
}

func (nm *NetworkMgr) safeIterateCb(client *UdpClient) {

}

func (nm *NetworkMgr) Publish() {
	for !nm.Pubsub.closed {
		// Try to Lock to wait if no subs here
		for _, clients := range nm.Pubsub.subs {
			clients.Range(func(k interface{}, c interface{}) bool {
				//				client := c.(*UdpClient)
				select {
				/*
					case msg := <-client.Chan:
						//log.Printf("Sending to %v\n", client.Addr.String())
						_, err := nm.conn.WriteToUDP(msg, client.Addr)
						if err != nil {
							log.Printf("Error sending to UDP Client %s: %v", client.Addr, err)
						}
				*/
				default:
				}
				return true
			})
		}
	}
}

func (nm *NetworkMgr) Close() {
	nm.Pubsub.Close()
	_ = nm.conn.Close()
	nm.ShutdownCompleted <- true
}
