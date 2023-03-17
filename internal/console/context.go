package console

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

const defaultRefreshRate = time.Millisecond

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
					var record Record
					if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
						return err
					}
					if err := reporter.restore(record); err != nil {
						return err
					}
				}
				return nil
			},
			closer: reporter.Stop,
		}
	case JSONFormat:
		reporter := newStructuredReport(writer)
		return &Context{
			options:     options,
			newStatus:   reporter.NewStatus,
			newProgress: reporter.NewProgress,
		}
	default:
		panic("unknown console format")
	}
}

type Context struct {
	options     Options
	restore     func(io.Reader) error
	newStatus   func() statusReport
	newProgress func(string, ...any) progressReport
	closer      func()
}

func (c *Context) Restore(reader io.Reader) error {
	return c.restore(reader)
}

func (c *Context) Fork(desc string, f func(context *Context) error) Joiner {
	report := c.newProgress(desc)
	report.Start()

	context := &Context{
		options:     c.options,
		newStatus:   report.NewStatus,
		newProgress: report.NewProgress,
		restore: func(reader io.Reader) error {
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				var record Record
				if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
					return err
				}
				if err := report.(*liveProgressReport).restore(record); err != nil {
					return err
				}
			}
			return nil
		},
	}

	ch := make(chan error, 1)
	go func() {
		defer close(ch)
		if err := f(context); err != nil {
			report.Error(err)
			ch <- err
		} else {
			report.Done()
		}
	}()
	return newChannelJoiner(ch)
}

func (c *Context) Run(f func(status *Status) error) Waiter {
	report := c.newStatus()
	status := newStatus(report, c.options.Verbose)
	ch := make(chan error, 1)
	go func() {
		defer close(ch)
		if err := f(status); err != nil {
			report.Error(err)
			ch <- err
		} else {
			report.Done()
		}
	}()
	return newChannelWaiter(ch)
}

func (c *Context) Close() {
	if c.closer != nil {
		c.closer()
	}
}

type Joiner interface {
	Join() error
}

func Join(joiners ...Joiner) error {
	var err error
	for _, joiner := range joiners {
		if e := joiner.Join(); e != nil {
			err = e
		}
	}
	return err
}

func newChannelJoiner(ch <-chan error) Joiner {
	return &channelJoiner{
		ch: ch,
	}
}

type channelJoiner struct {
	ch <-chan error
}

func (w *channelJoiner) Join() error {
	return <-w.ch
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

func newStatus(report statusReport, verbose bool) *Status {
	return &Status{
		report:  report,
		writer:  newStatusReportWriter(report),
		verbose: verbose,
	}
}

type Status struct {
	report  statusReport
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

func newStatusReportWriter(report statusReport) io.Writer {
	return &statusReportWriter{
		report: report,
	}
}

type statusReportWriter struct {
	report statusReport
	buf    bytes.Buffer
}

func (w *statusReportWriter) Write(bytes []byte) (n int, err error) {
	i, err := w.buf.Write(bytes)
	if err == nil {
		w.report.Update(w.buf.String())
	}
	return i, err
}
