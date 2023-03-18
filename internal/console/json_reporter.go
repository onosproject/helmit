package console

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

type jsonLogger struct {
	writer io.Writer
}

func (l *jsonLogger) Log(entry reportEntry) error {
	data, err := json.Marshal(&entry)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(data)
	buf.WriteByte('\n')
	_, err = l.writer.Write(buf.Bytes())
	return err
}

func newJSONReporter(writer io.Writer) Reporter {
	return &jsonProgressReport{
		logger: &jsonLogger{
			writer: writer,
		},
		address: []int{},
	}
}

func newJSONProgressReport(logger *jsonLogger, address []int, message string) ProgressReport {
	return &jsonProgressReport{
		logger:  logger,
		address: address,
		message: message,
	}
}

type jsonProgressReport struct {
	logger   *jsonLogger
	address  []int
	message  string
	children int
	mu       sync.Mutex
}

func (r *jsonProgressReport) NewProgress(msg string, args ...any) ProgressReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	_ = r.logger.Log(reportEntry{
		AppendProgress: &appendProgressEntry{
			Address: r.address,
			Message: msg,
		},
	})
	report := newJSONProgressReport(r.logger, append(r.address, r.children), msg)
	r.children++
	return report
}

func (r *jsonProgressReport) NewStatus() StatusReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	_ = r.logger.Log(reportEntry{
		AppendStatus: &appendStatusEntry{
			Address: r.address,
		},
	})
	report := newJSONStatusReport(r.logger, append(r.address, r.children))
	r.children++
	return report
}

func (r *jsonProgressReport) Start() {
	_ = r.logger.Log(reportEntry{
		ProgressStart: &progressStartEntry{
			Address: r.address,
		},
	})
}

func (r *jsonProgressReport) Finish() {
	_ = r.logger.Log(reportEntry{
		ProgressFinish: &progressFinishEntry{
			Address: r.address,
		},
	})
}

func (r *jsonProgressReport) Error(err error) {
	_ = r.logger.Log(reportEntry{
		ProgressError: &progressErrorEntry{
			Address: r.address,
			Message: err.Error(),
		},
	})
}

func newJSONStatusReport(logger *jsonLogger, address []int) *jsonStatusReport {
	return &jsonStatusReport{
		logger:  logger,
		address: address,
	}
}

type jsonStatusReport struct {
	logger  *jsonLogger
	address []int
}

// Update updates the report
func (r *jsonStatusReport) Update(contents string) {
	_ = r.logger.Log(reportEntry{
		StatusUpdate: &statusUpdateEntry{
			Address: r.address,
			Message: contents,
		},
	})
}

// Done marks the sub-task complete
func (r *jsonStatusReport) Done() {
	_ = r.logger.Log(reportEntry{
		StatusDone: &statusDoneEntry{
			Address: r.address,
		},
	})
}

func (r *jsonStatusReport) Error(err error) {
	_ = r.logger.Log(reportEntry{
		StatusError: &statusErrorEntry{
			Address: r.address,
			Message: err.Error(),
		},
	})
}
