package persistence

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"viveSyncBroker/lib"
)

const StudyPrefix = "study"

// FileHandler represents a handler to persist Command to a CSV file.
type FileHandler struct {
	paused         *sync.WaitGroup
	prefix         string
	participantNum int
	passNum        int
	fileHandle     *os.File
	writer         *csv.Writer
	writeChan      chan []string
}

// NewFileHandler creates a new PersistenceHandler and creates persistence files
func NewFileHandler() *FileHandler {
	ph := &FileHandler{
		paused: &sync.WaitGroup{},
	}
	return ph
}

func (ph *FileHandler) Init() error {
	if err := ph.openFile(); err != nil {
		return err
	}
	ph.writeChan = make(chan []string)
	go ph.persistRoutine()
	return nil
}

func (ph *FileHandler) Close() error {
	return ph.closeFile()
}

// SetPrefix sets the study prefix and restarts the file persistor
func (ph *FileHandler) SetPrefix(prefix string) error {
	ph.prefix = prefix
	return ph.restart()
}

// SetParticipant sets the participant number and restarts the file persistor
func (ph *FileHandler) SetParticipant(participant int) error {
	ph.participantNum = participant
	return ph.restart()
}

func (ph *FileHandler) LastParticipant() (int, error) {
	participantSet := lib.NewSet[int]()
	files, err := filepath.Glob(os.Getenv("PERSIST_FOLDER") + "/*.csv")
	if err != nil {
		return 0, err
	}
	for _, f := range files {
		part, err := strconv.Atoi(strings.Split(f, "_")[0])
		if err != nil {
			return 0, err
		}
		participantSet.Add(part)
	}
	lastParticipant := participantSet.Lowest()
	if lastParticipant == nil {
		return 0, nil
	}
	return *lastParticipant, nil
}

// SetTrial sets the pass number and restarts the file persistor
func (ph *FileHandler) SetTrial(pass int) error {
	ph.passNum = pass
	return ph.restart()
}

func (ph *FileHandler) LastTrial(participant int) (int, error) {
	trialSet := lib.NewSet[int]()
	files, err := filepath.Glob(os.Getenv("PERSIST_FOLDER") + fmt.Sprintf("/%d_*.csv", participant))
	if err != nil {
		return 0, err
	}
	for _, f := range files {
		part, err := strconv.Atoi(strings.Split(f, "_")[1])
		if err != nil {
			return 0, err
		}
		trialSet.Add(part)
	}
	lastTrial := trialSet.Lowest()
	if lastTrial == nil {
		return 0, nil
	}
	return *lastTrial, nil
}

// restart closes the old storage file and open a new one
func (ph *FileHandler) restart() error {
	ph.paused.Add(1)
	// Close old file
	if err := ph.closeFile(); err != nil {
		return err
	}
	// Open new file
	if err := ph.openFile(); err != nil {
		return err
	}
	ph.paused.Done()
	return nil
}

// createFilename assembles a filename for the logs.
func (ph *FileHandler) createFilename() string {
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
		folder += string(filepath.Separator)
	}
	return folder + strings.Join(pattern, "_") + ".csv"
}

// openFile creates a new file and provides a new csv.Writer to it.
func (ph *FileHandler) openFile() error {
	f, err := os.OpenFile(ph.createFilename(), os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	ph.fileHandle = f
	ph.writer = csv.NewWriter(ph.fileHandle)
	ph.writer.Comma = ';'
	return nil
}

// closeFile closes the writers to the persistence file.
func (ph *FileHandler) closeFile() error {
	close(ph.writeChan)
	return ph.fileHandle.Close()
}

// persistRoutine reads from the persistence channel and writes to the file
func (ph *FileHandler) persistRoutine() {
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
func (ph *FileHandler) AddEntry(id string, msg []byte) error {
	ph.writeChan <- []string{id, string(msg)}
	return nil
}
