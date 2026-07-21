package persistence

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"viveSyncBroker/lib"

	"google.golang.org/protobuf/proto"
)

const StudyPrefix = "study"

// FileHandler represents a handler to persist Command to a CSV file.
type FileHandler struct {
	paused         *sync.Mutex
	prefix         string
	participantNum int
	passNum        int
	fileHandle     *os.File
	writeChan      chan proto.Message
}

// NewFileHandler creates a new PersistenceHandler and creates persistence files
func NewFileHandler() *FileHandler {
	ph := &FileHandler{
		paused: &sync.Mutex{},
	}
	return ph
}

func (ph *FileHandler) Init() error {
	if err := ph.openFile(); err != nil {
		return err
	}
	ph.writeChan = make(chan proto.Message)
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

// AddParticipantData adds data for a participant
func (ph *FileHandler) AddParticipantData(data any) error {
	return errors.New("not implemented")
}

func (ph *FileHandler) SaveQuestionnaire(subscale string, data []byte) error {
	return errors.New("not implemented")
}

// SetParticipant sets the participant number and restarts the file persistor
func (ph *FileHandler) SetParticipant(participant int) error {
	ph.participantNum = participant
	return ph.restart()
}

func (ph *FileHandler) LastParticipant() (int, error) {
	participantSet := lib.NewSet[int]()
	files, err := filepath.Glob(os.Getenv("PERSIST_FOLDER") + "/*.txt")
	if err != nil {
		return 0, err
	}
	for _, f := range files {
		partName := strings.Split(strings.TrimSuffix(filepath.Base(f), ".txt"), "_")[0]
		part, err := strconv.Atoi(partName)
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

// AddTrialData adds data for a trial
func (ph *FileHandler) AddTrialData(data any) error {
	return fmt.Errorf("Not implemented")
}

// SetTrial sets the pass number and restarts the file persistor
func (ph *FileHandler) SetTrial(pass int) error {
	ph.passNum = pass
	return ph.restart()
}

func (ph *FileHandler) LastTrial() (int, error) {
	trialSet := lib.NewSet[int]()
	files, err := filepath.Glob(os.Getenv("PERSIST_FOLDER") + fmt.Sprintf("/%d_*.txt", ph.participantNum))
	if err != nil {
		return 0, err
	}
	for _, f := range files {
		parts := strings.Split(strings.TrimSuffix(filepath.Base(f), ".txt"), "_")
		if len(parts) < 2 {
			return 0, fmt.Errorf("invalid trial filename: %s", f)
		}
		part, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
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
	ph.paused.Lock()
	defer ph.paused.Unlock()
	// Close old file
	if err := ph.closeFile(); err != nil {
		return err
	}
	// Open new file
	if err := ph.openFile(); err != nil {
		return err
	}
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
	return folder + strings.Join(pattern, "_") + ".txt"
}

// openFile creates a new file and provides a new csv.Writer to it.
func (ph *FileHandler) openFile() error {
	var err error
	ph.fileHandle, err = os.OpenFile(ph.createFilename(), os.O_CREATE|os.O_WRONLY, 0755)
	return err
}

// closeFile closes the writers to the persistence file.
func (ph *FileHandler) closeFile() error {
	return ph.fileHandle.Close()
}

// persistRoutine reads from the persistence channel and writes to the file
func (ph *FileHandler) persistRoutine() {
	for {
		ph.paused.TryLock()
		buf, ok := <-ph.writeChan
		if !ok {
			return
		}
		msg, err := json.Marshal(buf)
		if err != nil {
			log.Println(err)
		}
		if _, err = ph.fileHandle.Write(msg); err != nil {
			fmt.Println(err)
		}
		ph.paused.Unlock()
	}
}

// AddEntry adds a message with an identifier to the persistence channel.
func (ph *FileHandler) AddEntry(id string, msg proto.Message) error {
	ph.writeChan <- msg
	return nil
}
