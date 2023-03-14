// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package job

import (
	"encoding/json"
	corev1 "k8s.io/api/core/v1"
	"os"
	"path/filepath"
	"time"
)

const configPath = "/etc/helmit"
const configFile = "job.json"
const readyFile = "/tmp/job-ready"

// Config is a job configuration
type Config struct {
	ID              string
	Namespace       string
	ServiceAccount  string
	Labels          map[string]string
	Annotations     map[string]string
	Image           string
	ImagePullPolicy corev1.PullPolicy
	Executable      string
	Context         string
	Values          map[string][]string
	ValueFiles      map[string][]string
	Args            []string
	Env             map[string]string
	Timeout         time.Duration
	NoTeardown      bool
	Secrets         map[string]string
}

// Job is a job configuration
type Job struct {
	*Config
	JobConfig interface{}
	Type      string
}

// Bootstrap bootstraps the job
func Bootstrap(config interface{}) error {
	awaitReady()
	return LoadConfig(config)
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

// LoadConfig returns the job configuration
func LoadConfig(config interface{}) error {
	bytes, err := os.ReadFile(filepath.Join(configPath, configFile))
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, config)
	if err != nil {
		return err
	}
	return nil
}
