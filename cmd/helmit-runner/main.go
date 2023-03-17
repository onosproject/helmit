// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
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
	bytes, err := os.ReadFile(readyFile)
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
