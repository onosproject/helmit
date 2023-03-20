package log

import (
	"bytes"
	"io"
)

func NewTextWriter(writer io.Writer) Writer {
	return &textWriter{
		Writer: writer,
	}
}

type textWriter struct {
	io.Writer
}

func (w *textWriter) WriteRecord(record Record) error {
	buf := bytes.NewBufferString(record.String())
	buf.WriteByte('\n')
	_, err := w.Write(buf.Bytes())
	return err
}
