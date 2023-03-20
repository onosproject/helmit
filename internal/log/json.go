package log

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

func NewJSONWriter(writer io.Writer) Writer {
	return &jsonWriter{
		Writer: writer,
	}
}

func NewJSONReader(reader io.Reader) Reader {
	return &jsonReader{
		Reader:  reader,
		scanner: bufio.NewScanner(reader),
	}
}

type jsonReader struct {
	io.Reader
	scanner *bufio.Scanner
}

func (w *jsonReader) ReadRecord() (Record, error) {
	if w.scanner.Scan() {
		data := w.scanner.Bytes()
		var kindEntry struct {
			Kind   string          `json:"kind"`
			Record json.RawMessage `json:"record"`
		}
		if err := json.Unmarshal(data, &kindEntry); err != nil {
			return nil, err
		}

		switch Kind(kindEntry.Kind) {
		case taskAddedKind:
			record := TaskAddedRecord{}
			if err := json.Unmarshal(kindEntry.Record, &record); err != nil {
				return nil, err
			}
			return record, nil
		case taskStartedKind:
			record := TaskStartedRecord{}
			if err := json.Unmarshal(kindEntry.Record, &record); err != nil {
				return nil, err
			}
			return record, nil
		case taskStatusKind:
			record := TaskStatusRecord{}
			if err := json.Unmarshal(kindEntry.Record, &record); err != nil {
				return nil, err
			}
			return record, nil
		case taskCompleteKind:
			record := TaskCompleteRecord{}
			if err := json.Unmarshal(kindEntry.Record, &record); err != nil {
				return nil, err
			}
			return record, nil
		case taskFailedKind:
			record := TaskFailedRecord{}
			if err := json.Unmarshal(kindEntry.Record, &record); err != nil {
				return nil, err
			}
			return record, nil
		case taskCanceledKind:
			record := TaskCanceledRecord{}
			if err := json.Unmarshal(kindEntry.Record, &record); err != nil {
				return nil, err
			}
			return record, nil
		case taskOutputKind:
			record := TaskOutputRecord{}
			if err := json.Unmarshal(kindEntry.Record, &record); err != nil {
				return nil, err
			}
			return record, nil
		default:
			return nil, errors.New("unknown entry kind")
		}
	}
	return nil, io.EOF
}

type jsonWriter struct {
	io.Writer
}

func (w *jsonWriter) WriteRecord(record Record) error {
	rawEntry := struct {
		Kind   string `json:"kind"`
		Record Record `json:"record"`
	}{
		Kind:   string(record.kind()),
		Record: record,
	}

	data, err := json.Marshal(rawEntry)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(data)
	buf.WriteByte('\n')
	_, err = w.Write(buf.Bytes())
	return err
}
