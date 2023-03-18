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

var progressFrames = []string{"⠈⠁", "⠈⠑", "⠈⠱", "⠈⡱", "⢀⡱", "⢄⡱", "⢄⡱", "⢆⡱", "⢎⡱", "⢎⡰", "⢎⡠", "⢎⡀", "⢎⠁", "⠎⠁", "⠊⠁"}
var statusFrames = []string{"⊷", "⊶"}

const spinnerSpeed = 150 * time.Millisecond
const errorHighlightDuration = 5 * time.Second
const minStatusDelay = 250 * time.Millisecond

var (
	pendingMsgColor      = color.New(color.FgWhite, color.Faint, color.Concealed)
	runningMsgColor      = color.New(color.FgBlue)
	succeededMsgColor    = color.New(color.FgGreen)
	failedMsgColor       = color.New(color.FgRed)
	failedHighlightColor = color.New(color.FgRed, color.BlinkRapid)
	failedErrColor       = color.New(color.FgRed, color.Bold)
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
	if entry.AppendProgress != nil {
		if len(entry.AppendProgress.Address) == 0 {
			r.NewProgress(entry.AppendProgress.Message)
		} else {
			r.mu.RLock()
			defer r.mu.RUnlock()
			progress := r.reports[entry.AppendProgress.Address[0]].(*liveProgressReport)
			return progress.restore(reportEntry{
				AppendProgress: &appendProgressEntry{
					Address: entry.AppendProgress.Address[1:],
					Message: entry.AppendProgress.Message,
				},
			})
		}
	} else if entry.ProgressStart != nil {
		r.mu.RLock()
		defer r.mu.RUnlock()
		progress := r.reports[entry.ProgressStart.Address[0]].(*liveProgressReport)
		return progress.restore(reportEntry{
			ProgressStart: &progressStartEntry{
				Address: entry.ProgressStart.Address[1:],
			},
		})
	} else if entry.ProgressFinish != nil {
		r.mu.RLock()
		defer r.mu.RUnlock()
		progress := r.reports[entry.ProgressFinish.Address[0]].(*liveProgressReport)
		return progress.restore(reportEntry{
			ProgressFinish: &progressFinishEntry{
				Address: entry.ProgressFinish.Address[1:],
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
	} else if entry.AppendStatus != nil {
		if len(entry.AppendStatus.Address) == 0 {
			r.NewStatus()
		} else {
			r.mu.RLock()
			defer r.mu.RUnlock()
			progress := r.reports[entry.AppendStatus.Address[0]].(*liveProgressReport)
			return progress.restore(reportEntry{
				AppendStatus: &appendStatusEntry{
					Address: entry.AppendStatus.Address[1:],
				},
			})
		}
	} else if entry.StatusUpdate != nil {
		r.mu.RLock()
		defer r.mu.RUnlock()
		progress := r.reports[entry.StatusUpdate.Address[0]].(*liveProgressReport)
		return progress.restore(reportEntry{
			StatusUpdate: &statusUpdateEntry{
				Address: entry.StatusUpdate.Address[1:],
				Message: entry.StatusUpdate.Message,
			},
		})
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

type progressState int

const (
	progressPending progressState = iota
	progressRunning
	progressSucceeded
	progressFailed
)

func newLiveProgressReport(msg string, args ...any) *liveProgressReport {
	var desc string
	if len(args) > 0 {
		desc = fmt.Sprintf(msg, args...)
	} else {
		desc = msg
	}
	return &liveProgressReport{
		desc:      desc,
		state:     progressPending,
		startTime: time.Now(),
	}
}

type liveProgressReport struct {
	desc       string
	state      progressState
	children   []reportWriter
	startTime  time.Time
	updateTime time.Time
	err        error
	mu         sync.RWMutex
}

func (r *liveProgressReport) failed() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state == progressFailed
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
	r.setState(progressRunning, nil)
}

func (r *liveProgressReport) Finish() {
	r.setState(progressSucceeded, nil)
}

func (r *liveProgressReport) Error(err error) {
	r.setState(progressFailed, err)
}

func (r *liveProgressReport) setState(state progressState, err error) {
	r.mu.Lock()
	r.state = state
	r.err = err
	r.updateTime = time.Now()
	r.mu.Unlock()
}

func (r *liveProgressReport) write(writer *uilive.Writer, depth int) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	switch r.state {
	case progressPending:
		fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*3), pendingMsgColor.Sprintf(" · %s", r.desc))
	case progressRunning:
		frameIndex := int(time.Since(r.updateTime)/spinnerSpeed) % len(progressFrames)
		spinnerFrame := progressFrames[frameIndex]
		fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*3), runningMsgColor.Sprintf("%s %s", spinnerFrame, r.desc))
		for _, child := range r.children {
			child.write(writer, depth+1)
		}
	case progressSucceeded:
		fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*3), succeededMsgColor.Sprintf(" ✔ %s", r.desc))
	case progressFailed:
		var failures []reportWriter
		for _, child := range r.children {
			if report, ok := child.(*liveProgressReport); ok && report.failed() {
				failures = append(failures, child)
			}
		}

		if len(failures) > 0 {
			fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*3), failedMsgColor.Sprintf(" ✘ %s", r.desc))
		} else {
			if time.Since(r.updateTime) <= errorHighlightDuration {
				fmt.Fprintf(writer.Newline(), "%s%s%s\n", strings.Repeat(" ", depth*3), failedMsgColor.Sprintf(" ✘ %s ← ", r.desc), failedHighlightColor.Sprint(r.err.Error()))
			} else {
				fmt.Fprintf(writer.Newline(), "%s%s%s\n", strings.Repeat(" ", depth*3), failedMsgColor.Sprintf(" ✘ %s ← ", r.desc), failedErrColor.Sprint(r.err.Error()))
			}
		}

		for _, child := range failures {
			child.write(writer, depth+1)
		}
	}
}

func (r *liveProgressReport) restore(entry reportEntry) error {
	if entry.AppendProgress != nil {
		if len(entry.AppendProgress.Address) == 0 {
			r.NewProgress(entry.AppendProgress.Message)
		} else {
			r.mu.RLock()
			defer r.mu.RUnlock()
			progress := r.children[entry.AppendProgress.Address[0]].(*liveProgressReport)
			return progress.restore(reportEntry{
				AppendProgress: &appendProgressEntry{
					Address: entry.AppendProgress.Address[1:],
					Message: entry.AppendProgress.Message,
				},
			})
		}
	} else if entry.ProgressStart != nil {
		if len(entry.ProgressStart.Address) == 1 {
			r.children[entry.ProgressStart.Address[0]].(*liveProgressReport).Start()
		} else {
			r.mu.RLock()
			defer r.mu.RUnlock()
			progress := r.children[entry.ProgressStart.Address[0]].(*liveProgressReport)
			return progress.restore(reportEntry{
				ProgressStart: &progressStartEntry{
					Address: entry.ProgressStart.Address[1:],
				},
			})
		}
	} else if entry.ProgressFinish != nil {
		if len(entry.ProgressFinish.Address) == 1 {
			r.children[entry.ProgressFinish.Address[0]].(*liveProgressReport).Finish()
		} else {
			r.mu.RLock()
			defer r.mu.RUnlock()
			progress := r.children[entry.ProgressFinish.Address[0]].(*liveProgressReport)
			return progress.restore(reportEntry{
				ProgressFinish: &progressFinishEntry{
					Address: entry.ProgressFinish.Address[1:],
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
	} else if entry.AppendStatus != nil {
		if len(entry.AppendStatus.Address) == 0 {
			r.NewStatus()
		} else {
			r.mu.RLock()
			defer r.mu.RUnlock()
			progress := r.children[entry.AppendStatus.Address[0]].(*liveProgressReport)
			return progress.restore(reportEntry{
				AppendStatus: &appendStatusEntry{
					Address: entry.AppendStatus.Address[1:],
				},
			})
		}
	} else if entry.StatusUpdate != nil {
		if len(entry.StatusUpdate.Address) == 1 {
			r.children[entry.StatusUpdate.Address[0]].(*liveStatusReport).Update(entry.StatusUpdate.Message)
		} else {
			r.mu.RLock()
			defer r.mu.RUnlock()
			progress := r.children[entry.StatusUpdate.Address[0]].(*liveProgressReport)
			return progress.restore(reportEntry{
				StatusUpdate: &statusUpdateEntry{
					Address: entry.StatusUpdate.Address[1:],
					Message: entry.StatusUpdate.Message,
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

func newLiveStatusReport() *liveStatusReport {
	return &liveStatusReport{
		started: time.Now(),
	}
}

// liveStatusReport is a liveProgressReport status report
type liveStatusReport struct {
	value   atomic.Pointer[string]
	started time.Time
}

// Update updates the report
func (r *liveStatusReport) Update(contents string) {
	r.value.Store(&contents)
}

// Done marks the sub-task complete
func (r *liveStatusReport) Done() {
	r.afterMinDelay(func() {
		r.value = atomic.Pointer[string]{}
	})
}

func (r *liveStatusReport) Error(err error) {
	r.afterMinDelay(func() {
		value := r.value.Load()
		if value != nil {
			r.Update(failedErrColor.Sprint(*value))
		}
	})
}

func (r *liveStatusReport) afterMinDelay(f func()) {
	if time.Since(r.started) < minStatusDelay {
		time.AfterFunc(time.Until(r.started.Add(minStatusDelay)), f)
	} else {
		f()
	}
}

func (r *liveStatusReport) write(writer *uilive.Writer, depth int) {
	value := r.value.Load()
	if value == nil {
		return
	}
	frameIndex := int(time.Since(r.started)/spinnerSpeed) % len(statusFrames)
	fmt.Fprintf(writer.Newline(), "%s %s %s\n", strings.Repeat(" ", depth*3), statusFrames[frameIndex], *value)
}
