package cli

import (
	"bytes"
	"fmt"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var spinnerCharSet = []string{
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
	taskMsgColor      = color.New(color.FgBlue)
	completeMsgColor  = color.New(color.FgGreen)
	cautionMsgColor   = color.New(color.FgHiYellow, color.Bold)
	cautionErrorColor = color.New(color.FgHiYellow, color.Italic)
	failMsgColor      = color.New(color.FgRed, color.Bold)
	failErrorColor    = color.New(color.FgRed, color.Italic)
)

// Task logs progress of a task associated with a logger
type Task struct {
	spinner *spinner.Spinner
	header  string
	lines   map[int]string
	index   int
	done    atomic.Bool
	closer  func(*Task)
	wg      sync.WaitGroup
}

// Start marks the start of the request
func (t *Task) Start() {
	t.spinner.Start()
}

func (t *Task) Sub() *SubTask {
	t.spinner.Lock()
	index := t.index
	t.index++
	t.spinner.Unlock()
	return &SubTask{
		task:  t,
		index: index,
	}
}

func (t *Task) Task(desc string) *Task {
	t.spinner.Lock()
	index := t.index
	t.index++
	t.spinner.Unlock()

	buf := &bytes.Buffer{}
	spin := spinner.New(spinnerCharSet, spinnerSpeed, spinner.WithWriter(buf))
	_ = spin.Color("blue", "bold")
	spin.Prefix = t.spinner.Prefix + "   "
	spin.PostUpdate = func(spin *spinner.Spinner) {
		t.spinner.Lock()
		defer t.spinner.Unlock()
		t.lines[index] = fmt.Sprintf("%s %s", buf.String(), taskMsgColor.Sprint(desc))
		buf.Reset()
		t.update()
	}
	t.wg.Add(1)
	return &Task{
		spinner: spin,
		header:  desc,
		lines:   make(map[int]string),
		closer: func(task *Task) {
			t.lines[index] = task.spinner.FinalMSG
			t.update()
			t.wg.Done()
		},
	}
}

func (t *Task) Complete() {
	if t.done.CompareAndSwap(false, true) {
		t.wg.Wait()
		t.spinner.Lock()
		lines := append([]string{
			completeMsgColor.Sprintf(" ✔ %s", t.header)},
			t.children()...)
		t.spinner.FinalMSG = fmt.Sprintf("%s\n", strings.Join(lines, "\n"))
		t.spinner.Unlock()
		t.spinner.Stop()
		if t.closer != nil {
			t.closer(t)
		}
	}
}

func (t *Task) Caution(err error) {
	if t.done.CompareAndSwap(false, true) {
		t.wg.Wait()
		t.spinner.Lock()
		lines := append([]string{
			cautionMsgColor.Sprintf(" ✘ %s", t.header),
			cautionErrorColor.Sprintf("   %s", err.Error())},
			t.children()...)
		t.spinner.FinalMSG = fmt.Sprintf("%s\n", strings.Join(lines, "\n"))
		t.spinner.Unlock()
		t.spinner.Stop()
		if t.closer != nil {
			t.closer(t)
		}
	}
}

func (t *Task) Fail(err error) {
	if t.done.CompareAndSwap(false, true) {
		t.wg.Wait()
		t.spinner.Lock()
		lines := append([]string{
			failMsgColor.Sprintf(" ✘ %s", t.header),
			failErrorColor.Sprintf("   %s", err.Error())},
			t.children()...)
		t.spinner.FinalMSG = fmt.Sprintf("%s\n", strings.Join(lines, "\n"))
		t.spinner.Unlock()
		t.spinner.Stop()
		if t.closer != nil {
			t.closer(t)
		}
	}
}

func (t *Task) update() {
	if t.done.Load() {
		return
	}
	lines := append([]string{taskMsgColor.Sprintf(" %s", t.header)}, t.children()...)
	t.spinner.Suffix = fmt.Sprintf("%s\n", strings.Join(lines, "\n"))
}

func (t *Task) children() []string {
	ids := make([]int, 0, len(t.lines))
	for sid := range t.lines {
		ids = append(ids, sid)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	lines := make([]string, 0, len(t.lines))
	for _, sid := range ids {
		lines = append(lines, fmt.Sprintf("   %s", strings.Trim(t.lines[sid], "\n")))
	}
	return lines
}

func (t *Task) log(id int, msg string) {
	t.spinner.Lock()
	defer t.spinner.Unlock()
	t.lines[id] = msg
	t.update()
}

func (t *Task) close(id int) {
	t.spinner.Lock()
	defer t.spinner.Unlock()
	delete(t.lines, id)
	t.update()
}

// SubTask is a sub Task logger
type SubTask struct {
	task  *Task
	index int
}

// Log logs a message to the Task
func (s *SubTask) Log(msg string) {
	s.task.log(s.index, msg)
}

// Logf logs a formatted message to the Task
func (s *SubTask) Logf(msg string, args ...any) {
	s.task.log(s.index, fmt.Sprintf(msg, args...))
}

// Close closes the logger
func (s *SubTask) Close() {
	s.task.close(s.index)
}
