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
	NewProgress    *newProgressEntry    `json:"newProgress"`
	ProgressStart  *progressStartEntry  `json:"progressStart"`
	ProgressFinish *progressFinishEntry `json:"progressFinish"`
	ProgressError  *progressErrorEntry  `json:"progressError"`
	NewStatus      *newStatusEntry      `json:"newStatus"`
	StatusUpdate   *statusUpdateEntry   `json:"statusUpdate"`
	StatusDone     *statusDoneEntry     `json:"statusDone"`
	StatusError    *statusErrorEntry    `json:"statusError"`
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
