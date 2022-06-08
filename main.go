package main

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
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
	RegisterCommands()
	if err := netmgr.Connect(); err != nil {
		log.Fatal(err)
	}
}

func setupLogger() {
	logEnabled, err := strconv.ParseBool(os.Getenv("LOG_MESSAGES"))
	if err != nil {
		logEnabled = true // fallback to true
	}
	logfile := os.Getenv("LOG_FILE_NAME")
	file, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		file = nil
	}
	if !logEnabled {
		file, _ = os.OpenFile(os.DevNull, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	}
	if file != nil {
		log.SetOutput(file)
	}
}
