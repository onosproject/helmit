package console

import (
	"github.com/gosuri/uilive"
)

type reportWriter interface {
	write(writer *uilive.Writer, depth int)
}

type Reporter interface {
	NewProgress(msg string, args ...any) ProgressReport
	NewStatus() StatusReport
}

type ProgressReport interface {
	Reporter
	Start()
	Finish()
	Error(err error)
}

type StatusReport interface {
	Update(message string)
	Done()
	Error(err error)
}

type reportEntry struct {
	AppendProgress *appendProgressEntry `json:"appendProgress,omitempty"`
	ProgressStart  *progressStartEntry  `json:"progressStart,omitempty"`
	ProgressFinish *progressFinishEntry `json:"progressFinish,omitempty"`
	ProgressError  *progressErrorEntry  `json:"progressError,omitempty"`
	AppendStatus   *appendStatusEntry   `json:"appendStatus,omitempty"`
	StatusUpdate   *statusUpdateEntry   `json:"statusUpdate,omitempty"`
	StatusDone     *statusDoneEntry     `json:"statusDone,omitempty"`
	StatusError    *statusErrorEntry    `json:"statusError,omitempty"`
}

type appendProgressEntry struct {
	Address []int  `json:"address"`
	Message string `json:"message"`
}

type progressStartEntry struct {
	Address []int `json:"address"`
}

type progressFinishEntry struct {
	Address []int `json:"address"`
}

type progressErrorEntry struct {
	Address []int  `json:"address"`
	Message string `json:"message"`
}

type appendStatusEntry struct {
	Address []int `json:"address"`
}

type statusUpdateEntry struct {
	Address []int  `json:"address"`
	Message string `json:"message"`
}

type statusDoneEntry struct {
	Address []int `json:"address"`
}

type statusErrorEntry struct {
	Address []int  `json:"address"`
	Message string `json:"message"`
}
