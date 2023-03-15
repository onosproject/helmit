package cli

import (
	"bytes"
	"fmt"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

var spinnerCharSet = []string{
	"⠈⠁ ",
	"⠈⠑ ",
	"⠈⠱ ",
	"⠈⡱ ",
	"⢀⡱ ",
	"⢄⡱ ",
	"⢄⡱ ",
	"⢆⡱ ",
	"⢎⡱ ",
	"⢎⡰ ",
	"⢎⡠ ",
	"⢎⡀ ",
	"⢎⠁ ",
	"⠎⠁ ",
	"⠊⠁ ",
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

type Node interface {
	fmt.Stringer
	Root() Node
	Parent() (Node, bool)
	string(level int) string
}

type Branch interface {
	Node
	AddChild(node Node) int
	RemoveChild(index int)
	Children() []Node
}

type Leaf interface {
	Node
}

func newNode(parent Branch, child Node, log *Logger) *node {
	var index int
	if parent != nil {
		index = parent.AddChild(child)
	}
	return &node{
		Node:   child,
		log:    log,
		parent: parent,
		index:  index,
	}
}

type node struct {
	Node
	log        *Logger
	parent     Branch
	index      int
	lastOutput string
}

func (n *node) Root() Node {
	var root Node = n
	parent, ok := root.Parent()
	for ok {
		root = parent
		parent, ok = root.Parent()
	}
	return root
}

func (n *node) Parent() (Node, bool) {
	if n.parent == nil {
		return nil, false
	}
	return n.parent, true
}

func (n *node) delete() {
	if n.parent != nil {
		n.parent.RemoveChild(n.index)
	}
}

func (n *node) String() string {
	return n.string(0)
}

func newBranch(parent Branch, node Node, log *Logger) *branch {
	return &branch{
		node:     newNode(parent, node, log),
		children: make(map[int]Node),
	}
}

type branch struct {
	*node
	children map[int]Node
	index    int
}

func (b *branch) AddChild(child Node) int {
	index := b.index
	b.children[index] = child
	b.index++
	return index
}

func (b *branch) RemoveChild(index int) {
	delete(b.children, index)
}

func (b *branch) Children() []Node {
	ids := make([]int, 0, len(b.children))
	for sid := range b.children {
		ids = append(ids, sid)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	children := make([]Node, 0, len(b.children))
	for _, sid := range ids {
		children = append(children, b.children[sid])
	}
	return children
}

func newTask(parent Branch, desc string, log *Logger) *Task {
	task := &Task{
		desc:    desc,
		spinner: spinner.New(spinnerCharSet, spinnerSpeed, spinner.WithSuffix(desc)),
	}
	task.branch = newBranch(parent, task, log)
	return task
}

type Task struct {
	*branch
	desc    string
	buf     *bytes.Buffer
	spinner *spinner.Spinner
	value   atomic.Pointer[string]
	done    atomic.Bool
}

func (t *Task) Task(desc string) *Task {
	return newTask(t, desc, t.log)
}

func (t *Task) SubTask() *SubTask {
	return newSubTask(t, t.log)
}

func (t *Task) print() {
	value := t.buf.String()
	t.buf.Reset()
	t.value.Store(&value)
	t.log.output(t)
}

func (t *Task) Start() {
	t.buf = &bytes.Buffer{}
	t.spinner.Writer = t.buf
	t.spinner.PostUpdate = func(s *spinner.Spinner) {
		t.print()
	}
	t.spinner.Start()
}

func (t *Task) Done() {
	if t.done.CompareAndSwap(false, true) {
		t.spinner.FinalMSG = doneMsgColor.Sprintf(" ✔ %s", t.desc)
		t.spinner.Stop()
		t.print()
	}
}

func (t *Task) Warn(err error) {
	if t.done.CompareAndSwap(false, true) {
		t.spinner.FinalMSG = fmt.Sprint(warnMsgColor.Sprintf(" ✘ %s: ", t.desc), warnErrorColor.Sprint(err.Error()))
		t.spinner.Stop()
		t.print()
	}
}

func (t *Task) Error(err error) {
	if t.done.CompareAndSwap(false, true) {
		t.spinner.FinalMSG = fmt.Sprint(errorMsgColor.Sprintf(" ✘ %s: ", t.desc), errorErrColor.Sprint(err.Error()))
		t.spinner.Stop()
		t.print()
	}
}

func (t *Task) string(level int) string {
	var lines []string
	value := t.value.Load()
	if value != nil {
		lines = append(lines, *value)
	}
	for _, child := range t.Children() {
		lines = append(lines, child.string(level+1))
	}
	return fmt.Sprintf("%*s", level*2, strings.Join(lines, "\n"))
}

func newSubTask(parent Branch, log *Logger) *SubTask {
	task := &SubTask{}
	task.node = newNode(parent, task, log)
	return task
}

// SubTask is a sub Task logger
type SubTask struct {
	*node
	value atomic.Pointer[string]
}

// Log logs a message to the Task
func (t *SubTask) Log(msg string) {
	t.value.Store(&msg)
	t.log.output(t)
}

// Logf logs a formatted message to the Task
func (t *SubTask) Logf(msg string, args ...any) {
	t.Log(fmt.Sprintf(msg, args...))
	t.log.output(t)
}

// Close closes the logger
func (t *SubTask) Close() {
	t.delete()
}

func (t *SubTask) string(level int) string {
	value := t.value.Load()
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%*s", level*2, *value)
}
