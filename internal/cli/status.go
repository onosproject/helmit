package cli

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

type Options struct {
	RefreshRate time.Duration
}

func NewStatus(writer io.Writer, opts ...Option) *Status {
	options := Options{
		RefreshRate: defaultRefreshRate,
	}
	for _, opt := range opts {
		opt(&options)
	}

	lwriter := uilive.New()
	lwriter.Out = writer
	lwriter.RefreshInterval = time.Hour

	return &Status{
		Options: options,
		writer:  lwriter,
	}
}

type Status struct {
	Options
	writer  *uilive.Writer
	tasks   []*TaskStatus
	ticker  *time.Ticker
	stop    chan struct{}
	stopped chan struct{}
	mu      sync.RWMutex
}

func (s *Status) NewTask(msg string, args ...any) *TaskStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	task := newTaskStatus(msg, args...)
	s.tasks = append(s.tasks, task)
	return task
}

func (s *Status) Start() {
	s.ticker = time.NewTicker(s.RefreshRate)
	s.stop = make(chan struct{})
	s.stopped = make(chan struct{})
	go s.run()
}

func (s *Status) run() {
	for {
		select {
		case <-s.ticker.C:
			s.write()
		case <-s.stop:
			s.write()
			close(s.stopped)
			return
		}
	}
}

func (s *Status) write() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, task := range s.tasks {
		task.write(s.writer, 0)
	}
	s.writer.Flush()
}

func (s *Status) Stop() {
	s.ticker.Stop()
	close(s.stop)
	<-s.stopped
}

func newTaskStatus(msg string, args ...any) *TaskStatus {
	var desc string
	if len(args) > 0 {
		desc = fmt.Sprintf(msg, args...)
	} else {
		desc = msg
	}
	return &TaskStatus{
		desc: desc,
	}
}

type taskResult int

const (
	taskDone taskResult = iota
	taskError
)

type statusWriter interface {
	write(writer *uilive.Writer, depth int)
}

type TaskStatus struct {
	desc     string
	children []statusWriter
	start    time.Time
	done     bool
	result   taskResult
	err      error
	closer   func(*TaskStatus)
	mu       sync.RWMutex
}

func (t *TaskStatus) NewSubTask(msg string, args ...any) *TaskStatus {
	t.mu.Lock()
	defer t.mu.Unlock()
	task := newTaskStatus(msg, args...)
	t.children = append(t.children, task)
	return task
}

func (t *TaskStatus) NewThread() *ThreadStatus {
	t.mu.Lock()
	defer t.mu.Unlock()
	thread := newThreadStatus()
	t.children = append(t.children, thread)
	return thread
}

func (t *TaskStatus) Start() {
	t.start = time.Now()
}

func (t *TaskStatus) Done() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.close(taskDone, nil)
}

func (t *TaskStatus) Error(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.close(taskError, err)
}

func (t *TaskStatus) close(result taskResult, err error) {
	if !t.done {
		t.done = true
		t.result = result
		t.err = err
		if t.closer != nil {
			t.closer(t)
		}
	}
}

func (t *TaskStatus) write(writer *uilive.Writer, depth int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.done {
		switch t.result {
		case taskDone:
			fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*2), doneMsgColor.Sprintf(" ✔ %s", t.desc))
		case taskError:
			fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*2), errorMsgColor.Sprintf(" ✘ %s", t.desc))
			fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*2+2), errorErrColor.Sprint(t.err.Error()))
		default:
			panic("unexpected task state")
		}
	} else {
		frameIndex := int(time.Since(t.start)/spinnerSpeed) % len(spinnerFrames)
		spinnerFrame := spinnerFrames[frameIndex]
		fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*2), taskMsgColor.Sprintf("%s %s", spinnerFrame, t.desc))
	}

	for _, child := range t.children {
		child.write(writer, depth+1)
	}
}

func newThreadStatus() *ThreadStatus {
	return &ThreadStatus{}
}

// ThreadStatus is a sub TaskStatus logger
type ThreadStatus struct {
	value atomic.Pointer[string]
}

// Log logs a message to the TaskStatus
func (t *ThreadStatus) Log(msg string) {
	t.value.Store(&msg)
}

// Logf logs a formatted message to the TaskStatus
func (t *ThreadStatus) Logf(msg string, args ...any) {
	t.Log(fmt.Sprintf(msg, args...))
}

// Done marks the sub-task complete
func (t *ThreadStatus) Done() {
	t.value = atomic.Pointer[string]{}
}

func (t *ThreadStatus) write(writer *uilive.Writer, depth int) {
	value := t.value.Load()
	if value == nil {
		return
	}
	fmt.Fprintf(writer.Newline(), "%s %s\n", strings.Repeat(" ", depth*2), *value)
}
