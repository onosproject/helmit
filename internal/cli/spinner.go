package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/fatih/color"
	"github.com/gosuri/uilive"
	"io"
	"sync"
	"time"
)

var (
	spinnerFrames = []string{"⠈⠁", "⠈⠑", "⠈⠱", "⠈⡱", "⢀⡱", "⢄⡱", "⢄⡱", "⢆⡱", "⢎⡱", "⢎⡰", "⢎⡠", "⢎⡀", "⢎⠁", "⠎⠁", "⠊⠁"}
	runningColor  = color.New(color.FgBlue)
	successColor  = color.New(color.FgGreen)
	failureColor  = color.New(color.FgRed, color.Bold)
)

const (
	spinnerSpeed = 150 * time.Millisecond
	refreshRate  = 5 * time.Millisecond
	successIcon  = "✔"
	failureIcon  = "✘"
)

type spinnerState int

const (
	spinnerPending spinnerState = iota
	spinnerRunning
	spinnerSucceeded
	spinnerFailed
)

func NewLogger(writer io.Writer) *Logger {
	lwriter := uilive.New()
	lwriter.Out = writer
	return &Logger{
		writer:      lwriter,
		refreshRate: refreshRate,
	}
}

type Logger struct {
	writer      *uilive.Writer
	refreshRate time.Duration
	ticker      *time.Ticker
	stopCh      chan struct{}
	spinners    []*Spinner
	mu          sync.Mutex
}

func (l *Logger) Start() {
	l.mu.Lock()
	l.ticker = time.NewTicker(l.refreshRate)
	l.stopCh = make(chan struct{})
	l.mu.Unlock()
	go l.run()
}

func (l *Logger) run() {
	for {
		select {
		case <-l.ticker.C:
			_ = l.Refresh()
		case <-l.stopCh:
			return
		}
	}
}

func (l *Logger) Refresh() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, spinner := range l.spinners {
		if err := spinner.Flush(); err != nil {
			return err
		}
	}
	return l.writer.Flush()
}

func (l *Logger) Stop() {
	l.ticker.Stop()
	close(l.stopCh)
	l.Refresh()
}

func (l *Logger) NewSpinner(message string, args ...any) *Spinner {
	l.mu.Lock()
	defer l.mu.Unlock()
	spinner := &Spinner{
		writer:  l.writer,
		message: fmt.Sprintf(message, args...),
	}
	l.spinners = append(l.spinners, spinner)
	return spinner
}

type Spinner struct {
	writer  *uilive.Writer
	state   spinnerState
	started time.Time
	message string
	log     bytes.Buffer
	mu      sync.Mutex
}

func (s *Spinner) Log(message string) {
	s.mu.Lock()
	s.log.WriteString(message)
	s.log.WriteByte('\n')
	s.mu.Unlock()
}

func (s *Spinner) Logf(message string, args ...any) {
	s.Log(fmt.Sprintf(message, args...))
}

func (s *Spinner) Start() {
	s.mu.Lock()
	s.state = spinnerRunning
	s.started = time.Now()
	s.mu.Unlock()
}

func (s *Spinner) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch s.state {
	case spinnerRunning:
		frameIndex := int(time.Since(s.started)/spinnerSpeed) % len(spinnerFrames)
		spinnerFrame := spinnerFrames[frameIndex]
		_, err := fmt.Fprintf(s.writer.Newline(), "%s\n", runningColor.Sprintf("%s %s", spinnerFrame, s.message))
		if err != nil {
			return err
		}
	case spinnerSucceeded:
		_, err := fmt.Fprintf(s.writer.Newline(), "%s\n", successColor.Sprintf(" %s %s", successIcon, s.message))
		if err != nil {
			return err
		}
	case spinnerFailed:
		_, err := fmt.Fprintf(s.writer.Newline(), "%s\n", failureColor.Sprintf(" %s %s", failureIcon, s.message))
		if err != nil {
			return err
		}
	}

	if s.log.Len() > 0 {
		scanner := bufio.NewScanner(bytes.NewBuffer(s.log.Bytes()))
		for scanner.Scan() {
			_, err := fmt.Fprintf(s.writer.Newline(), "      %s\n", scanner.Text())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Spinner) Succeed() {
	s.mu.Lock()
	s.state = spinnerSucceeded
	s.mu.Unlock()
}

func (s *Spinner) Fail() {
	s.mu.Lock()
	s.state = spinnerFailed
	s.mu.Unlock()
}
