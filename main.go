package main

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/signal"
	"syscall"
	"viveSyncBroker/empatica"
)

var (
	netmgr *NetworkMgr
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err)
	}
	setupLogger()
	netmgr = NewNetworkMgr()
	empatica.Setup(netmgr.Persist)
	RegisterCommands()
	if err := netmgr.Connect(); err != nil {
		log.Fatal(err)
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	<-signals
	netmgr.Close()
	<-netmgr.ShutdownCompleted
}
