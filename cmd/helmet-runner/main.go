// Copyright 2020-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const readyFile = "/tmp/bin-ready"

func main() {
	awaitReady()
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// awaitReady waits for the job to become ready
func awaitReady() {
	for {
		if isReady() {
			return
		}
		time.Sleep(time.Second)
	}
}

// isReady checks if the job is ready
func isReady() bool {
	info, err := os.Stat(readyFile)
	return err == nil && !info.IsDir()
}

// getBinaryFile returns the binary file name
func getBinaryFile() (string, error) {
	file, err := os.Open(readyFile)
	if err != nil {
		return "", err
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	fileName := strings.TrimSpace(string(bytes))
	return fileName, nil
}

// run runs the main
func run() error {
	fileName, err := getBinaryFile()
	if err != nil {
		return err
	}
	absPath, err := filepath.Abs(fileName)
	if err != nil {
		return err
	}
	cmd := exec.Command(absPath)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Env = os.Environ()
	return cmd.Run()
}
