package console

import (
	"bytes"
	"encoding/json"
	"errors"
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

func newLiveReporter(writer io.Writer, rate time.Duration) *liveReporter {
	lwriter := uilive.New()
	lwriter.Out = writer
	return &liveReporter{
		writer: lwriter,
		rate:   rate,
	}
}

type liveReporter struct {
	writer  *uilive.Writer
	rate    time.Duration
	reports []reportWriter
	ticker  *time.Ticker
	stopCh  chan struct{}
	stopped chan struct{}
	mu      sync.RWMutex
}

func (r *liveReporter) restore(record Record) error {
	if record.NewProgress != nil {
		if len(record.NewProgress.Address) == 0 {
			r.NewProgress(record.NewProgress.Message).Start()
		} else {
			progress := r.reports[record.NewProgress.Address[0]].(*liveProgressReport)
			record.NewProgress.Address = record.NewProgress.Address[1:]
			return progress.restore(record)
		}
	} else if record.ProgressDone != nil {
		progress := r.reports[record.ProgressDone.Address[0]].(*liveProgressReport)
		record.ProgressDone.Address = record.ProgressDone.Address[1:]
		return progress.restore(record)
	} else if record.ProgressError != nil {
		progress := r.reports[record.ProgressError.Address[0]].(*liveProgressReport)
		record.ProgressError.Address = record.ProgressError.Address[1:]
		return progress.restore(record)
	} else if record.NewStatus != nil {
		if len(record.NewStatus.Address) == 0 {
			r.NewStatus()
		} else {
			progress := r.reports[record.NewStatus.Address[0]].(*liveProgressReport)
			record.NewStatus.Address = record.NewStatus.Address[1:]
			return progress.restore(record)
		}
	} else if record.StatusDone != nil {
		progress := r.reports[record.StatusDone.Address[0]].(*liveProgressReport)
		record.StatusDone.Address = record.StatusDone.Address[1:]
		return progress.restore(record)
	} else if record.StatusError != nil {
		progress := r.reports[record.StatusError.Address[0]].(*liveProgressReport)
		record.StatusError.Address = record.StatusError.Address[1:]
		return progress.restore(record)
	}
	return nil
}

func (r *liveReporter) NewStatus() statusReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	status := newLiveStatusReport()
	r.reports = append(r.reports, status)
	return status
}

func (r *liveReporter) NewProgress(msg string, args ...any) progressReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	progress := newLiveProgressReport(msg, args...)
	r.reports = append(r.reports, progress)
	return progress
}

func (r *liveReporter) Start() {
	r.ticker = time.NewTicker(r.rate)
	r.stopCh = make(chan struct{})
	r.stopped = make(chan struct{})
	go r.run()
}

func (r *liveReporter) run() {
	for {
		select {
		case <-r.ticker.C:
			r.write()
		case <-r.stopCh:
			r.write()
			close(r.stopped)
			return
		}
	}
}

func (r *liveReporter) write() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, report := range r.reports {
		report.write(r.writer, 0)
	}
	r.writer.Flush()
}

func (r *liveReporter) Stop() {
	r.ticker.Stop()
	close(r.stopCh)
	<-r.stopped
}

type progressReport interface {
	NewProgress(msg string, args ...any) progressReport
	NewStatus() statusReport
	Start()
	Done()
	Error(err error)
}

func newLiveProgressReport(msg string, args ...any) *liveProgressReport {
	var desc string
	if len(args) > 0 {
		desc = fmt.Sprintf(msg, args...)
	} else {
		desc = msg
	}
	return &liveProgressReport{
		desc: desc,
	}
}

type liveProgressReport struct {
	desc     string
	children []reportWriter
	start    time.Time
	done     bool
	err      error
	mu       sync.RWMutex
}

func (r *liveProgressReport) restore(record Record) error {
	if record.NewProgress != nil {
		if len(record.NewProgress.Address) == 0 {
			r.NewProgress(record.NewProgress.Message).Start()
		} else {
			progress := r.children[record.NewProgress.Address[0]].(*liveProgressReport)
			record.NewProgress.Address = record.NewProgress.Address[1:]
			return progress.restore(record)
		}
	} else if record.ProgressDone != nil {
		if len(record.ProgressDone.Address) == 1 {
			r.children[record.ProgressDone.Address[0]].(*liveProgressReport).Done()
		} else {
			progress := r.children[record.ProgressDone.Address[0]].(*liveProgressReport)
			record.ProgressDone.Address = record.ProgressDone.Address[1:]
			return progress.restore(record)
		}
	} else if record.ProgressError != nil {
		if len(record.ProgressError.Address) == 1 {
			r.children[record.ProgressError.Address[0]].(*liveProgressReport).Error(errors.New(record.ProgressError.Message))
		} else {
			progress := r.children[record.ProgressError.Address[0]].(*liveProgressReport)
			record.ProgressError.Address = record.ProgressError.Address[1:]
			return progress.restore(record)
		}
	} else if record.NewStatus != nil {
		if len(record.NewStatus.Address) == 0 {
			r.NewStatus()
		} else {
			progress := r.children[record.NewStatus.Address[0]].(*liveProgressReport)
			record.NewStatus.Address = record.NewStatus.Address[1:]
			return progress.restore(record)
		}
	} else if record.StatusDone != nil {
		if len(record.StatusDone.Address) == 1 {
			r.children[record.StatusDone.Address[0]].(*liveStatusReport).Done()
		} else {
			progress := r.children[record.StatusDone.Address[0]].(*liveProgressReport)
			record.StatusDone.Address = record.StatusDone.Address[1:]
			return progress.restore(record)
		}
	} else if record.StatusError != nil {
		if len(record.StatusError.Address) == 1 {
			r.children[record.StatusError.Address[0]].(*liveStatusReport).Error(errors.New(record.StatusError.Message))
		} else {
			progress := r.children[record.StatusError.Address[0]].(*liveProgressReport)
			record.StatusError.Address = record.StatusError.Address[1:]
			return progress.restore(record)
		}
	}
	return nil
}

func (r *liveProgressReport) NewProgress(msg string, args ...any) progressReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	progress := newLiveProgressReport(msg, args...)
	r.children = append(r.children, progress)
	return progress
}

func (r *liveProgressReport) NewStatus() statusReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	detail := newLiveStatusReport()
	r.children = append(r.children, detail)
	return detail
}

func (r *liveProgressReport) Start() {
	r.start = time.Now()
}

func (r *liveProgressReport) Done() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.close(nil)
}

func (r *liveProgressReport) Error(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.close(err)
}

func (r *liveProgressReport) close(err error) {
	if !r.done {
		r.done = true
		r.err = err
	}
}

func (r *liveProgressReport) write(writer *uilive.Writer, depth int) {
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

type statusReport interface {
	Update(message string)
	Done()
	Error(err error)
}

func newLiveStatusReport() *liveStatusReport {
	return &liveStatusReport{}
}

// liveStatusReport is a liveProgressReport status report
type liveStatusReport struct {
	value atomic.Pointer[string]
}

// Update updates the report
func (r *liveStatusReport) Update(contents string) {
	r.value.Store(&contents)
}

// Done marks the sub-task complete
func (r *liveStatusReport) Done() {
	r.value = atomic.Pointer[string]{}
}

func (r *liveStatusReport) Error(err error) {
	value := r.value.Load()
	if value != nil {
		r.Update(errorErrColor.Sprintf(" %s ← %s", *value, err.Error()))
	}
}

func (r *liveStatusReport) write(writer *uilive.Writer, depth int) {
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

type structuredLogger struct {
	writer io.Writer
}

func (l *structuredLogger) Log(record Record) error {
	data, err := json.Marshal(&record)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(data)
	buf.WriteByte('\n')
	_, err = l.writer.Write(buf.Bytes())
	return err
}

func newStructuredReport(writer io.Writer) progressReport {
	return &structuredProgressReport{
		logger: &structuredLogger{
			writer: writer,
		},
		address: []int{},
	}
}

func newStructuredProgressReport(logger *structuredLogger, address []int, message string) progressReport {
	return &structuredProgressReport{
		logger:  logger,
		address: address,
		message: message,
	}
}

type structuredProgressReport struct {
	logger   *structuredLogger
	address  []int
	message  string
	children int
	mu       sync.Mutex
}

func (r *structuredProgressReport) NewProgress(msg string, args ...any) progressReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	var report progressReport
	if len(args) == 0 {
		report = newStructuredProgressReport(r.logger, append(r.address, r.children), msg)
	} else {
		report = newStructuredProgressReport(r.logger, append(r.address, r.children), fmt.Sprintf(msg, args...))
	}
	r.children++
	return report
}

func (r *structuredProgressReport) NewStatus() statusReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.children++
	return newStructuredStatusReport(r.logger, append(r.address, r.children))
}

func (r *structuredProgressReport) Start() {
	_ = r.logger.Log(Record{
		NewProgress: &NewProgressRecord{
			Address: r.address,
			Message: r.message,
		},
	})
}

func (r *structuredProgressReport) Done() {
	_ = r.logger.Log(Record{
		ProgressDone: &ProgressDoneRecord{
			Address: r.address,
		},
	})
}

func (r *structuredProgressReport) Error(err error) {
	_ = r.logger.Log(Record{
		ProgressError: &ProgressErrorRecord{
			Address: r.address,
			Message: err.Error(),
		},
	})
}

func newStructuredStatusReport(logger *structuredLogger, address []int) *structuredStatusReport {
	_ = logger.Log(Record{
		NewStatus: &NewStatusRecord{
			Address: address,
		},
	})
	return &structuredStatusReport{
		logger:  logger,
		address: address,
	}
}

type structuredStatusReport struct {
	logger  *structuredLogger
	address []int
}

// Update updates the report
func (r *structuredStatusReport) Update(contents string) {
	_ = r.logger.Log(Record{
		StatusUpdate: &StatusUpdateRecord{
			Address: r.address,
			Message: contents,
		},
	})
}

// Done marks the sub-task complete
func (r *structuredStatusReport) Done() {
	_ = r.logger.Log(Record{
		StatusDone: &StatusDoneRecord{
			Address: r.address,
		},
	})
}

func (r *structuredStatusReport) Error(err error) {
	_ = r.logger.Log(Record{
		StatusError: &StatusErrorRecord{
			Address: r.address,
			Message: err.Error(),
		},
	})
}
