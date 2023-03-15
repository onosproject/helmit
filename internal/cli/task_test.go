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

	sub1 := task1.SubTask()
	wg.Add(1)
	go func() {
		time.Sleep(1500 * time.Millisecond)
		sub1.Log("foobar")
		time.Sleep(200 * time.Millisecond)
		sub1.Log("barbaz")
		time.Sleep(200 * time.Millisecond)
		sub1.Log("bazfoo")
		time.Sleep(200 * time.Millisecond)
		sub1.Close()
		wg.Done()
	}()

	sub2 := task1.SubTask()
	wg.Add(1)
	go func() {
		time.Sleep(500 * time.Millisecond)
		sub2.Log("bazbar")
		time.Sleep(300 * time.Millisecond)
		sub2.Log("barfoo")
		time.Sleep(300 * time.Millisecond)
		sub2.Log("foobaz")
		time.Sleep(300 * time.Millisecond)
		sub2.Close()
		wg.Done()
	}()

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

func TestSubTask(t *testing.T) {
	log := NewLogger(os.Stdout)
	log.Log("Hello world!")

	task := log.Task("Hello")
	task.Start()
	time.Sleep(2 * time.Second)

	wg := &sync.WaitGroup{}

	foo := task.Task("foo")
	wg.Add(1)
	go func() {
		foo.Start()
		time.Sleep(time.Second)
		foo.Complete()
		wg.Done()
	}()

	sub := foo.SubTask()
	wg.Add(1)
	go func() {
		time.Sleep(1500 * time.Millisecond)
		sub.Log("foobar")
		time.Sleep(200 * time.Millisecond)
		sub.Log("barbaz")
		time.Sleep(200 * time.Millisecond)
		sub.Log("bazfoo")
		time.Sleep(200 * time.Millisecond)
		sub.Close()
		wg.Done()
	}()

	wg.Wait()
	task.Complete()
}
