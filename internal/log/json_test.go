package log

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJSONReaderWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := NewJSONWriter(buf)
	assert.NoError(t, writer.WriteRecord(TaskAdded(nil, "foo")))
	assert.NoError(t, writer.WriteRecord(TaskStarted([]string{"foo"})))
	assert.NoError(t, writer.WriteRecord(TaskStatus([]string{"foo"}, "Hello world!")))
	assert.NoError(t, writer.WriteRecord(TaskComplete([]string{"foo"})))
	assert.NoError(t, writer.WriteRecord(TaskFailed([]string{"foo"}, "something happened")))
	assert.NoError(t, writer.WriteRecord(TaskCanceled([]string{"foo"})))
	assert.NoError(t, writer.WriteRecord(TaskOutput([]string{"foo"}, []byte("bar"))))

	reader := NewJSONReader(buf)
	record, err := reader.ReadRecord()
	assert.NoError(t, err)
	assert.Equal(t, taskAddedKind, record.kind())
	assert.Nil(t, record.(TaskAddedRecord).Task)
	assert.Equal(t, "foo", record.(TaskAddedRecord).Name)

	record, err = reader.ReadRecord()
	assert.NoError(t, err)
	assert.Equal(t, taskStartedKind, record.kind())
	assert.Equal(t, []string{"foo"}, record.(TaskStartedRecord).Task)

	record, err = reader.ReadRecord()
	assert.NoError(t, err)
	assert.Equal(t, taskStatusKind, record.kind())
	assert.Equal(t, []string{"foo"}, record.(TaskStatusRecord).Task)
	assert.Equal(t, "Hello world!", record.(TaskStatusRecord).Status)

	record, err = reader.ReadRecord()
	assert.NoError(t, err)
	assert.Equal(t, taskCompleteKind, record.kind())
	assert.Equal(t, []string{"foo"}, record.(TaskCompleteRecord).Task)

	record, err = reader.ReadRecord()
	assert.NoError(t, err)
	assert.Equal(t, taskFailedKind, record.kind())
	assert.Equal(t, []string{"foo"}, record.(TaskFailedRecord).Task)
	assert.Equal(t, "something happened", record.(TaskFailedRecord).Error)

	record, err = reader.ReadRecord()
	assert.NoError(t, err)
	assert.Equal(t, taskCanceledKind, record.kind())
	assert.Equal(t, []string{"foo"}, record.(TaskCanceledRecord).Task)

	record, err = reader.ReadRecord()
	assert.NoError(t, err)
	assert.Equal(t, taskOutputKind, record.kind())
	assert.Equal(t, []string{"foo"}, record.(TaskOutputRecord).Task)
	assert.Equal(t, "bar", string(record.(TaskOutputRecord).Output))
}
