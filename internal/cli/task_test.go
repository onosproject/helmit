package cli

import (
	"os"
	"testing"
	"time"
)

func TestTask(t *testing.T) {
	status := NewStatus(os.Stdout)
	status.Start()
	defer status.Stop()

	tasks := NewTaskManager(status)

	task := tasks.New("Hello world!")

	task1 := task.Fork("foo")
	task1.Run(func(log Logger) error {
		log.Log("foobar")
		time.Sleep(100 * time.Millisecond)
		log.Log("barbaz")
		time.Sleep(100 * time.Millisecond)
		log.Log("bazfoo")
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	task1.Run(func(log Logger) error {
		log.Log("bazbar")
		time.Sleep(200 * time.Millisecond)
		log.Log("barfoo")
		time.Sleep(200 * time.Millisecond)
		log.Log("foobaz")
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	//task1.Wait()

	task2 := task.Fork("bar")
	task2.Run(func(log Logger) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	//task2.Wait()

	task.Wait()
}
