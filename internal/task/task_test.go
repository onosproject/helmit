package task

import (
	"errors"
	"os"
	"sync"
	"testing"
	"time"
)

func TestTask(t *testing.T) {
	manager := NewManager(os.Stdout)
	manager.Start()
	defer manager.Stop()

	task := manager.Task("Hello")
	task.Start()
	time.Sleep(2 * time.Second)
	task.Done()

	task = manager.Task("world!")
	task.Start()
	time.Sleep(1 * time.Second)

	wg := &sync.WaitGroup{}

	task1 := task.Task("foo")
	wg.Add(1)
	go func() {
		task1.Start()
		time.Sleep(time.Second)
		task1.Done()
		wg.Done()
	}()

	task2 := task.Task("bar")
	wg.Add(1)
	go func() {
		task2.Start()
		time.Sleep(2 * time.Second)
		task2.Done()
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
		sub1.Done()
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
		sub2.Done()
		wg.Done()
	}()

	wg.Wait()
	task.Done()

	task = manager.Task("Hello world!")
	task.Start()
	time.Sleep(time.Second)
	task.Warn(errors.New("caution"))

	task = manager.Task("Hello world again!")
	task.Start()
	time.Sleep(time.Second)
	task.Error(errors.New("fail"))
}

func TestSubTask(t *testing.T) {
	manager := NewManager(os.Stdout)
	manager.Start()
	defer manager.Stop()

	task := manager.Task("Hello")
	task.Start()
	time.Sleep(2 * time.Second)

	wg := &sync.WaitGroup{}

	foo := task.Task("foo")
	wg.Add(1)
	go func() {
		foo.Start()
		time.Sleep(time.Second)
		foo.Done()
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
		sub.Done()
		wg.Done()
	}()

	wg.Wait()
	task.Done()
}
