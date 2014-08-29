package logmessage

import "time"

// Message is a typed log message from the underlying log stream
// representing a single write from the stream
type Message struct {
	// ID is the unique identifier of the source of the message
	ID string `json:"id,omitempty"`

	// RawMessage is the raw bytes from the write
	RawMessage []byte `json:"raw_message,omitempty"`

	// Source specifies the source stream for where the message originated, stdout/stderr
	Source string `json:"source,omitempty"`

	// Time is the time the message was written
	Time time.Time `json:"time,omitempty"`

	// Fields are user defined attributes attached to a log message
	Fields map[string]string `json:"fields,omitempty"`
}

// Size returns the number of bytes in the message
func (m *Message) Size() int {
	return len(m.RawMessage)
}

// NewMessage returns an initialized message
func NewMessage(id, source string, msg []byte) *Message {
	return &Message{
		ID:         id,
		Source:     source,
		RawMessage: msg,
		Time:       time.Now(),
	}
}
