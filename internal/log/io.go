package log

import (
	"fmt"
	"io"
	"strings"
)

type Logger interface {
	Log(message string)
	Logf(format string, args ...any)
}

type Writer interface {
	io.Writer
	WriteRecord(record Record) error
}

type Reader interface {
	io.Reader
	ReadRecord() (Record, error)
}

func Copy(writer Writer, reader Reader) error {
	for {
		entry, err := reader.ReadRecord()
		if err != nil {
			if err == io.EOF {
				return nil
			}
		} else {
			if err := writer.WriteRecord(entry); err != nil {
				return err
			}
		}
	}
}

type Kind string

type Record interface {
	fmt.Stringer
	kind() Kind
}

const (
	taskAddedKind    Kind = "TaskAdded"
	taskStartedKind  Kind = "TaskStarted"
	taskStatusKind   Kind = "TaskStatus"
	taskCompleteKind Kind = "TaskComplete"
	taskCanceledKind Kind = "TaskCanceled"
	taskFailedKind   Kind = "TaskFailed"
	taskOutputKind   Kind = "TaskOutput"
)

type TaskRecord struct {
	Task []string
}

func TaskAdded(task []string, name string) Record {
	return TaskAddedRecord{
		TaskRecord: TaskRecord{
			Task: task,
		},
		Name: name,
	}
}

type TaskAddedRecord struct {
	TaskRecord `json:",inline"`
	Name       string
}

func (r TaskAddedRecord) kind() Kind {
	return taskAddedKind
}

func (r TaskAddedRecord) String() string {
	return fmt.Sprintf("TaskAdded %s/%s", strings.Join(r.Task, " -> "), r.Name)
}

func TaskStarted(task []string) Record {
	return TaskStartedRecord{
		TaskRecord: TaskRecord{
			Task: task,
		},
	}
}

type TaskStartedRecord struct {
	TaskRecord `json:",inline"`
}

func (r TaskStartedRecord) kind() Kind {
	return taskStartedKind
}

func (r TaskStartedRecord) String() string {
	return fmt.Sprintf("TaskStarted %s", strings.Join(r.Task, " -> "))
}

func TaskStatus(task []string, status string) Record {
	return TaskStatusRecord{
		TaskRecord: TaskRecord{
			Task: task,
		},
		Status: status,
	}
}

type TaskStatusRecord struct {
	TaskRecord `json:",inline"`
	Status     string
}

func (r TaskStatusRecord) kind() Kind {
	return taskStatusKind
}

func (r TaskStatusRecord) String() string {
	return fmt.Sprintf("TaskStatus %s: %s", strings.Join(r.Task, " -> "), r.Status)
}

func TaskComplete(task []string) Record {
	return TaskCompleteRecord{
		TaskRecord: TaskRecord{
			Task: task,
		},
	}
}

type TaskCompleteRecord struct {
	TaskRecord `json:",inline"`
}

func (r TaskCompleteRecord) kind() Kind {
	return taskCompleteKind
}

func (r TaskCompleteRecord) String() string {
	return fmt.Sprintf("TaskComplete %s", strings.Join(r.Task, " -> "))
}

func TaskFailed(task []string, message string) Record {
	return TaskFailedRecord{
		TaskRecord: TaskRecord{
			Task: task,
		},
		Error: message,
	}
}

type TaskFailedRecord struct {
	TaskRecord `json:",inline"`
	Error      string
}

func (r TaskFailedRecord) kind() Kind {
	return taskFailedKind
}

func (r TaskFailedRecord) String() string {
	return fmt.Sprintf("TaskFailed %s: %s", strings.Join(r.Task, " -> "), r.Error)
}

func TaskCanceled(task []string) Record {
	return TaskCanceledRecord{
		TaskRecord: TaskRecord{
			Task: task,
		},
	}
}

type TaskCanceledRecord struct {
	TaskRecord `json:",inline"`
}

func (r TaskCanceledRecord) kind() Kind {
	return taskCanceledKind
}

func (r TaskCanceledRecord) String() string {
	return fmt.Sprintf("TaskCanceled %s", strings.Join(r.Task, " -> "))
}

func TaskOutput(task []string, output []byte) Record {
	return TaskOutputRecord{
		TaskRecord: TaskRecord{
			Task: task,
		},
		Output: output,
	}
}

type TaskOutputRecord struct {
	TaskRecord `json:",inline"`
	Output     []byte
}

func (r TaskOutputRecord) kind() Kind {
	return taskOutputKind
}

func (r TaskOutputRecord) String() string {
	return fmt.Sprintf("TaskOutput %s: %s", strings.Join(r.Task, " -> "), r.Output)
}
