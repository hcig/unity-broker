package persistence

import (
	"google.golang.org/protobuf/proto"
	"os"
	"sync"
)

// BlackholeHandler represents a handler to persist nothing
type BlackholeHandler struct {
	paused         *sync.WaitGroup
	prefix         string
	participantNum int
	passNum        int
	fileHandle     *os.File
	writeChan      chan proto.Message
}

// NewBlackholeHandler creates a new BlackholeHandler
func NewBlackholeHandler() *BlackholeHandler {
	return &BlackholeHandler{}
}

func (ph *BlackholeHandler) Init() error {
	return nil
}

func (ph *BlackholeHandler) Close() error {
	return nil
}

// SetPrefix sets the study prefix and restarts the file persistor
func (ph *BlackholeHandler) SetPrefix(prefix string) error {
	ph.prefix = prefix
	return nil
}

// AddParticipantData adds data for a participant
func (ph *BlackholeHandler) AddParticipantData(data any) error {
	return nil
}

func (ph *BlackholeHandler) SaveQuestionnaire(subscale string, data []byte) error {
	return nil
}

// SetParticipant sets the participant number and restarts the file persistor
func (ph *BlackholeHandler) SetParticipant(participant int) error {
	return nil
}

func (ph *BlackholeHandler) LastParticipant() (int, error) {
	return 0, nil
}

// AddTrialData adds data for a trial
func (ph *BlackholeHandler) AddTrialData(data any) error {
	return nil
}

// SetTrial sets the pass number and restarts the file persistor
func (ph *BlackholeHandler) SetTrial(pass int) error {
	return nil
}

func (ph *BlackholeHandler) LastTrial() (int, error) {
	return 0, nil
}

// AddEntry adds a message with an identifier to the persistence channel.
func (ph *BlackholeHandler) AddEntry(id string, msg proto.Message) error {
	return nil
}
