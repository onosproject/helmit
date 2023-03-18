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
	NewProgress    *newProgressEntry    `json:"newProgress,omitempty"`
	ProgressStart  *progressStartEntry  `json:"progressStart,omitempty"`
	ProgressFinish *progressFinishEntry `json:"progressFinish,omitempty"`
	ProgressError  *progressErrorEntry  `json:"progressError,omitempty"`
	NewStatus      *newStatusEntry      `json:"newStatus,omitempty"`
	StatusUpdate   *statusUpdateEntry   `json:"statusUpdate,omitempty"`
	StatusDone     *statusDoneEntry     `json:"statusDone,omitempty"`
	StatusError    *statusErrorEntry    `json:"statusError,omitempty"`
}

type newProgressEntry struct {
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

type newStatusEntry struct {
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
