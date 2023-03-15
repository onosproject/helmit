package task

import (
	"github.com/gosuri/uilive"
	"io"
	"time"
)

const defaultRefreshRate = time.Millisecond

type Option func(*Options)

func WithRefreshRate(d time.Duration) Option {
	return func(options *Options) {
		options.RefreshRate = d
	}
}

type Options struct {
	RefreshRate time.Duration
}

func NewManager(writer io.Writer, opts ...Option) *Manager {
	options := Options{
		RefreshRate: defaultRefreshRate,
	}
	for _, opt := range opts {
		opt(&options)
	}

	lwriter := uilive.New()
	lwriter.Out = writer
	lwriter.RefreshInterval = time.Hour

	return &Manager{
		Options: options,
		writer:  lwriter,
		root:    newRoot(),
	}
}

type Manager struct {
	Options
	writer  *uilive.Writer
	root    Root
	ticker  *time.Ticker
	stop    chan struct{}
	stopped chan struct{}
}

func (m *Manager) Task(desc string) *Task {
	return newTask(m.root, desc)
}

func (m *Manager) Start() {
	m.ticker = time.NewTicker(m.RefreshRate)
	m.stop = make(chan struct{})
	m.stopped = make(chan struct{})
	go m.run()
}

func (m *Manager) run() {
	for {
		select {
		case <-m.ticker.C:
			m.root.write(m.writer)
			m.writer.Flush()
		case <-m.stop:
			m.root.write(m.writer)
			m.writer.Flush()
			close(m.stopped)
			return
		}
	}
}

func (m *Manager) Stop() {
	m.ticker.Stop()
	close(m.stop)
	<-m.stopped
}
