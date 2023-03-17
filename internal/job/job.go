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

const (
	configPath = "/etc/helmit"
	specFile   = "spec.json"
	configFile = "config.json"
	readyFile  = "/tmp/job-ready"
	typeEnv    = "JOB_TYPE"
)

type Type string

const (
	ExecutorType Type = "executor"
	WorkerType   Type = "worker"
)

func GetType() Type {
	return Type(os.Getenv(typeEnv))
}

// Job is a job specification
type Job[C any] struct {
	Spec
	Config C
}

type Spec struct {
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
	ManagementPort  int
}

// Bootstrap bootstraps the job
func Bootstrap[C any]() (Job[C], error) {
	var job Job[C]
	awaitReady()
	if err := loadSpec(&job.Spec); err != nil {
		return job, err
	}
	if err := loadConfig(&job.Config); err != nil {
		return job, err
	}
	return job, nil
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

// loadSpec loads the job specification
func loadSpec(spec *Spec) error {
	bytes, err := os.ReadFile(filepath.Join(configPath, configFile))
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, spec)
	if err != nil {
		return err
	}
	return nil
}

// loadConfig loads the job configuration
func loadConfig(config any) error {
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
