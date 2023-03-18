package console

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/onosproject/helmit/internal/async"
	"io"
	"time"
)

const defaultRefreshRate = 5 * time.Millisecond

type Format int

const (
	LiveFormat Format = iota
	JSONFormat
)

type Option func(*Options)

func WithRefreshRate(d time.Duration) Option {
	return func(options *Options) {
		options.RefreshRate = d
	}
}

func WithVerbose() Option {
	return func(options *Options) {
		options.Verbose = true
	}
}

func WithFormat(format Format) Option {
	return func(options *Options) {
		options.Format = format
	}
}

type Options struct {
	Format      Format
	RefreshRate time.Duration
	Verbose     bool
}

func NewContext(writer io.Writer, opts ...Option) *Context {
	options := Options{
		Format:      LiveFormat,
		RefreshRate: defaultRefreshRate,
	}
	for _, opt := range opts {
		opt(&options)
	}

	switch options.Format {
	case LiveFormat:
		reporter := newLiveReporter(writer, options.RefreshRate)
		reporter.Start()
		return &Context{
			options:     options,
			newStatus:   reporter.NewStatus,
			newProgress: reporter.NewProgress,
			restore: func(reader io.Reader) error {
				scanner := bufio.NewScanner(reader)
				for scanner.Scan() {
					var entry reportEntry
					if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
						return err
					}
					if err := reporter.restore(entry); err != nil {
						return err
					}
				}
				return nil
			},
			closer: reporter.Stop,
		}
	case JSONFormat:
		reporter := newJSONReporter(writer)
		return &Context{
			options:     options,
			newStatus:   reporter.NewStatus,
			newProgress: reporter.NewProgress,
			restore: func(reader io.Reader) error {
				return errors.New("restore not supported in JSON format")
			},
		}
	default:
		panic("unknown console format")
	}
}

type Context struct {
	options     Options
	restore     func(io.Reader) error
	newStatus   func() StatusReport
	newProgress func(string, ...any) ProgressReport
	closer      func()
}

func (c *Context) Restore(reader io.Reader) error {
	return c.restore(reader)
}

func (c *Context) Fork(desc string, f func(context *Context) error) Fork {
	return newFork(c, c.newProgress(desc), f)
}

func (c *Context) Run(f func(status *Status) error) Task {
	return newTask(c, c.newStatus(), f)
}

func (c *Context) Close() {
	if c.closer != nil {
		c.closer()
	}
}

type Fork interface {
	Join() error
}

func Join(forks ...Fork) error {
	return async.IterAsync(len(forks), func(i int) error {
		return forks[i].Join()
	})
}

func newFork(context *Context, report ProgressReport, f func(context *Context) error) Fork {
	return &contextFork{
		context: context,
		report:  report,
		f:       f,
	}
}

type contextFork struct {
	report  ProgressReport
	context *Context
	f       func(context *Context) error
}

func (f *contextFork) Join() error {
	f.report.Start()
	context := &Context{
		options:     f.context.options,
		newStatus:   f.report.NewStatus,
		newProgress: f.report.NewProgress,
		restore: func(reader io.Reader) error {
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				var entry reportEntry
				if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
					return err
				}
				if err := f.report.(*liveProgressReport).restore(entry); err != nil {
					return err
				}
			}
			return nil
		},
	}
	if err := f.f(context); err != nil {
		f.report.Error(err)
		return err
	}
	f.report.Finish()
	return nil
}

type Task interface {
	Await() error
}

func newTask(context *Context, report StatusReport, f func(status *Status) error) Task {
	return &contextTask{
		context: context,
		report:  report,
		f:       f,
	}
}

type contextTask struct {
	context *Context
	report  StatusReport
	f       func(status *Status) error
}

func (t *contextTask) Await() error {
	status := newStatus(t.report, t.context.options.Verbose)
	if err := t.f(status); err != nil {
		t.report.Error(err)
		return err
	}
	t.report.Done()
	return nil
}

func Await(tasks ...Task) error {
	return async.IterAsync(len(tasks), func(i int) error {
		return tasks[i].Await()
	})
}

func newStatus(report StatusReport, verbose bool) *Status {
	return &Status{
		report:  report,
		writer:  newStatusReportWriter(report),
		verbose: verbose,
	}
}

type Status struct {
	report  StatusReport
	writer  io.Writer
	verbose bool
}

func (s *Status) Writer() io.Writer {
	return s.writer
}

func (s *Status) Report(message string) {
	s.report.Update(message)
}

func (s *Status) Reportf(message string, args ...any) {
	s.report.Update(fmt.Sprintf(message, args...))
}

func (s *Status) Log(message string) {
	s.log(message)
}

func (s *Status) Logf(message string, args ...any) {
	s.log(fmt.Sprintf(message, args...))
}

func (s *Status) log(message string) {
	if s.verbose {
		buf := bytes.NewBufferString(message)
		if buf.Len() == 0 || buf.Bytes()[buf.Len()-1] != '\n' {
			buf.WriteByte('\n')
		}
		_, _ = s.writer.Write(buf.Bytes())
	}
}

func newStatusReportWriter(report StatusReport) io.Writer {
	return &statusReportWriter{
		report: report,
	}
}

type statusReportWriter struct {
	report StatusReport
	buf    bytes.Buffer
}

func (w *statusReportWriter) Write(bytes []byte) (n int, err error) {
	i, err := w.buf.Write(bytes)
	if err == nil {
		w.report.Update(w.buf.String())
	}
	return i, err
}
