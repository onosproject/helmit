// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package job

import (
	"encoding/json"
	"github.com/onosproject/helmit/internal/k8s"
	"github.com/onosproject/helmit/internal/logging"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"path/filepath"
	"time"
)

const (
	configPath  = "/etc/helmit/config"
	secretsPath = "/etc/helmit/secrets"
	configFile  = "config.json"
	readyFile   = "/tmp/bin-ready"
	// HomeDir is the home directory of the helmit-runner container
	HomeDir = "/home/helmit"
	// ContextDir is the directory to which job contexts will be copied if specified
	ContextDir = "context"
)

const (
	defaultRoleBindingName = "cluster-test"
	defaultRoleName        = "cluster-admin"
)

// LoadConfig loads the job configuration
func LoadConfig(config any) error {
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

// LoadSecrets loads the job secrets
func LoadSecrets() (map[string]string, error) {
	secrets := make(map[string]string)
	files, err := os.ReadDir(secretsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return secrets, nil
		}
		return nil, err
	}
	for _, file := range files {
		if !file.IsDir() {
			bytes, err := os.ReadFile(file.Name())
			if err != nil {
				return nil, err
			}
			secrets[filepath.Base(file.Name())] = string(bytes)
		}
	}
	return secrets, nil
}

// Job manages the lifecycle of a Kubernetes job
type Job[T any] struct {
	ID              string
	Namespace       string
	CreateNamespace bool
	DeleteNamespace bool
	ServiceAccount  string
	Labels          map[string]string
	Annotations     map[string]string
	Image           string
	ImagePullPolicy corev1.PullPolicy
	Args            []string
	Env             map[string]string
	Secrets         map[string]string
	Context         string
	ValueFiles      map[string][]string
	Executable      string
	Config          T
	config          *rest.Config
	client          *kubernetes.Clientset
	pod             *corev1.Pod
}

func (j *Job[T]) init() error {
	if j.client != nil {
		return nil
	}

	config, err := k8s.GetConfig()
	if err != nil {
		panic(err)
	}
	j.config = config

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	j.client = client
	return nil
}

// GetStatus gets the status message and exit code of the given pod
func (j *Job[T]) GetStatus(ctx context.Context) (string, int, error) {
	for {
		pod, err := j.getPod(ctx)
		if err != nil {
			return "", 0, err
		} else if pod != nil {
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.Name == "job" && containerStatus.State.Terminated != nil {
					return containerStatus.State.Terminated.Message, int(containerStatus.State.Terminated.ExitCode), nil
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (j *Job[T]) getPod(ctx context.Context) (*corev1.Pod, error) {
	pods, err := j.client.CoreV1().Pods(j.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "job=" + j.ID,
	})
	if err != nil {
		return nil, err
	} else if len(pods.Items) > 0 {
		for _, pod := range pods.Items {
			return &pod, nil
		}
	}
	return nil, nil
}

func (j *Job[T]) waitForRunning(ctx context.Context, log logging.Logger) error {
	log.Logf("Waiting for Job to start running...")
	for {
		pod, err := j.getPod(ctx)
		if err != nil {
			return err
		} else if pod != nil && len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].State.Running != nil {
			j.pod = pod
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}
