package console

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/gosuri/uilive"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var spinnerFrames = []string{
	"⠈⠁",
	"⠈⠑",
	"⠈⠱",
	"⠈⡱",
	"⢀⡱",
	"⢄⡱",
	"⢄⡱",
	"⢆⡱",
	"⢎⡱",
	"⢎⡰",
	"⢎⡠",
	"⢎⡀",
	"⢎⠁",
	"⠎⠁",
	"⠊⠁",
}

const spinnerSpeed = 150 * time.Millisecond

var (
	taskMsgColor  = color.New(color.FgBlue)
	doneMsgColor  = color.New(color.FgGreen)
	errorMsgColor = color.New(color.FgRed, color.Bold)
	errorErrColor = color.New(color.FgRed, color.Italic)
)

const defaultRefreshRate = time.Millisecond

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

type Options struct {
	RefreshRate time.Duration
	Verbose     bool
}

func NewReporter(writer io.Writer, opts ...Option) *Reporter {
	options := Options{
		RefreshRate: defaultRefreshRate,
	}
	for _, opt := range opts {
		opt(&options)
	}

	lwriter := uilive.New()
	lwriter.Out = writer
	lwriter.RefreshInterval = time.Hour

	return &Reporter{
		Options: options,
		writer:  lwriter,
	}
}

type Reporter struct {
	Options
	writer   *uilive.Writer
	progress []*ProgressReport
	ticker   *time.Ticker
	stop     chan struct{}
	stopped  chan struct{}
	mu       sync.RWMutex
}

func (r *Reporter) NewProgress(msg string, args ...any) *ProgressReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	progress := newProgressReport(r.Options, msg, args...)
	r.progress = append(r.progress, progress)
	return progress
}

func (r *Reporter) Start() {
	r.ticker = time.NewTicker(r.RefreshRate)
	r.stop = make(chan struct{})
	r.stopped = make(chan struct{})
	go r.run()
}

func (r *Reporter) run() {
	for {
		select {
		case <-r.ticker.C:
			r.write()
		case <-r.stop:
			r.write()
			close(r.stopped)
			return
		}
	}
}

func (r *Reporter) write() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, task := range r.progress {
		task.write(r.writer, 0)
	}
	r.writer.Flush()
}

func (r *Reporter) Stop() {
	r.ticker.Stop()
	close(r.stop)
	<-r.stopped
}

type statusWriter interface {
	write(writer *uilive.Writer, depth int)
}

func newProgressReport(options Options, msg string, args ...any) *ProgressReport {
	var desc string
	if len(args) > 0 {
		desc = fmt.Sprintf(msg, args...)
	} else {
		desc = msg
	}
	return &ProgressReport{
		StatusReport: newStatusReport(),
		options:      options,
		desc:         desc,
	}
}

type ProgressReport struct {
	*StatusReport
	options  Options
	desc     string
	children []statusWriter
	start    time.Time
	done     bool
	err      error
	closer   func(*ProgressReport)
	mu       sync.RWMutex
}

func (r *ProgressReport) NewProgress(msg string, args ...any) *ProgressReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	progress := newProgressReport(r.options, msg, args...)
	r.children = append(r.children, progress)
	return progress
}

func (r *ProgressReport) NewStatus() *StatusReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	detail := newStatusReport()
	r.children = append(r.children, detail)
	return detail
}

func (r *ProgressReport) Start() {
	r.start = time.Now()
}

func (r *ProgressReport) Done() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.close(nil)
}

func (r *ProgressReport) Error(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.close(err)
}

func (r *ProgressReport) close(err error) {
	if !r.done {
		r.done = true
		r.err = err
		if r.closer != nil {
			r.closer(r)
		}
	}
}

func (r *ProgressReport) write(writer *uilive.Writer, depth int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.done {
		if r.err != nil {
			fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*2), errorMsgColor.Sprintf(" ✘ %s", r.desc))
			fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*2+2), errorErrColor.Sprint(r.err.Error()))
		} else {
			fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*2), doneMsgColor.Sprintf(" ✔ %s", r.desc))
		}
	} else {
		frameIndex := int(time.Since(r.start)/spinnerSpeed) % len(spinnerFrames)
		spinnerFrame := spinnerFrames[frameIndex]
		fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*2), taskMsgColor.Sprintf("%s %s", spinnerFrame, r.desc))
	}

	if r.options.Verbose {
		r.StatusReport.write(writer, depth)
	}

	for _, child := range r.children {
		child.write(writer, depth+1)
	}
}

func newStatusReport() *StatusReport {
	return &StatusReport{}
}

// StatusReport is a ProgressReport status report
type StatusReport struct {
	value atomic.Pointer[string]
}

// Update updates the report
func (r *StatusReport) Update(contents string) {
	r.value.Store(&contents)
}

// Done marks the sub-task complete
func (r *StatusReport) Done() {
	r.value = atomic.Pointer[string]{}
}

func (r *StatusReport) write(writer *uilive.Writer, depth int) {
	value := r.value.Load()
	if value == nil {
		return
	}
	fmt.Fprintf(writer.Newline(), "%s %s\n", strings.Repeat(" ", depth*2), *value)
}
