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
	errorMsgColor = color.New(color.FgRed)
	errorErrColor = color.New(color.FgRed, color.Bold)
)

type reportWriter interface {
	write(writer *uilive.Writer, depth int)
}

func newReporter(writer io.Writer, rate time.Duration) *Reporter {
	lwriter := uilive.New()
	lwriter.Out = writer
	return &Reporter{
		writer: lwriter,
		rate:   rate,
	}
}

type Reporter struct {
	writer  *uilive.Writer
	rate    time.Duration
	reports []reportWriter
	ticker  *time.Ticker
	stop    chan struct{}
	stopped chan struct{}
	mu      sync.RWMutex
}

func (r *Reporter) NewStatus() *StatusReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	status := newStatusReport()
	r.reports = append(r.reports, status)
	return status
}

func (r *Reporter) NewProgress(msg string, args ...any) *ProgressReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	progress := newProgressReport(msg, args...)
	r.reports = append(r.reports, progress)
	return progress
}

func (r *Reporter) Start() {
	r.ticker = time.NewTicker(r.rate)
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
	for _, task := range r.reports {
		task.write(r.writer, 0)
	}
	r.writer.Flush()
}

func (r *Reporter) Stop() {
	r.ticker.Stop()
	close(r.stop)
	<-r.stopped
}

func newProgressReport(msg string, args ...any) *ProgressReport {
	var desc string
	if len(args) > 0 {
		desc = fmt.Sprintf(msg, args...)
	} else {
		desc = msg
	}
	return &ProgressReport{
		desc: desc,
	}
}

type ProgressReport struct {
	desc     string
	children []reportWriter
	start    time.Time
	done     bool
	err      error
	mu       sync.RWMutex
}

func (r *ProgressReport) NewProgress(msg string, args ...any) *ProgressReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	progress := newProgressReport(msg, args...)
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
	}
}

func (r *ProgressReport) write(writer *uilive.Writer, depth int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.done {
		if r.err != nil {
			fmt.Fprintf(writer.Newline(), "%s%s %s\n", strings.Repeat(" ", depth*2), errorMsgColor.Sprintf(" ✘ %s", r.desc), errorErrColor.Sprintf("← %s", r.err.Error()))
		} else {
			fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*2), doneMsgColor.Sprintf(" ✔ %s", r.desc))
		}
	} else {
		frameIndex := int(time.Since(r.start)/spinnerSpeed) % len(spinnerFrames)
		spinnerFrame := spinnerFrames[frameIndex]
		fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*2), taskMsgColor.Sprintf("%s %s", spinnerFrame, r.desc))
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
	lines := strings.Split(*value, "\n")
	for _, line := range lines {
		if line != "" {
			fmt.Fprintf(writer.Newline(), "%s %s\n", strings.Repeat(" ", depth*2), line)
		}
	}
}
