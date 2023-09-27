package persistence

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const FilePrefix = "study"

// Handler represents a handler to persist Command to a CSV file.
type Handler struct {
	active         bool
	paused         *sync.WaitGroup
	prefix         string
	participantNum int
	passNum        int
	fileHandle     *os.File
	writer         *csv.Writer
	writeChan      chan []string
}

// NewHandler creates a new PersistenceHandler and creates persistence files
func NewHandler() *Handler {
	var found bool
	ph := &Handler{
		paused: &sync.WaitGroup{},
	}
	enabled, err := strconv.ParseBool(os.Getenv("PERSIST_EVENTS"))
	if err != nil {
		enabled = false // fallback to false
	}
	ph.active = enabled
	ph.prefix, found = os.LookupEnv("PERSIST_PREFIX")
	if !found {
		ph.prefix = FilePrefix // fall back to deafult prefix
	}
	if ph.active {
		ph.openFile()
		ph.writeChan = make(chan []string)
		go ph.persistRoutine()
	}
	return ph
}

// SetPrefix sets the study prefix and restarts the file persistor
func (ph *Handler) SetPrefix(prefix string) {
	ph.prefix = prefix
	ph.restart()
}

// SetParticipant sets the participant number and restarts the file persistor
func (ph *Handler) SetParticipant(participant int) {
	ph.participantNum = participant
	ph.restart()
}

// SetPass sets the pass number and restarts the file persistor
func (ph *Handler) SetPass(pass int) {
	ph.passNum = pass
	ph.restart()
}

// restart closes the old storage file and open a new one
func (ph *Handler) restart() {
	ph.paused.Add(1)
	// Close old file
	ph.closeFile()
	// Open new file
	ph.openFile()
	ph.paused.Done()
}

// createFilename assembles a filename for the logs.
func (ph *Handler) createFilename() string {
	pattern := []string{ph.prefix}
	if ph.participantNum > 0 {
		pattern = append(pattern, strconv.Itoa(ph.participantNum))
		// Passes can only be set if a participant is set
		if ph.passNum > 0 {
			pattern = append(pattern, strconv.Itoa(ph.passNum))
		} else {
			pattern = append(pattern, "pre")
		}
	}
	pattern = append(pattern, time.Now().Format("20060102-150405"))

	folder := os.Getenv("PERSIST_FOLDER")
	if folder != "" {
		folder += string(os.PathSeparator)
	}
	return folder + strings.Join(pattern, "_") + ".csv"
}

// openFile creates a new file and provides a new csv.Writer to it.
func (ph *Handler) openFile() {
	f, err := os.OpenFile(ph.createFilename(), os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		fmt.Println(err)
	}
	ph.fileHandle = f
	ph.writer = csv.NewWriter(ph.fileHandle)
	ph.writer.Comma = ';'
}

// closeFile closes the writers to the persistence file.
func (ph *Handler) closeFile() {
	close(ph.writeChan)
	if err := ph.fileHandle.Close(); err != nil {
		fmt.Println(err)
	}
}

// persistRoutine reads from the persistence channel and writes to the file
func (ph *Handler) persistRoutine() {
	for {
		ph.paused.Wait()
		buf := <-ph.writeChan
		if err := ph.writer.Write(buf); err != nil {
			fmt.Println(err)
		}
		ph.writer.Flush()
	}
}

// AddEntry adds a message with an identifier to the persistence channel.
func (ph *Handler) AddEntry(id string, msg []byte) {
	if ph.active {
		ph.writeChan <- []string{id, string(msg)}
	}
}
