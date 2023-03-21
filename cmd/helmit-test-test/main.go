package main

import (
	"os"
	"testing"
)

func main() {
	var tests []testing.InternalTest
	tests = append(tests, testing.InternalTest{
		Name: "foo",
		F: func(t *testing.T) {

		},
	})

	os.Args = []string{
		os.Args[0],
		"-test.v",
	}
	testing.Main(func(_, _ string) (bool, error) { return true, nil }, tests, nil, nil)
	//testing.Init()
	//if !testing.RunTests(func(_, _ string) (bool, error) { return true, nil }, tests) {
	//	os.Exit(1)
	//}
	//os.Exit(0)
}
