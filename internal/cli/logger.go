package cli

import (
	"bytes"
	"fmt"
	"github.com/briandowns/spinner"
	"io"
	"sync"
)

// NewLogger creates a new CLI logger
func NewLogger(writer io.Writer) *Logger {
	return &Logger{
		writer:     writer,
		bufferPool: newBufferPool(),
	}
}

// Logger provides logging with spinners for the CLI
type Logger struct {
	writer     io.Writer
	writerMu   sync.Mutex
	bufferPool *bufferPool
}

// Log logs a message
func (l *Logger) Log(message string) {
	l.print(message)
}

// Logf logs a formatted message
func (l *Logger) Logf(format string, args ...interface{}) {
	l.printf(format, args...)
}

// print writes a simple string to the log writer
func (l *Logger) print(message string) {
	buf := bytes.NewBufferString(message)
	l.writeBuffer(buf)
}

// printf is roughly fmt.Fprintf against the log writer
func (l *Logger) printf(format string, args ...interface{}) {
	buf := l.bufferPool.Get()
	fmt.Fprintf(buf, format, args...)
	l.writeBuffer(buf)
	l.bufferPool.Put(buf)
}

// synchronized write to the inner writer
func (l *Logger) write(p []byte) (n int, err error) {
	l.writerMu.Lock()
	defer l.writerMu.Unlock()
	return l.writer.Write(p)
}

// writeBuffer writes buf with write, ensuring there is a trailing newline
func (l *Logger) writeBuffer(buf *bytes.Buffer) {
	// ensure trailing newline
	if buf.Len() == 0 || buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	// TODO: should we handle this somehow??
	// Who logs for the logger? 🤔
	_, _ = l.write(buf.Bytes())
}

// Task creates a new CLI task
func (l *Logger) Task(desc string) *Task {
	return &Task{
		spinner: spinner.New(spinnerCharSet, spinnerSpeed, spinner.WithWriter(l.writer), spinner.WithColor("blue"), spinner.WithSuffix(taskMsgColor.Sprintf(" %s", desc))),
		header:  desc,
		lines:   make(map[int]string),
	}
}

// bufferPool is a type safe sync.Pool of *byte.Buffer, guaranteed to be Reset
type bufferPool struct {
	sync.Pool
}

// newBufferPool returns a new bufferPool
func newBufferPool() *bufferPool {
	return &bufferPool{
		sync.Pool{
			New: func() interface{} {
				// The Pool's New function should generally only return pointer
				// types, since a pointer can be put into the return interface
				// value without an allocation:
				return new(bytes.Buffer)
			},
		},
	}
}

// Get obtains a buffer from the pool
func (b *bufferPool) Get() *bytes.Buffer {
	return b.Pool.Get().(*bytes.Buffer)
}

// Put returns a buffer to the pool, resetting it first
func (b *bufferPool) Put(x *bytes.Buffer) {
	// only store small buffers to avoid pointless allocation
	// avoid keeping arbitrarily large buffers
	if x.Len() > 256 {
		return
	}
	x.Reset()
	b.Pool.Put(x)
}
