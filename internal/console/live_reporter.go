package console

import (
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

func (r *liveReporter) restore(entry reportEntry) error {
	if entry.NewProgress != nil {
		if len(entry.NewProgress.Address) == 0 {
			r.NewProgress(entry.NewProgress.Message).Start()
		} else {
			r.mu.RLock()
			defer r.mu.RUnlock()
			progress := r.reports[entry.NewProgress.Address[0]].(*liveProgressReport)
			return progress.restore(reportEntry{
				NewProgress: &newProgressEntry{
					Address: entry.NewProgress.Address[1:],
					Message: entry.NewProgress.Message,
				},
			})
		}
	} else if entry.ProgressDone != nil {
		r.mu.RLock()
		defer r.mu.RUnlock()
		progress := r.reports[entry.ProgressDone.Address[0]].(*liveProgressReport)
		return progress.restore(reportEntry{
			ProgressDone: &progressDoneEntry{
				Address: entry.ProgressDone.Address[1:],
			},
		})
	} else if entry.ProgressError != nil {
		r.mu.RLock()
		defer r.mu.RUnlock()
		progress := r.reports[entry.ProgressError.Address[0]].(*liveProgressReport)
		return progress.restore(reportEntry{
			ProgressError: &progressErrorEntry{
				Address: entry.ProgressError.Address[1:],
				Message: entry.ProgressError.Message,
			},
		})
	} else if entry.NewStatus != nil {
		if len(entry.NewStatus.Address) == 0 {
			r.NewStatus()
		} else {
			r.mu.RLock()
			defer r.mu.RUnlock()
			progress := r.reports[entry.NewStatus.Address[0]].(*liveProgressReport)
			return progress.restore(reportEntry{
				NewStatus: &newStatusEntry{
					Address: entry.NewStatus.Address[1:],
				},
			})
		}
	} else if entry.StatusDone != nil {
		r.mu.RLock()
		defer r.mu.RUnlock()
		progress := r.reports[entry.StatusDone.Address[0]].(*liveProgressReport)
		return progress.restore(reportEntry{
			StatusDone: &statusDoneEntry{
				Address: entry.StatusDone.Address[1:],
			},
		})
	} else if entry.StatusError != nil {
		r.mu.RLock()
		defer r.mu.RUnlock()
		progress := r.reports[entry.StatusError.Address[0]].(*liveProgressReport)
		return progress.restore(reportEntry{
			StatusError: &statusErrorEntry{
				Address: entry.StatusError.Address[1:],
				Message: entry.StatusError.Message,
			},
		})
	}
	return nil
}

func (r *liveReporter) NewStatus() StatusReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	status := newLiveStatusReport()
	r.reports = append(r.reports, status)
	return status
}

func (r *liveReporter) NewProgress(msg string, args ...any) ProgressReport {
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

func (r *liveProgressReport) restore(entry reportEntry) error {
	if entry.NewProgress != nil {
		if len(entry.NewProgress.Address) == 0 {
			r.NewProgress(entry.NewProgress.Message).Start()
		} else {
			r.mu.RLock()
			defer r.mu.RUnlock()
			progress := r.children[entry.NewProgress.Address[0]].(*liveProgressReport)
			return progress.restore(reportEntry{
				NewProgress: &newProgressEntry{
					Address: entry.NewProgress.Address[1:],
					Message: entry.NewProgress.Message,
				},
			})
		}
	} else if entry.ProgressDone != nil {
		if len(entry.ProgressDone.Address) == 1 {
			r.children[entry.ProgressDone.Address[0]].(*liveProgressReport).Done()
		} else {
			r.mu.RLock()
			defer r.mu.RUnlock()
			progress := r.children[entry.ProgressDone.Address[0]].(*liveProgressReport)
			return progress.restore(reportEntry{
				ProgressDone: &progressDoneEntry{
					Address: entry.ProgressDone.Address[1:],
				},
			})
		}
	} else if entry.ProgressError != nil {
		if len(entry.ProgressError.Address) == 1 {
			r.children[entry.ProgressError.Address[0]].(*liveProgressReport).Error(errors.New(entry.ProgressError.Message))
		} else {
			r.mu.RLock()
			defer r.mu.RUnlock()
			progress := r.children[entry.ProgressError.Address[0]].(*liveProgressReport)
			return progress.restore(reportEntry{
				ProgressError: &progressErrorEntry{
					Address: entry.ProgressError.Address[1:],
					Message: entry.ProgressError.Message,
				},
			})
		}
	} else if entry.NewStatus != nil {
		if len(entry.NewStatus.Address) == 0 {
			r.NewStatus()
		} else {
			r.mu.RLock()
			defer r.mu.RUnlock()
			progress := r.children[entry.NewStatus.Address[0]].(*liveProgressReport)
			return progress.restore(reportEntry{
				NewStatus: &newStatusEntry{
					Address: entry.NewStatus.Address[1:],
				},
			})
		}
	} else if entry.StatusDone != nil {
		if len(entry.StatusDone.Address) == 1 {
			r.children[entry.StatusDone.Address[0]].(*liveStatusReport).Done()
		} else {
			r.mu.RLock()
			defer r.mu.RUnlock()
			progress := r.children[entry.StatusDone.Address[0]].(*liveProgressReport)
			return progress.restore(reportEntry{
				StatusDone: &statusDoneEntry{
					Address: entry.StatusDone.Address[1:],
				},
			})
		}
	} else if entry.StatusError != nil {
		if len(entry.StatusError.Address) == 1 {
			r.children[entry.StatusError.Address[0]].(*liveStatusReport).Error(errors.New(entry.StatusError.Message))
		} else {
			r.mu.RLock()
			defer r.mu.RUnlock()
			progress := r.children[entry.StatusError.Address[0]].(*liveProgressReport)
			return progress.restore(reportEntry{
				StatusError: &statusErrorEntry{
					Address: entry.StatusError.Address[1:],
					Message: entry.StatusError.Message,
				},
			})
		}
	}
	return nil
}

func (r *liveProgressReport) NewProgress(msg string, args ...any) ProgressReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	progress := newLiveProgressReport(msg, args...)
	r.children = append(r.children, progress)
	return progress
}

func (r *liveProgressReport) NewStatus() StatusReport {
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
			fmt.Fprintf(writer.Newline(), "%s%s %s\n", strings.Repeat(" ", depth*3), errorMsgColor.Sprintf(" ✘ %s", r.desc), errorErrColor.Sprintf("← %s", r.err.Error()))
			for _, child := range r.children {
				child.write(writer, depth+1)
			}
		} else {
			fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*3), doneMsgColor.Sprintf(" ✔ %s", r.desc))
		}
	} else {
		frameIndex := int(time.Since(r.start)/spinnerSpeed) % len(spinnerFrames)
		spinnerFrame := spinnerFrames[frameIndex]
		fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*3), taskMsgColor.Sprintf("%s %s", spinnerFrame, r.desc))
		for _, child := range r.children {
			child.write(writer, depth+1)
		}
	}
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
			fmt.Fprintf(writer.Newline(), "%s %s\n", strings.Repeat(" ", depth*3), line)
		}
	}
}
