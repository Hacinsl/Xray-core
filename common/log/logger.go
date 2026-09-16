package log

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/xtls/xray-core/common/platform"
	"github.com/xtls/xray-core/common/signal/done"
	"github.com/xtls/xray-core/common/signal/semaphore"
)

// Writer is the interface for writing logs.
type Writer interface {
	Write(string) error
	io.Closer
}

// WriterCreator is a function to create LogWriters.
type WriterCreator func() Writer

type generalLogger struct {
	creator WriterCreator
	buffer  chan Message
	access  *semaphore.Instance
	done    *done.Instance
}

type defaultLogger struct {
	writer Writer
}

type serverityLogger struct {
	inner    *generalLogger
	logLevel Severity
}

// NewLogger returns a generic log handler that can handle all type of messages.
func NewLogger(logWriterCreator WriterCreator) Handler {
	return &generalLogger{
		creator: logWriterCreator,
		buffer:  make(chan Message, 128),
		access:  semaphore.New(1),
		done:    done.New(),
	}
}

// NewDefaultLogger returns a direct log handler that can handle all type of messages.
func NewDefaultLogger(creator WriterCreator) Handler {
	w := creator()
	if w == nil {
		w = StdoutLogWriterCreator()() // Use console as fallback.
	}
	return &defaultLogger{writer: w}
}

func (l *defaultLogger) Handle(msg Message) {
	_ = l.writer.Write(msg.String() + platform.LineSeparator())
}

func ReplaceWithSeverityLogger(serverity Severity) {
	w := StdoutLogWriterCreator()
	g := &generalLogger{
		creator: w,
		buffer:  make(chan Message, 128),
		access:  semaphore.New(1),
		done:    done.New(),
	}
	s := &serverityLogger{
		inner:    g,
		logLevel: serverity,
	}
	// RegisterHandler(s) // This line should be removed once all pre-build configuration log outputs have been migrated to the default logger.
	RegisterDefaultHandler(s)
}

// Only for ReplaceWithSeverityLogger, which only used in -dump.
func (l *serverityLogger) Handle(msg Message) {
	switch msg := msg.(type) {
	case *GeneralMessage:
		if msg.Severity <= l.logLevel {
			l.inner.Handle(msg)
		}
	default:
		l.inner.Handle(msg)
	}
}

func (l *generalLogger) run() {
	defer l.access.Signal()

	dataWritten := false
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	logger := l.creator()
	if logger == nil {
		return
	}
	defer logger.Close()

	for {
		select {
		case <-l.done.Wait():
			// The logger is closed. Write out every accepted message that is still buffered
			// before the writer is closed by the deferred call above, so that nothing is lost.
			l.flush(logger)
			return
		case msg := <-l.buffer:
			logger.Write(msg.String() + platform.LineSeparator())
			dataWritten = true
		case <-ticker.C:
			if !dataWritten {
				return
			}
			dataWritten = false
		}
	}
}

// flush writes out all messages that are buffered at the moment of the call. It returns once the
// buffer is empty.
func (l *generalLogger) flush(w Writer) {
	for {
		select {
		case msg := <-l.buffer:
			w.Write(msg.String() + platform.LineSeparator())
		default:
			return
		}
	}
}

func (l *generalLogger) Handle(msg Message) {
	// A closed logger must not accept new messages, otherwise run() would be started again and
	// the writer would be created once more.
	if l.done.Done() {
		return
	}

	select {
	case l.buffer <- msg:
	default:
		os.Stderr.Write([]byte("Log buffer is full. New Log has been dropped."))
	}

	select {
	case <-l.access.Wait():
		go l.run()
	default:
	}
}

// Close stops the logger. It returns after the buffered messages have been written out and the
// writer has been closed, or after closeTimeout if the writer is stuck. Close is idempotent and
// must not be called concurrently with Handle.
func (l *generalLogger) Close() error {
	if l.done.Done() {
		return nil
	}
	_ = l.done.Close()

	// run() holds the access permit for its whole lifetime and returns it only after flushing the
	// buffer and closing the writer. Acquiring the permit therefore means that the last writer is
	// gone, and that Handle() will not start a new one.
	select {
	case <-l.access.Wait():
	case <-time.After(5 * time.Second):
	}
	return nil
}

type consoleLogWriter struct {
	logger *log.Logger
}

func (w *consoleLogWriter) Write(s string) error {
	w.logger.Print(s)
	return nil
}

func (w *consoleLogWriter) Close() error {
	return nil
}

type fileLogWriter struct {
	file   *os.File
	logger *log.Logger
}

func (w *fileLogWriter) Write(s string) error {
	w.logger.Print(s)
	return nil
}

func (w *fileLogWriter) Close() error {
	return w.file.Close()
}

// StdoutLogWriterCreator returns a LogWriterCreator that creates LogWriter for stdout.
func StdoutLogWriterCreator() WriterCreator {
	return func() Writer {
		return &consoleLogWriter{
			logger: log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds),
		}
	}
}

// StderrLogWriterCreator returns a LogWriterCreator that creates LogWriter for stderr.
func StderrLogWriterCreator() WriterCreator {
	return func() Writer {
		return &consoleLogWriter{
			logger: log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lmicroseconds),
		}
	}
}

// FileLogWriterCreator returns a LogWriterCreator that creates LogWriter for the given file.
func FileLogWriterCreator(path string) (WriterCreator, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	file.Close()
	return func() Writer {
		file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
		if err != nil {
			return nil
		}
		return &fileLogWriter{
			file:   file,
			logger: log.New(file, "", log.Ldate|log.Ltime|log.Lmicroseconds),
		}
	}, nil
}

func init() {
	RegisterDefaultHandler(NewDefaultLogger(StdoutLogWriterCreator()))
}
