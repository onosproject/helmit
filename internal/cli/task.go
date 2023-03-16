package cli

import (
	"sync"
)

func NewTaskManager(status *Status) *TaskManager {
	return &TaskManager{
		status: status,
	}
}

type TaskManager struct {
	status *Status
}

func (r *TaskManager) New(msg string, args ...any) *Task {
	task := r.status.NewTask(msg, args...)
	task.Start()
	return newTask(task)
}

func newTask(status *TaskStatus) *Task {
	return &Task{
		status: status,
	}
}

type Task struct {
	status *TaskStatus
	tasks  []*Task
	mu     sync.RWMutex
	wg     sync.WaitGroup
}

func (t *Task) Fork(msg string, args ...any) *Task {
	t.wg.Add(1)
	status := t.status.NewSubTask(msg, args...)
	status.closer = func(status *TaskStatus) {
		t.wg.Done()
	}
	status.Start()
	task := newTask(status)
	t.mu.Lock()
	t.tasks = append(t.tasks, task)
	t.mu.Unlock()
	return task
}

func (t *Task) Run(f func(log Logger) error) {
	t.wg.Add(1)
	thread := t.status.NewThread()
	go func() {
		if err := f(thread); err != nil {
			t.status.Error(err)
		}
		thread.Done()
		t.wg.Done()
	}()
}

func (t *Task) Wait() {
	t.mu.RLock()
	for _, task := range t.tasks {
		task.Wait()
	}
	t.mu.RUnlock()

	t.wg.Wait()

	t.status.mu.Lock()
	defer t.status.mu.Unlock()
	if t.status.result != taskError {
		for _, child := range t.status.children {
			if status, ok := child.(*TaskStatus); ok {
				var done bool
				status.mu.RLock()
				if status.result == taskError {
					t.status.close(status.result, status.err)
					done = true
				}
				status.mu.RUnlock()
				if done {
					return
				}
			}
		}
	}
	t.status.close(taskDone, nil)
}

type Logger interface {
	Log(msg string)
	Logf(msg string, args ...any)
}
