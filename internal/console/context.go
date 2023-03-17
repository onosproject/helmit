package console

import (
	"bytes"
	"fmt"
	"io"
)

func NewContext(reporter *Reporter) *Context {
	return &Context{
		newProgress: reporter.NewProgress,
	}
}

type Context struct {
	newProgress func(string, ...any) *ProgressReport
}

func (c *Context) Run(desc string, f func(task *Task) error) error {
	progress := c.newProgress(desc)
	progress.Start()
	if err := f(newTask(progress)); err != nil {
		progress.Error(err)
		return err
	}
	progress.Done()
	return nil
}

func (c *Context) RunAsync(desc string, f func(task *Task) error) Waiter {
	progress := c.newProgress(desc)
	progress.Start()
	ch := make(chan error, 1)
	go func() {
		defer close(ch)
		if err := f(newTask(progress)); err != nil {
			progress.Error(err)
			ch <- err
		} else {
			progress.Done()
		}
	}()
	return newChannelWaiter(ch)
}

func newTask(progress *ProgressReport) *Task {
	return &Task{
		Context: &Context{
			newProgress: progress.NewProgress,
		},
		progress: progress,
		writer:   newStatusReportWriter(progress.StatusReport),
	}
}

type Task struct {
	*Context
	progress *ProgressReport
	writer   io.Writer
}

func (t *Task) Writer() io.Writer {
	return t.writer
}

func (t *Task) Log(message string) {
	t.log(message)
}

func (t *Task) Logf(message string, args ...any) {
	t.log(fmt.Sprintf(message, args...))
}

func (t *Task) log(message string) {
	buf := bytes.NewBufferString(message)
	if buf.Len() == 0 || buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	_, _ = t.writer.Write(buf.Bytes())
}

func (t *Task) Fork(f func(status *Status) error) Waiter {
	status := t.progress.NewStatus()
	ch := make(chan error, 1)
	go func() {
		defer close(ch)
		if err := f(newStatus(status)); err != nil {
			ch <- err
		}
		status.Done()
	}()
	return newChannelWaiter(ch)
}

type Waiter interface {
	Wait() error
}

func Wait(waiters ...Waiter) error {
	var err error
	for _, waiter := range waiters {
		if e := waiter.Wait(); e != nil {
			err = e
		}
	}
	return err
}

func newChannelWaiter(ch <-chan error) Waiter {
	return &channelWaiter{
		ch: ch,
	}
}

type channelWaiter struct {
	ch <-chan error
}

func (w *channelWaiter) Wait() error {
	return <-w.ch
}

func newStatus(report *StatusReport) *Status {
	return &Status{
		report: report,
	}
}

type Status struct {
	report *StatusReport
}

func (s *Status) Report(message string) {
	s.report.Update(message)
}

func (s *Status) Reportf(message string, args ...any) {
	s.report.Update(fmt.Sprintf(message, args...))
}

func newStatusReportWriter(report *StatusReport) io.Writer {
	return &statusReportWriter{
		report: report,
	}
}

type statusReportWriter struct {
	report *StatusReport
	buf    bytes.Buffer
}

func (w *statusReportWriter) Write(bytes []byte) (n int, err error) {
	i, err := w.buf.Write(bytes)
	if err == nil {
		w.report.Update(w.buf.String())
	}
	return i, err
}
