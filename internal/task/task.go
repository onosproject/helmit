package task

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/gosuri/uilive"
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
	taskMsgColor   = color.New(color.FgBlue)
	doneMsgColor   = color.New(color.FgGreen)
	warnMsgColor   = color.New(color.FgHiYellow, color.Bold)
	warnErrorColor = color.New(color.FgHiYellow, color.Italic)
	errorMsgColor  = color.New(color.FgRed, color.Bold)
	errorErrColor  = color.New(color.FgRed, color.Italic)
)

func newTask(parent Parent, desc string) *Task {
	return &Task{
		Parent: newParent(),
		parent: parent,
		desc:   desc,
	}
}

type Status int

const (
	StatusOK Status = iota
	StatusWarn
	StatusError
)

type Task struct {
	Parent
	parent Parent
	desc   string
	start  time.Time
	index  int
	done   bool
	status Status
	err    error
	mu     sync.Mutex
}

func (t *Task) Task(desc string) *Task {
	return newTask(t, desc)
}

func (t *Task) SubTask() *SubTask {
	return newSubTask(t)
}

func (t *Task) write(writer *uilive.Writer, depth int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.done {
		switch t.status {
		case StatusOK:
			fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*2), doneMsgColor.Sprintf(" ✔ %s", t.desc))
		case StatusWarn:
			fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*2), warnMsgColor.Sprintf(" ✘ %s", t.desc))
			fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*2+2), warnErrorColor.Sprint(t.err.Error()))
		case StatusError:
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

	for _, child := range t.list() {
		child.write(writer, depth+1)
	}
}

func (t *Task) Start() {
	t.start = time.Now()
	t.index = t.parent.add(t)
}

func (t *Task) Done() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.done {
		return
	}
	t.done = true
	t.status = StatusOK
}

func (t *Task) Warn(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.done {
		return
	}
	t.done = true
	t.status = StatusWarn
	t.err = err
}

func (t *Task) Error(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.done {
		return
	}
	t.done = true
	t.status = StatusError
	t.err = err
}

func newSubTask(parent Parent) *SubTask {
	task := &SubTask{
		parent: parent,
	}
	task.index = parent.add(task)
	return task
}

// SubTask is a sub Task logger
type SubTask struct {
	parent Parent
	index  int
	value  atomic.Pointer[string]
}

// Log logs a message to the Task
func (t *SubTask) Log(msg string) {
	t.value.Store(&msg)
}

// Logf logs a formatted message to the Task
func (t *SubTask) Logf(msg string, args ...any) {
	t.Log(fmt.Sprintf(msg, args...))
}

// Done marks the sub-task complete
func (t *SubTask) Done() {
	t.parent.remove(t.index)
}

func (t *SubTask) write(writer *uilive.Writer, depth int) {
	value := t.value.Load()
	if value == nil {
		return
	}
	fmt.Fprintf(writer.Newline(), "%s %s\n", strings.Repeat(" ", depth*2), *value)
}
