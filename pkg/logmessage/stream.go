package logmessage

import "io"

// LogWriter
type LogWriter struct {
	ID     string
	Source string

	adapter Logger
}

// NewLogWriter returns an io.Writer than is able to translate writes into
// discrete log messages for the provided Adapter
func NewLogWriter(id, source string, adapter Logger) io.WriteCloser {
	return &LogWriter{
		ID:      id,
		Source:  source,
		adapter: adapter,
	}
}

func (l *LogWriter) Write(p []byte) (int, error) {
	m := NewMessage(l.ID, l.Source, p)

	if err := l.adapter.Log(m); err != nil {
		return m.Size(), err
	}

	return m.Size(), nil
}

func (l *LogWriter) Close() error {
	return l.adapter.Close()
}
