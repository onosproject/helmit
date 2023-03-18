package console

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"strings"
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
	}).Await()
	assert.NoError(t, err)

	err = Await(
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
	}).Await()
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
	}).Await()
	assert.Error(t, err)

	err = Await(
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
		}).Await()
	}).Join()
	assert.NoError(t, err)

	err = context.Fork("I'm the parent", func(context *Context) error {
		return Await(
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
		}).Await()
	}).Join()
	assert.Error(t, err)

	err = context.Fork("I'm the parent", func(context *Context) error {
		return Await(
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
			}).Await()
		}).Join()
		assert.Error(t, err)

		err = jsonContext.Fork("I'm the parent", func(context *Context) error {
			return Await(
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

const restoreSample = "{\"newProgress\":{\"address\":[],\"message\":\"Starting workers\"}}\n{\"progressStart\":{\"address\":[0]}}\n{\"newProgress\":{\"address\":[0],\"message\":\"Starting worker 0\"}}\n{\"progressStart\":{\"address\":[0,0]}}\n{\"newProgress\":{\"address\":[0,0],\"message\":\"Setting up cluster\"}}\n{\"progressStart\":{\"address\":[0,0,0]}}\n{\"newStatus\":{\"address\":[0,0,0]}}\n{\"statusUpdate\":{\"address\":[0,0,0,0],\"message\":\"Creating Job\"}}\n{\"statusUpdate\":{\"address\":[0,0,0,0],\"message\":\"Creating ServiceAccount\"}}\n{\"statusUpdate\":{\"address\":[0,0,0,0],\"message\":\"Creating ClusterRoleBinding\"}}\n{\"statusUpdate\":{\"address\":[0,0,0,0],\"message\":\"Creating Secret\"}}\n{\"statusUpdate\":{\"address\":[0,0,0,0],\"message\":\"Waiting for job to start\"}}\n{\"statusDone\":{\"address\":[0,0,0,0]}}\n{\"progressFinish\":{\"address\":[0,0,0]}}\n{\"newProgress\":{\"address\":[0,0],\"message\":\"Starting job\"}}\n{\"progressStart\":{\"address\":[0,0,1]}}\n{\"newStatus\":{\"address\":[0,0,1]}}\n{\"newStatus\":{\"address\":[0,0,1]}}\n{\"statusUpdate\":{\"address\":[0,0,1,1],\"message\":\"Copying capable-pegasus\"}}\n{\"statusUpdate\":{\"address\":[0,0,1,1],\"message\":\"Copying atomix\"}}\n{\"statusDone\":{\"address\":[0,0,1,1]}}\n{\"statusDone\":{\"address\":[0,0,1,1]}}\n{\"newStatus\":{\"address\":[0,0,1]}}\n{\"statusUpdate\":{\"address\":[0,0,1,2],\"message\":\"Waiting for ready\"}}\n{\"statusDone\":{\"address\":[0,0,1,2]}}\n{\"progressFinish\":{\"address\":[0,0,1]}}\n{\"progressFinish\":{\"address\":[0,0]}}\n{\"progressFinish\":{\"address\":[0]}}\n{\"newProgress\":{\"address\":[],\"message\":\"Running suite 'chart'\"}}\n{\"progressStart\":{\"address\":[1]}}\n{\"newProgress\":{\"address\":[1],\"message\":\"Setting up the suite\"}}\n{\"progressStart\":{\"address\":[1,0]}}\n{\"progressFinish\":{\"address\":[1,0]}}\n{\"newProgress\":{\"address\":[1],\"message\":\"TestLocalInstall\"}}\n{\"newProgress\":{\"address\":[1],\"message\":\"TestRemoteInstall\"}}\n{\"progressStart\":{\"address\":[1,1]}}\n{\"newProgress\":{\"address\":[1,1],\"message\":\"Setting up the test\"}}\n{\"progressStart\":{\"address\":[1,1,0]}}\n{\"progressFinish\":{\"address\":[1,1,0]}}\n{\"newProgress\":{\"address\":[1,1],\"message\":\"Running the test\"}}\n{\"progressStart\":{\"address\":[1,1,1]}}"

func TestRestoreSample(t *testing.T) {
	reader := strings.NewReader(restoreSample)
	context := NewContext(os.Stdout, WithFormat(LiveFormat))
	defer context.Close()
	err := context.Fork("Restoring context", func(context *Context) error {
		return context.Restore(reader)
	}).Join()
	time.Sleep(time.Second)
	assert.NoError(t, err)
}
