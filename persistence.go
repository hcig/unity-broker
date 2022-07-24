package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// PersistenceHandler represents a handler to persist Command to a CSV file.
type PersistenceHandler struct {
	active     bool
	fileHandle *os.File
	writer     *csv.Writer
	writeChan  chan []string
}

// NewPersistenceHandler creates a new PersistenceHandler and creates persistence files
func NewPersistenceHandler() *PersistenceHandler {
	ph := &PersistenceHandler{}
	enabled, err := strconv.ParseBool(os.Getenv("PERSIST_EVENTS"))
	if err != nil {
		enabled = false // fallback to false
	}
	ph.active = enabled
	if ph.active {
		ph.openFile()
		ph.writeChan = make(chan []string)
		go ph.persistRoutine()
	}
	return ph
}

// createFilename assembles a filename for the logs.
func (ph *PersistenceHandler) createFilename() string {
	date := time.Now().Format("20060102-150405")
	pattern := os.Getenv("PERSIST_PATTERN")
	if pattern == "" {
		pattern = "events_{date}.csv"
	}
	placeholders := make(map[string]string)
	placeholders[`{date}`] = date
	for name, subst := range placeholders {
		pattern = strings.Replace(pattern, name, subst, 1)
	}
	folder := os.Getenv("PERSIST_FOLDER")
	if folder != "" {
		folder += string(os.PathSeparator)
	}
	return folder + pattern
}

// openFile creates a new file and provides a new csv.Writer to it.
func (ph *PersistenceHandler) openFile() {
	f, err := os.OpenFile(ph.createFilename(), os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		fmt.Println(err)
	}
	ph.fileHandle = f
	ph.writer = csv.NewWriter(ph.fileHandle)
	ph.writer.Comma = ';'
}

// closeFile closes the writers to the persistence file.
func (ph *PersistenceHandler) closeFile() {
	close(ph.writeChan)
	if err := ph.fileHandle.Close(); err != nil {
		fmt.Println(err)
	}
}

// persistRoutine reads from the persistence channel and writes to the file
func (ph *PersistenceHandler) persistRoutine() {
	for {
		buf := <-ph.writeChan
		if err := ph.writer.Write(buf); err != nil {
			fmt.Println(err)
		}
		ph.writer.Flush()
	}
}

// AddEntry adds a message with an identifier to the persistence channel.
func (ph *PersistenceHandler) AddEntry(id string, msg []byte) {
	if ph.active {
		ph.writeChan <- []string{id, string(msg)}
	}
}
