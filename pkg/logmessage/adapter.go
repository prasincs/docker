package logmessage

import "io"

type Logger interface {
	io.Closer

	// Log commits the message to the underlying logging system
	Log(*Message) error
}
