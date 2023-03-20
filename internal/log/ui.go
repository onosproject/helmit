package log

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/fatih/color"
	"github.com/gosuri/uilive"
	"golang.org/x/term"
	"io"
	"strings"
	"sync"
	"time"
)

var progressFrames = []string{"⠈⠁", "⠈⠑", "⠈⠱", "⠈⡱", "⢀⡱", "⢄⡱", "⢄⡱", "⢆⡱", "⢎⡱", "⢎⡰", "⢎⡠", "⢎⡀", "⢎⠁", "⠎⠁", "⠊⠁"}

const (
	spinnerSpeed = 150 * time.Millisecond
	refreshRate  = 5 * time.Millisecond
)

var (
	pendingMsgColor   = color.New(color.FgWhite, color.Faint, color.Concealed)
	runningMsgColor   = color.New(color.FgBlue)
	succeededMsgColor = color.New(color.FgGreen)
	failedMsgColor    = color.New(color.FgRed)
	failedErrColor    = color.New(color.FgRed, color.Bold)
	cancelledMsgColor = color.New(color.FgWhite, color.Faint, color.Concealed)
)

func NewUIWriter(writer io.Writer) *UIWriter {
	lwriter := uilive.New()
	lwriter.Out = writer
	uiwriter := &UIWriter{
		writer:      lwriter,
		uiTask:      newTask(""),
		refreshRate: refreshRate,
	}
	uiwriter.open()
	return uiwriter
}

type UIWriter struct {
	io.Writer
	*uiTask
	writer      *uilive.Writer
	refreshRate time.Duration
	ticker      *time.Ticker
	stopCh      chan struct{}
	mu          sync.Mutex
}

func (w *UIWriter) open() {
	w.ticker = time.NewTicker(w.refreshRate)
	w.stopCh = make(chan struct{})
	go w.run()
}

func (w *UIWriter) run() {
	for {
		select {
		case <-w.ticker.C:
			_ = w.Refresh()
		case <-w.stopCh:
			return
		}
	}
}

func (w *UIWriter) Refresh() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	var lines int
	for _, child := range w.children {
		lines += child.lines()
	}

	for _, child := range w.children {
		err := child.write(w.writer, lines, 0)
		if err != nil {
			return err
		}
	}
	return w.writer.Flush()
}

func (w *UIWriter) WriteRecord(record Record) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	switch r := record.(type) {
	case TaskAddedRecord:
		task, err := w.getTask(r.Task)
		if err != nil {
			return err
		}
		task.AddTask(r.Name)
	case TaskStartedRecord:
		task, err := w.getTask(r.Task)
		if err != nil {
			return err
		}
		task.Start()
	case TaskStatusRecord:
		task, err := w.getTask(r.Task)
		if err != nil {
			return err
		}
		task.SetStatus(r.Status)
	case TaskCompleteRecord:
		task, err := w.getTask(r.Task)
		if err != nil {
			return err
		}
		task.Complete()
	case TaskFailedRecord:
		task, err := w.getTask(r.Task)
		if err != nil {
			return err
		}
		task.Fail(r.Error)
	case TaskCanceledRecord:
		task, err := w.getTask(r.Task)
		if err != nil {
			return err
		}
		task.Cancel()
	case TaskOutputRecord:
		task, err := w.getTask(r.Task)
		if err != nil {
			return err
		}
		task.Output(r.Output)
	default:
		return fmt.Errorf("unknown record kind: %s", record.kind())
	}
	return nil
}

func (w *UIWriter) getTask(path []string) (*uiTask, error) {
	task := w.uiTask
	for _, name := range path {
		index, ok := task.tasks[name]
		if !ok {
			return nil, fmt.Errorf("task %s not found", name)
		}
		if len(task.children) <= index {
			return nil, fmt.Errorf("task %s not found", name)
		}
		task = task.children[index]
	}
	return task, nil
}

func (w *UIWriter) Close() error {
	w.ticker.Stop()
	close(w.stopCh)
	return w.Refresh()
}

type taskState int

const (
	taskPending taskState = iota
	taskRunning
	taskComplete
	taskFailed
	taskCanceled
)

func newTask(name string) *uiTask {
	return &uiTask{
		name:    name,
		tasks:   make(map[string]int),
		updated: time.Now(),
	}
}

type uiTask struct {
	name     string
	status   string
	log      bytes.Buffer
	tasks    map[string]int
	children []*uiTask
	state    taskState
	updated  time.Time
	error    string
}

func (t *uiTask) AddTask(name string) *uiTask {
	task := newTask(name)
	t.tasks[name] = len(t.children)
	t.children = append(t.children, task)
	return task
}

func (t *uiTask) Start() {
	t.state = taskRunning
	t.updated = time.Now()
}

func (t *uiTask) SetStatus(message string) {
	t.status = message
}

func (t *uiTask) Complete() {
	t.state = taskComplete
	t.updated = time.Now()
}

func (t *uiTask) Fail(message string) {
	t.state = taskFailed
	t.error = message
	t.updated = time.Now()
}

func (t *uiTask) Cancel() {
	t.state = taskCanceled
	t.updated = time.Now()
}

func (t *uiTask) Output(output []byte) {
	t.log.Write(output)
}

func (t *uiTask) lines() int {
	switch t.state {
	case taskRunning:
		var lines int
		for _, child := range t.children {
			lines += child.lines()
		}
		return lines + 1
	case taskFailed:
		var count int
		for _, child := range t.children {
			if child.state == taskFailed {
				count++
			}
		}
		return count + 1
	default:
		return 1
	}
}

func (t *uiTask) write(writer *uilive.Writer, height int, depth int) error {
	switch t.state {
	case taskPending:
		_, err := fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*3), pendingMsgColor.Sprintf(" ▹ %s", t.name))
		return err
	case taskRunning:
		frameIndex := int(time.Since(t.updated)/spinnerSpeed) % len(progressFrames)
		spinnerFrame := progressFrames[frameIndex]
		_, err := fmt.Fprintf(writer.Newline(), "%s%s %s\n", strings.Repeat(" ", depth*3), runningMsgColor.Sprintf("%s %s", spinnerFrame, t.name), t.status)
		if err != nil {
			return err
		}

		if t.log.Len() > 0 {
			termWidth, termHeight, err := term.GetSize(0)
			if err != nil {
				return err
			}

			var lines []string
			scanner := bufio.NewScanner(bytes.NewBuffer(t.log.Bytes()))
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}

			numLines := termHeight - height - 1
			if len(lines) > numLines {
				lines = lines[len(lines)-numLines:]
			}

			prefix := strings.Repeat(" ", (depth+2)*3)
			for _, line := range lines {
				if len(line)+len(prefix)+1 > termWidth {
					line = line[:termWidth-len(prefix)-1]
				}
				_, err := fmt.Fprintf(writer.Newline(), "%s%s\n", prefix, line)
				if err != nil {
					return err
				}
			}
		}

		for _, child := range t.children {
			if err := child.write(writer, height, depth+1); err != nil {
				return err
			}
		}
		return nil
	case taskComplete:
		_, err := fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*3), succeededMsgColor.Sprintf(" ✔ %s", t.name))
		return err
	case taskCanceled:
		_, err := fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*3), cancelledMsgColor.Sprintf(" ▹ %s", t.name))
		return err
	case taskFailed:
		var failures []*uiTask
		for _, child := range t.children {
			if child.state == taskFailed {
				failures = append(failures, child)
			}
		}

		if len(failures) > 0 {
			_, err := fmt.Fprintf(writer.Newline(), "%s%s\n", strings.Repeat(" ", depth*3), failedMsgColor.Sprintf(" ✘ %s", t.name))
			if err != nil {
				return err
			}
		} else {
			_, err := fmt.Fprintf(writer.Newline(), "%s%s%s\n", strings.Repeat(" ", depth*3), failedMsgColor.Sprintf(" ✘ %s ← ", t.name), failedErrColor.Sprint(t.error))
			if err != nil {
				return err
			}
		}

		for _, child := range failures {
			if err := child.write(writer, height, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	return nil
}
