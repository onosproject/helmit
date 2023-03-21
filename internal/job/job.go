package job

import (
	"encoding/json"
	"github.com/onosproject/helmit/internal/k8s"
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
	configPath = "/etc/helmit"
	configFile = "config.json"
)

const (
	defaultServiceAccountName = "helmit"
	defaultRoleBindingName    = "helmit"
	defaultRoleName           = "helmit"
	helmitSecretsName         = "helmit"
)

// Bootstrap bootstraps the job
func Bootstrap(config any) error {
	return loadConfig(&config)
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

type Job struct {
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
	Timeout         time.Duration
	Config          any
	config          *rest.Config
	client          *kubernetes.Clientset
	pod             *corev1.Pod
}

func (j *Job) init() error {
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
func (j *Job) GetStatus(ctx context.Context) (string, int, error) {
	for {
		pod, err := j.getPod(ctx)
		if err != nil {
			return "", 0, err
		} else if pod != nil {
			state := pod.Status.ContainerStatuses[0].State
			if state.Terminated != nil {
				return state.Terminated.Message, int(state.Terminated.ExitCode), nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (j *Job) getPod(ctx context.Context) (*corev1.Pod, error) {
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

func (j *Job) waitForRunning(ctx context.Context) error {
	for {
		pod, err := j.getPod(ctx)
		if err != nil {
			return err
		} else if pod != nil && pod.Status.ContainerStatuses[0].State.Running != nil {
			j.pod = pod
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}
