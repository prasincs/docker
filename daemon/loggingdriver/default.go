package loggingdriver

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/docker/docker/pkg/logmessage"
)

type Driver interface {
	io.Closer

	// ReadLog fetches the messages for a specific id
	ReadLog(id string) (messages []*logmessage.Message, err error)

	// CloseLog tells the adapter that no more log messages will be written for the specific id
	// drivers can implement this to their requirements, it may mean compressing the logs or deleting
	// them off of the disk
	CloseLog(id string) error

	NewLogger(id, source string) (logmessage.Logger, error)
}

type Default struct {
	sync.Mutex

	root    string
	loggers []*jsonLogger
}

type jsonLogger struct {
	log     io.WriteCloser
	encoder *json.Encoder
}

func (j *jsonLogger) Log(m *logmessage.Message) error {
	return j.encoder.Encode(m)
}

func (j *jsonLogger) Close() error {
	return j.log.Close()
}

func NewDefaultDriver(root string) Driver {
	return &Default{
		root: root,
	}
}

func (d *Default) NewLogger(id, source string) (logmessage.Logger, error) {
	dst := filepath.Join(d.root, id, fmt.Sprintf("%s-json.log", id))

	f, err := os.OpenFile(dst, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}

	return &jsonLogger{
		log:     f,
		encoder: json.NewEncoder(f),
	}, nil
}

func (d *Default) CloseLog(id string) error {
	return nil
}

func (d *Default) ReadLog(id string) ([]*logmessage.Message, error) {
	return nil, nil
}

func (d *Default) Close() (err error) {
	for _, l := range d.loggers {
		if nerr := l.Close(); err == nil {
			err = nerr
		}
	}

	return err
}
