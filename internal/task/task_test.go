package task

import (
	"github.com/onosproject/helmit/internal/log"
	"io"
	"os"
	"testing"
	"time"
)

func TestTask(t *testing.T) {
	uiwriter := log.NewUIWriter(os.Stdout)
	defer uiwriter.Close()

	reader, writer := io.Pipe()
	jsonwriter := log.NewJSONWriter(writer)
	jsonreader := log.NewJSONReader(reader)

	go log.Copy(uiwriter, jsonreader)

	New(jsonwriter, "restore").Run(func(context Context) error {
		time.Sleep(1 * time.Second)
		return nil
	})

	New(jsonwriter, "Hello world!").Run(func(context Context) error {
		err := context.NewTask("foo").Run(func(context Context) error {
			err := context.NewTask("foo").Run(func(context Context) error {
				context.Status().Set("Hello world!")
				time.Sleep(1 * time.Second)
				context.Writer().Write([]byte("Hello world!\n"))
				time.Sleep(2 * time.Second)
				context.Writer().Write([]byte("Hello world again!\n"))
				time.Sleep(1 * time.Second)
				return nil
			})
			if err != nil {
				return err
			}

			err = context.NewTask("bar").Run(func(context Context) error {
				time.Sleep(1 * time.Second)
				return nil
			})
			if err != nil {
				return err
			}

			err = context.NewTask("baz").Run(func(context Context) error {
				time.Sleep(2 * time.Second)
				return nil
			})
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}

		err = context.NewTask("bar").Run(func(context Context) error {
			return Await(
				context.NewTask("one").Start(func(context Context) error {
					time.Sleep(1 * time.Second)
					return nil
				}),
				context.NewTask("two").Start(func(context Context) error {
					time.Sleep(2 * time.Second)
					return nil
				}),
				context.NewTask("three").Start(func(context Context) error {
					time.Sleep(1 * time.Second)
					return nil
				}))
		})
		if err != nil {
			return err
		}

		err = context.NewTask("baz").Run(func(context Context) error {
			tasks := []Task{
				context.NewTask("one"),
				context.NewTask("two"),
				context.NewTask("three"),
				context.NewTask("four"),
			}
			var err error
			for _, t := range tasks {
				e := t.Run(func(context Context) error {
					time.Sleep(2 * time.Second)
					return nil
				})
				if e != nil {
					err = e
				}
			}
			return err
		})
		if err != nil {
			return err
		}
		return nil
	})
}
