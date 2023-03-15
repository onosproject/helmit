package cli

import (
	"bytes"
	"fmt"
	"golang.org/x/term"
	"io"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"unicode/utf8"
)

var isWindows = runtime.GOOS == "windows"
var isWindowsTerminalOnWindows = len(os.Getenv("WT_SESSION")) > 0 && isWindows

// NewLogger creates a new CLI logger
func NewLogger(writer io.Writer) *Logger {
	return &Logger{
		writer: writer,
	}
}

// Logger provides logging with spinners for the CLI
type Logger struct {
	writer     io.Writer
	mu         sync.Mutex
	lastOutput string
}

// Task creates a new CLI task
func (l *Logger) Task(desc string) *Task {
	return newTask(nil, desc, l)
}

// Log logs a message
func (l *Logger) Log(message string) {
	l.print(message)
}

// Logf logs a formatted message
func (l *Logger) Logf(format string, args ...interface{}) {
	l.printf(format, args...)
}

func (l *Logger) output(node Node) {
	if !isWindowsTerminalOnWindows {
		l.erase()
	}
	output := node.Root().String()
	l.lastOutput = output
	l.print(output)
}

// print writes a simple string to the log writer
func (l *Logger) print(message string) {
	buf := bytes.NewBufferString(message)
	l.writeBuffer(buf)
}

// printf is roughly fmt.Fprintf against the log writer
func (l *Logger) printf(format string, args ...interface{}) {
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, format, args...)
	l.writeBuffer(buf)
}

// synchronized write to the inner writer
func (l *Logger) write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.writer.Write(p)
}

// writeBuffer writes buf with write, ensuring there is a trailing newline
func (l *Logger) writeBuffer(buf *bytes.Buffer) {
	// ensure trailing newline
	if buf.Len() == 0 || buf.Bytes()[buf.Len()-1] != '\n' {
		//buf.WriteByte('\n')
	}
	// TODO: should we handle this somehow??
	// Who logs for the logger? 🤔
	_, _ = l.write(buf.Bytes())
}

func (l *Logger) erase() {
	i := utf8.RuneCountInString(l.lastOutput)
	if runtime.GOOS == "windows" && !isWindowsTerminalOnWindows {
		clearString := "\r" + strings.Repeat(" ", i) + "\r"
		l.print(clearString)
		l.lastOutput = ""
		return
	}

	numberOfLinesToErase := computeNumberOfLinesNeededToPrintString(l.lastOutput)

	// Taken from https://en.wikipedia.org/wiki/ANSI_escape_code:
	// \r     - Carriage return - Moves the cursor to column zero
	// \033[K - Erases part of the line. If n is 0 (or missing), clear from
	// cursor to the end of the line. If n is 1, clear from cursor to beginning
	// of the line. If n is 2, clear entire line. Cursor position does not
	// change.
	// \033[F - Go to the beginning of previous line
	eraseCodeString := strings.Builder{}
	// current position is at the end of the last printed line. Start by erasing current line
	eraseCodeString.WriteString("\r\033[K") // start by erasing current line
	for i := 1; i < numberOfLinesToErase; i++ {
		// For each additional lines, go up one line and erase it.
		eraseCodeString.WriteString("\033[F\033[K")
	}
	output := eraseCodeString.String()
	l.print(output)
	l.lastOutput = ""
}

func computeNumberOfLinesNeededToPrintString(linePrinted string) int {
	terminalWidth := math.MaxInt // assume infinity by default to keep behaviour consistent with what we had before
	if term.IsTerminal(0) {
		if width, _, err := term.GetSize(0); err == nil && width > 0 {
			terminalWidth = width
		}
	}
	return computeNumberOfLinesNeededToPrintStringInternal(linePrinted, terminalWidth)
}

// isAnsiMarker returns if a rune denotes the start of an ANSI sequence
func isAnsiMarker(r rune) bool {
	return r == '\x1b'
}

// isAnsiTerminator returns if a rune denotes the end of an ANSI sequence
func isAnsiTerminator(r rune) bool {
	return (r >= 0x40 && r <= 0x5a) || (r == 0x5e) || (r >= 0x60 && r <= 0x7e)
}

// computeLineWidth returns the displayed width of a line
func computeLineWidth(line string) int {
	width := 0
	ansi := false

	for _, r := range []rune(line) {
		// increase width only when outside of ANSI escape sequences
		if ansi || isAnsiMarker(r) {
			ansi = !isAnsiTerminator(r)
		} else {
			width += utf8.RuneLen(r)
		}
	}

	return width
}

func computeNumberOfLinesNeededToPrintStringInternal(linePrinted string, maxLineWidth int) int {
	lineCount := 0
	for _, line := range strings.Split(linePrinted, "\n") {
		lineCount += 1

		lineWidth := computeLineWidth(line)
		if lineWidth > maxLineWidth {
			lineCount += int(float64(lineWidth) / float64(maxLineWidth))
		}
	}
	return lineCount
}
