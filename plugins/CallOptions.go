package plugins

import (
	"encoding/json"
	"io"
)

type CallOptions struct {
	logCapturer LogEventCapturer
}

type CallOption func(CallOptions) CallOptions

type LogEventCapturerFunc func(*LogEntry)

func (l LogEventCapturerFunc) Capture(e *LogEntry) {
	l(e)
}

type logFilterer struct {
	ident      string
	baseLogger io.Writer
}

func (l *logFilterer) Write(p []byte) (int, error) {
	return l.baseLogger.Write(p)
}

func (l *logFilterer) Capture(e *LogEntry) {
	if e.Identifier == l.ident {
		d, _ := json.Marshal(e)
		d = append(d, '\n')
		l.baseLogger.Write(d)
	}

}

func WithLogCapture(w io.Writer, ident string) CallOption {
	logger := &logFilterer{
		ident:      ident,
		baseLogger: w,
	}
	return func(co CallOptions) CallOptions {
		co.logCapturer = logger
		return co
	}
}
