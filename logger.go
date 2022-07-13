package main

import (
	"log"
	"os"
	"strconv"
)

// setupLogger creates the logging component.
// Configuration is provided through .env file
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
