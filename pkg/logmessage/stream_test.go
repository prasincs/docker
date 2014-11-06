package logmessage

import (
	"bytes"
	"fmt"
	"testing"
)

type adapter struct {
	messages []*Message
}

func (a *adapter) Close() error {
	return nil
}

func (a *adapter) Log(m *Message) error {
	a.messages = append(a.messages, m)

	return nil
}

func TestNewLogWriter(t *testing.T) {
	a := &adapter{}
	w := NewLogWriter("test", "stdout", a)

	if w == nil {
		t.Fatal("NewLogWriter returned a nil writer")
	}
}

func TestWriteIDAndSource(t *testing.T) {
	a := &adapter{}
	w := NewLogWriter("test", "stdout", a)

	_, err := fmt.Fprint(w, "test message")
	if err != nil {
		t.Fatal(err)
	}

	if len(a.messages) == 0 {
		t.Fatal("adapter did not receive log message")
	}

	msg := a.messages[0]

	if msg.ID != "test" {
		t.Fatalf("expected log message id to be %q but received %q", "test", msg.ID)
	}

	if msg.Source != "stdout" {
		t.Fatalf("expected log message source to be %q but received %q", "stdout", msg.Source)
	}
}

func TestWriteSize(t *testing.T) {
	a := &adapter{}
	w := NewLogWriter("test", "stdout", a)

	n, err := fmt.Fprint(w, "test message")
	if err != nil {
		t.Fatal(err)
	}

	if len(a.messages) == 0 {
		t.Fatal("adapter did not receive log message")
	}

	msg := a.messages[0]

	if msg.Size() != n {
		t.Fatalf("message size does not match returned size %d != %d", msg.Size(), n)
	}
}

func TestWriteBody(t *testing.T) {
	a := &adapter{}
	w := NewLogWriter("test", "stdout", a)

	_, err := fmt.Fprint(w, "test message")
	if err != nil {
		t.Fatal(err)
	}

	if len(a.messages) == 0 {
		t.Fatal("adapter did not receive log message")
	}

	msg := a.messages[0]

	if bytes.Compare(msg.RawMessage, []byte("test message")) != 0 {
		t.Fatalf("message body does not match sent message %q != %q", string(msg.RawMessage), "test message")
	}
}
