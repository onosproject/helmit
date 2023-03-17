package console

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestContextRun(t *testing.T) {
	reporter := NewReporter(os.Stdout)
	reporter.Start()
	defer reporter.Stop()

	context := NewContext(reporter)
	err := context.Run("Hello world!", func(task *Task) error {
		task.Log("Hello")
		task.Log("world!")
		return nil
	})
	assert.NoError(t, err)
}

func TestContextVerboseLog(t *testing.T) {
	reporter := NewReporter(os.Stdout, WithVerbose())
	reporter.Start()
	defer reporter.Stop()

	context := NewContext(reporter)
	err := context.Run("Hello world!", func(task *Task) error {
		task.Log("Hello")
		task.Log("world!")
		return nil
	})
	assert.NoError(t, err)
}

func TestContextRunAsync(t *testing.T) {
	reporter := NewReporter(os.Stdout)
	reporter.Start()
	defer reporter.Stop()

	context := NewContext(reporter)
	err := context.RunAsync("Hello world!", func(task *Task) error {
		task.Log("Hello")
		task.Log("world!")
		return nil
	}).Wait()
	assert.NoError(t, err)
}

func TestContextRunTaskRun(t *testing.T) {
	reporter := NewReporter(os.Stdout)
	reporter.Start()
	defer reporter.Stop()

	context := NewContext(reporter)
	err := context.Run("Hello world!", func(task *Task) error {
		return task.Run("again", func(task *Task) error {
			task.Log("Hello world again!")
			return nil
		})
	})
	assert.NoError(t, err)
}

func TestContextRunTaskRunAsync(t *testing.T) {
	reporter := NewReporter(os.Stdout)
	reporter.Start()
	defer reporter.Stop()

	context := NewContext(reporter)
	err := context.Run("Hello world!", func(task *Task) error {
		return task.RunAsync("again", func(task *Task) error {
			task.Log("Hello world again!")
			return nil
		}).Wait()
	})
	assert.NoError(t, err)
}

func TestContextRunTaskFork(t *testing.T) {
	reporter := NewReporter(os.Stdout)
	reporter.Start()
	defer reporter.Stop()

	context := NewContext(reporter)
	err := context.Run("Hello world!", func(task *Task) error {
		return task.Fork(func(status *Status) error {
			status.Report("foo")
			time.Sleep(time.Second)
			status.Report("bar")
			time.Sleep(time.Second)
			status.Report("baz")
			return nil
		}).Wait()
	})
	assert.NoError(t, err)
}
