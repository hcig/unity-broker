package persistence

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"os"
	"strings"
)

type Handler interface {
	SetPrefix(prefix string) error

	LastParticipant() (int, error)
	SetParticipant(participant int) error
	AddParticipantData(data any) error

	LastTrial() (int, error)
	SetTrial(pass int) error
	AddTrialData(data any) error

	AddEntry(id string, msg proto.Message) error

	Init() error
	Close() error
}

type Mode string

const (
	Off  Mode = "off"
	File Mode = "file"
	Db   Mode = "timescale"
)

func ParseMode(str string) Mode {
	plain := strings.ToLower(str)
	switch {
	case strings.HasPrefix(plain, string(File)):
		return File
	case strings.HasPrefix(plain, string(Db)):
		return Db
	case strings.HasPrefix(plain, string(Off)):
		return Off
	}
	return Off
}

func Factory() Handler {
	mode := ParseMode(os.Getenv("PERSIST_MODE"))
	var hdl Handler
	switch mode {
	case File:
		hdl = NewFileHandler()
	case Db:
		hdl = NewTimescaleHandler()
	}
	prefix, found := os.LookupEnv("PERSIST_PREFIX")
	if !found {
		prefix = StudyPrefix // fall back to default prefix
	}
	if err := hdl.SetPrefix(prefix); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
	if err := hdl.Init(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
	return hdl
}
