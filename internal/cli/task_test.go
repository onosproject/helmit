package cli

import (
	"errors"
	"os"
	"sync"
	"testing"
	"time"
)

func TestTask(t *testing.T) {
	log := NewLogger(os.Stdout)
	log.Log("Hello world!")

	task := log.Task("Hello")
	task.Start()
	time.Sleep(2 * time.Second)
	task.Complete()

	task = log.Task("world!")
	task.Start()
	time.Sleep(1 * time.Second)

	wg := &sync.WaitGroup{}

	task1 := task.Task("foo")
	wg.Add(1)
	go func() {
		task1.Start()
		time.Sleep(time.Second)
		task1.Complete()
		wg.Done()
	}()

	task2 := task.Task("bar")
	wg.Add(1)
	go func() {
		task2.Start()
		time.Sleep(2 * time.Second)
		task2.Complete()
		wg.Done()
	}()

	/*
		sub1 := task2.Sub()
		wg.Add(1)
		go func() {
			time.Sleep(1500 * time.Millisecond)
			sub1.Log("foobar")
			time.Sleep(200 * time.Millisecond)
			sub1.Close()
			wg.Done()
		}()

		sub2 := task2.Sub()
		wg.Add(1)
		go func() {
			time.Sleep(500 * time.Millisecond)
			sub2.Log("barbaz")
			time.Sleep(500 * time.Millisecond)
			sub2.Close()
			wg.Done()
		}()
	*/

	wg.Wait()
	task.Complete()

	task = log.Task("Hello world!")
	task.Start()
	time.Sleep(time.Second)
	task.Caution(errors.New("caution"))

	task = log.Task("Hello world again!")
	task.Start()
	time.Sleep(time.Second)
	task.Fail(errors.New("fail"))
}
