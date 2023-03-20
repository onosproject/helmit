package task

import (
	"bytes"
	"fmt"
	"github.com/onosproject/helmit/internal/log"
	"io"
)

func New(writer log.Writer, message string, args ...any) Task {
	name := fmt.Sprintf(message, args...)
	_ = writer.WriteRecord(log.TaskAdded(nil, name))
	return newTask(writer, []string{name})
}

type Context interface {
	log.Logger
	Status() Status
	Writer() io.Writer
	NewTask(message string, args ...any) Task
}

func newContext(log log.Writer, path []string) Context {
	return &taskContext{
		log:    log,
		path:   path,
		status: newStatus(log, path),
		writer: newWriter(log, path),
	}
}

type taskContext struct {
	log    log.Writer
	path   []string
	status Status
	writer io.Writer
}

func (c *taskContext) Log(message string) {
	buf := bytes.NewBufferString(message)
	buf.WriteByte('\n')
	_, _ = c.writer.Write(buf.Bytes())
}

func (c *taskContext) Logf(format string, args ...any) {
	c.Log(fmt.Sprintf(format, args...))
}

func (c *taskContext) Status() Status {
	return c.status
}

func (c *taskContext) Writer() io.Writer {
	return c.writer
}

func (c *taskContext) NewTask(message string, args ...any) Task {
	name := fmt.Sprintf(message, args...)
	_ = c.log.WriteRecord(log.TaskAdded(c.path, name))
	return newTask(c.log, append(c.path, name))
}

type Task interface {
	Start(func(Context) error) Future
	Run(func(Context) error) error
	Cancel()
}

func newTask(log log.Writer, path []string) Task {
	return &managedTask{
		log:  log,
		path: path,
	}
}

type managedTask struct {
	log  log.Writer
	path []string
}

func (t *managedTask) Start(f func(Context) error) Future {
	ch := make(chan error)
	go func() {
		ch <- t.Run(f)
	}()
	return newChannelFuture(ch)
}

func (t *managedTask) Run(f func(Context) error) error {
	_ = t.log.WriteRecord(log.TaskStarted(t.path))
	err := f(newContext(t.log, t.path))
	if err != nil {
		_ = t.log.WriteRecord(log.TaskFailed(t.path, err.Error()))
	} else {
		_ = t.log.WriteRecord(log.TaskComplete(t.path))
	}
	return err
}

func (t *managedTask) Cancel() {
	_ = t.log.WriteRecord(log.TaskCanceled(t.path))
}

func Await(futures ...Future) error {
	for _, future := range futures {
		if err := future.Await(); err != nil {
			return err
		}
	}
	return nil
}

type Future interface {
	Await() error
}

func newChannelFuture(ch <-chan error) Future {
	return &channelFuture{
		ch: ch,
	}
}

type channelFuture struct {
	ch <-chan error
}

func (f *channelFuture) Await() error {
	return <-f.ch
}

type Status interface {
	Set(message string)
	Setf(message string, args ...any)
}

func newStatus(log log.Writer, path []string) Status {
	return &taskStatus{
		log:  log,
		path: path,
	}
}

type taskStatus struct {
	log  log.Writer
	path []string
}

func (c *taskStatus) Set(status string) {
	_ = c.log.WriteRecord(log.TaskStatus(c.path, status))
}

func (c *taskStatus) Setf(status string, args ...any) {
	_ = c.log.WriteRecord(log.TaskStatus(c.path, fmt.Sprintf(status, args...)))
}

func newWriter(log log.Writer, path []string) io.Writer {
	return &taskWriter{
		log:  log,
		path: path,
	}
}

type taskWriter struct {
	log  log.Writer
	path []string
}

func (w *taskWriter) Write(data []byte) (int, error) {
	err := w.log.WriteRecord(log.TaskOutput(w.path, data))
	return len(data), err
}
