package console

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
	"time"
)

func TestRootFork(t *testing.T) {
	context := NewContext(os.Stdout)
	defer context.Close()
	err := context.Fork("Hello world!", func(context *Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}).Join()
	assert.NoError(t, err)

	err = Join(
		context.Fork("Hello world 1", func(context *Context) error {
			time.Sleep(400 * time.Millisecond)
			return nil
		}),
		context.Fork("Hello world 2", func(context *Context) error {
			time.Sleep(800 * time.Millisecond)
			return nil
		}))
	assert.NoError(t, err)
}

func TestRootRun(t *testing.T) {
	context := NewContext(os.Stdout)
	defer context.Close()
	err := context.Run(func(status *Status) error {
		status.Report("started")
		time.Sleep(200 * time.Millisecond)
		status.Report("continuing")
		time.Sleep(200 * time.Millisecond)
		status.Report("finished")
		return nil
	}).Wait()
	assert.NoError(t, err)

	err = Wait(
		context.Run(func(status *Status) error {
			status.Report("started")
			time.Sleep(400 * time.Millisecond)
			status.Report("continuing")
			time.Sleep(400 * time.Millisecond)
			status.Report("finished")
			return nil
		}),
		context.Run(func(status *Status) error {
			status.Report("started")
			time.Sleep(800 * time.Millisecond)
			status.Report("continuing")
			time.Sleep(800 * time.Millisecond)
			status.Report("finished")
			return nil
		}))
	assert.NoError(t, err)

	err = context.Run(func(status *Status) error {
		fmt.Fprintln(status.Writer(), "Hello")
		time.Sleep(time.Second)
		fmt.Fprintln(status.Writer(), "world!")
		time.Sleep(time.Second)
		return nil
	}).Wait()
	assert.NoError(t, err)
}

func TestRootForkError(t *testing.T) {
	context := NewContext(os.Stdout)
	defer context.Close()
	err := context.Fork("Hello world!", func(context *Context) error {
		time.Sleep(200 * time.Millisecond)
		return errors.New("oops")
	}).Join()
	assert.Error(t, err)

	err = Join(
		context.Fork("Hello world 1", func(context *Context) error {
			time.Sleep(400 * time.Millisecond)
			return errors.New("oops")
		}),
		context.Fork("Hello world 2", func(context *Context) error {
			time.Sleep(800 * time.Millisecond)
			return nil
		}))
	assert.Error(t, err)
}

func TestRootRunError(t *testing.T) {
	context := NewContext(os.Stdout)
	defer context.Close()
	err := context.Run(func(status *Status) error {
		status.Report("started")
		time.Sleep(200 * time.Millisecond)
		status.Report("oops")
		return errors.New("oops")
	}).Wait()
	assert.Error(t, err)

	err = Wait(
		context.Run(func(status *Status) error {
			status.Report("started")
			time.Sleep(400 * time.Millisecond)
			status.Report("oops")
			return errors.New("oops")
		}),
		context.Run(func(status *Status) error {
			status.Report("started")
			time.Sleep(800 * time.Millisecond)
			status.Report("continuing")
			time.Sleep(800 * time.Millisecond)
			status.Report("finished")
			return nil
		}))
	assert.Error(t, err)
}

func TestNestedFork(t *testing.T) {
	context := NewContext(os.Stdout)
	defer context.Close()
	err := context.Fork("I'm the parent", func(context *Context) error {
		return context.Fork("I'm the child", func(context *Context) error {
			time.Sleep(400 * time.Millisecond)
			return nil
		}).Join()
	}).Join()
	assert.NoError(t, err)

	err = context.Fork("I'm the parent", func(context *Context) error {
		return Join(
			context.Fork("I'm the first child", func(context *Context) error {
				time.Sleep(200 * time.Millisecond)
				return nil
			}),
			context.Fork("I'm the second child", func(context *Context) error {
				time.Sleep(800 * time.Millisecond)
				return nil
			}))
	}).Join()
	assert.NoError(t, err)
}

func TestNestedRun(t *testing.T) {
	context := NewContext(os.Stdout)
	defer context.Close()
	err := context.Fork("I'm the parent", func(context *Context) error {
		return context.Run(func(status *Status) error {
			status.Log("I'm the child")
			return nil
		}).Wait()
	}).Join()
	assert.NoError(t, err)

	err = context.Fork("I'm the parent", func(context *Context) error {
		return Wait(
			context.Run(func(status *Status) error {
				status.Report("I'm")
				time.Sleep(500 * time.Millisecond)
				status.Report("the")
				time.Sleep(500 * time.Millisecond)
				status.Report("first")
				time.Sleep(500 * time.Millisecond)
				status.Report("child")
				return nil
			}),
			context.Run(func(status *Status) error {
				status.Report("I'm")
				time.Sleep(500 * time.Millisecond)
				status.Report("the")
				time.Sleep(500 * time.Millisecond)
				status.Report("second")
				time.Sleep(500 * time.Millisecond)
				status.Report("one")
				return nil
			}),
			context.Run(func(status *Status) error {
				status.Log("And I just like to log stuff")
				time.Sleep(time.Second)
				status.Log("Logging is fun!")
				return nil
			}))
	}).Join()
	assert.NoError(t, err)
}

func TestNestedForkError(t *testing.T) {
	context := NewContext(os.Stdout)
	defer context.Close()
	err := context.Fork("I'm the parent", func(context *Context) error {
		return context.Fork("I'm the child", func(context *Context) error {
			time.Sleep(400 * time.Millisecond)
			return errors.New("oops")
		}).Join()
	}).Join()
	assert.Error(t, err)

	err = context.Fork("I'm the parent", func(context *Context) error {
		return Join(
			context.Fork("I'm the first child", func(context *Context) error {
				time.Sleep(200 * time.Millisecond)
				return nil
			}),
			context.Fork("I'm the second child", func(context *Context) error {
				time.Sleep(800 * time.Millisecond)
				return errors.New("oops")
			}))
	}).Join()
	assert.Error(t, err)
}

func TestNestedRunError(t *testing.T) {
	context := NewContext(os.Stdout)
	defer context.Close()
	err := context.Fork("I'm the parent", func(context *Context) error {
		return context.Run(func(status *Status) error {
			status.Log("I'm the only child")
			return errors.New("oops")
		}).Wait()
	}).Join()
	assert.Error(t, err)

	err = context.Fork("I'm the parent", func(context *Context) error {
		return Wait(
			context.Run(func(status *Status) error {
				status.Report("I'm")
				time.Sleep(500 * time.Millisecond)
				status.Report("the")
				time.Sleep(500 * time.Millisecond)
				status.Report("first")
				time.Sleep(500 * time.Millisecond)
				status.Report("child")
				return nil
			}),
			context.Run(func(status *Status) error {
				status.Report("I'm")
				time.Sleep(500 * time.Millisecond)
				status.Report("the")
				time.Sleep(500 * time.Millisecond)
				status.Report("second")
				time.Sleep(500 * time.Millisecond)
				status.Report("one")
				return nil
			}),
			context.Run(func(status *Status) error {
				status.Log("And I just like to log stuff")
				time.Sleep(time.Second)
				status.Log("Logging is fun!")
				return errors.New("oops")
			}))
	}).Join()
	assert.Error(t, err)
}

func TestRestore(t *testing.T) {
	reader, writer := io.Pipe()

	go func() {
		jsonContext := NewContext(writer, WithFormat(JSONFormat))
		defer jsonContext.Close()
		defer writer.Close()

		err := jsonContext.Fork("I'm the parent", func(context *Context) error {
			return context.Run(func(status *Status) error {
				status.Log("I'm the only child")
				return errors.New("oops")
			}).Wait()
		}).Join()
		assert.Error(t, err)

		err = jsonContext.Fork("I'm the parent", func(context *Context) error {
			return Wait(
				context.Run(func(status *Status) error {
					status.Report("I'm")
					time.Sleep(500 * time.Millisecond)
					status.Report("the")
					time.Sleep(500 * time.Millisecond)
					status.Report("first")
					time.Sleep(500 * time.Millisecond)
					status.Report("child")
					return nil
				}),
				context.Run(func(status *Status) error {
					status.Report("I'm")
					time.Sleep(500 * time.Millisecond)
					status.Report("the")
					time.Sleep(500 * time.Millisecond)
					status.Report("second")
					time.Sleep(500 * time.Millisecond)
					status.Report("one")
					return nil
				}),
				context.Run(func(status *Status) error {
					status.Log("And I just like to log stuff")
					time.Sleep(time.Second)
					status.Log("Logging is fun!")
					return errors.New("oops")
				}))
		}).Join()
		assert.Error(t, err)
	}()

	liveContext := NewContext(os.Stdout, WithFormat(LiveFormat))
	defer liveContext.Close()

	err := liveContext.Fork("Restoring context", func(context *Context) error {
		return context.Restore(reader)
	}).Join()
	assert.NoError(t, err)
}
