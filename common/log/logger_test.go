package log_test

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
	. "github.com/xtls/xray-core/common/log"
)

func TestFileLogger(t *testing.T) {
	f, err := os.CreateTemp("", "vtest")
	common.Must(err)
	path := f.Name()
	common.Must(f.Close())
	defer os.Remove(path)

	creator, err := FileLogWriterCreator(path)
	common.Must(err)

	handler := NewLogger(creator)
	handler.Handle(&GeneralMessage{Content: "Test Log"})

	// Close returns after the buffered message is written out, so no waiting is needed here.
	common.Must(common.Close(handler))

	f, err = os.Open(path)
	common.Must(err)
	defer f.Close()

	b, err := buf.ReadAllToBytes(f)
	common.Must(err)
	if !strings.Contains(string(b), "Test Log") {
		t.Fatal("Expect log text contains 'Test Log', but actually: ", string(b))
	}
}

func TestFileLoggerWritesBufferedMessagesOnClose(t *testing.T) {
	f, err := os.CreateTemp("", "vtest")
	common.Must(err)
	path := f.Name()
	common.Must(f.Close())
	defer os.Remove(path)

	creator, err := FileLogWriterCreator(path)
	common.Must(err)

	handler := NewLogger(creator)
	const count = 64
	for i := 0; i < count; i++ {
		handler.Handle(&GeneralMessage{Content: fmt.Sprintf("Buffered %d", i)})
	}

	common.Must(common.Close(handler))

	f, err = os.Open(path)
	common.Must(err)
	defer f.Close()

	b, err := buf.ReadAllToBytes(f)
	common.Must(err)
	if n := strings.Count(string(b), "Buffered "); n != count {
		t.Fatal("Expect ", count, " buffered messages, but actually: ", n)
	}
}

type testWriter struct {
	sync.Mutex
	lines  []string
	notify chan struct{}
}

func newTestWriter() *testWriter {
	return &testWriter{notify: make(chan struct{}, 4)}
}

func (w *testWriter) Write(s string) error {
	w.Lock()
	w.lines = append(w.lines, s)
	w.Unlock()

	select {
	case w.notify <- struct{}{}:
	default:
	}
	return nil
}

func (w *testWriter) Close() error {
	return nil
}

func (w *testWriter) Lines() []string {
	w.Lock()
	defer w.Unlock()
	return append([]string(nil), w.lines...)
}

func (w *testWriter) waitForWrite(timeout time.Duration) bool {
	select {
	case <-w.notify:
		return true
	case <-time.After(timeout):
		return false
	}
}

func TestLoggerCloseIsFinal(t *testing.T) {
	w := newTestWriter()
	handler := NewLogger(func() Writer { return w })

	handler.Handle(&GeneralMessage{Content: "Before Close"})
	common.Must(common.Close(handler))

	if !w.waitForWrite(time.Second) {
		t.Fatal("Expect the buffered message to be written on close, but nothing was written")
	}
	if lines := w.Lines(); len(lines) != 1 || !strings.Contains(lines[0], "Before Close") {
		t.Fatal("Expect only the buffered message, but actually: ", lines)
	}

	// A closed logger must not be revived: no message is accepted and no writer is created.
	handler.Handle(&GeneralMessage{Content: "After Close"})
	if w.waitForWrite(500 * time.Millisecond) {
		t.Fatal("Expect nothing written after close, but actually: ", w.Lines())
	}

	// Close is idempotent, and must return at once instead of waiting for the timeout.
	returned := make(chan struct{})
	go func() {
		common.Close(handler)
		close(returned)
	}()
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("Second Close did not return in time")
	}
}
