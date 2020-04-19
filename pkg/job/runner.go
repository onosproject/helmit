// Copyright 2019-present Open Networking Foundation.
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

package job

import (
	"bufio"
	"encoding/json"
	"fmt"
	"path"
	"time"

	"github.com/onosproject/helmit/pkg/kubernetes"
	"github.com/onosproject/helmit/pkg/util/files"
	"github.com/onosproject/helmit/pkg/util/logging"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
)

const clusterRole = "kube-test-cluster"

// NewNamespace returns a new job namespace
func NewNamespace(namespace string) *Runner {
	return newRunner(namespace, true)
}

// newRunner returns a new job runner
func newRunner(namespace string, server bool) *Runner {
	return &Runner{
		Client: kubernetes.NewForNamespaceOrDie(namespace),
		server: server,
	}
}

// Runner manages test jobs within a namespace
type Runner struct {
	kubernetes.Client
	server bool
}

// RunJob runs the given job
func (n *Runner) RunJob(job *Job) (int, error) {
	if err := n.StartJob(job); err != nil {
		return 0, err
	}
	return n.WaitForExit(job)
}

// StartJob starts the given job
func (n *Runner) StartJob(job *Job) error {
	if err := n.startJob(job); err != nil {
		return err
	}
	go n.streamLogs(job)
	return nil
}

// streamLogs streams logs from the given pod
func (n *Runner) streamLogs(job *Job) {
	// Get the stream of logs for the pod
	pod, err := n.getPod(job, func(pod corev1.Pod) bool {
		return len(pod.Status.ContainerStatuses) > 0 &&
			pod.Status.ContainerStatuses[0].Ready
	})
	if err != nil || pod == nil {
		return
	}

	req := n.Clientset().CoreV1().Pods(n.Namespace()).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container: "job",
		Follow:    true,
	})
	reader, err := req.Stream()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer reader.Close()

	// Stream the logs to stdout
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		logging.Print(scanner.Text())
	}
}

// WaitForExit waits for the job to exit
func (n *Runner) WaitForExit(job *Job) (int, error) {
	_, status, err := n.getStatus(job)
	if err != nil {
		return 0, err
	}
	return status, nil
}

// CreateNamespace creates the namespace
func (n *Runner) CreateNamespace() error {
	return n.setupNamespace()
}

// DeleteNamespace deletes the namespace
func (n *Runner) DeleteNamespace() error {
	return n.teardownNamespace()
}

// setupNamespace sets up the test namespace
func (n *Runner) setupNamespace() error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: n.Namespace(),
			Labels: map[string]string{
				"test": n.Namespace(),
			},
		},
	}
	step := logging.NewStep(n.Namespace(), "Setup namespace")
	step.Start()
	_, err := n.Clientset().CoreV1().Namespaces().Create(ns)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return n.setupRBAC()
}

// setupRBAC sets up role based access controls for the cluster
func (n *Runner) setupRBAC() error {
	step := logging.NewStep(n.Namespace(), "Set up RBAC")
	step.Start()
	if err := n.createClusterRole(); err != nil {
		step.Fail(err)
		return err
	}
	if err := n.createClusterRoleBinding(); err != nil {
		step.Fail(err)
		return err
	}
	if err := n.createServiceAccount(); err != nil {
		step.Fail(err)
		return err
	}
	step.Complete()
	return nil
}

// createClusterRole creates the ClusterRole required by the Atomix controller and tests if not yet created
func (n *Runner) createClusterRole() error {
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRole,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"pods",
					"pods/log",
					"pods/exec",
					"services",
					"endpoints",
					"persistentvolumeclaims",
					"events",
					"configmaps",
					"secrets",
					"serviceaccounts",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"namespaces",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"apps",
				},
				Resources: []string{
					"deployments",
					"daemonsets",
					"replicasets",
					"statefulsets",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"policy",
				},
				Resources: []string{
					"poddisruptionbudgets",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"batch",
				},
				Resources: []string{
					"jobs",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"rbac.authorization.k8s.io",
				},
				Resources: []string{
					"roles",
					"rolebindings",
					"clusterroles",
					"clusterrolebindings",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"apiextensions.k8s.io",
				},
				Resources: []string{
					"customresourcedefinitions",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"storage.cloud.atomix.io",
					"cloud.atomix.io",
					"k8s.atomix.io",
				},
				Resources: []string{
					"*",
				},
				Verbs: []string{
					"*",
				},
			},
		},
	}
	_, err := n.Clientset().RbacV1().ClusterRoles().Create(role)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// createClusterRoleBinding creates the ClusterRoleBinding required by the test manager
func (n *Runner) createClusterRoleBinding() error {
	roleBinding, err := n.Clientset().RbacV1().ClusterRoleBindings().Get(clusterRole, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
		roleBinding = &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterRole,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      n.Namespace(),
					Namespace: n.Namespace(),
				},
			},
			RoleRef: rbacv1.RoleRef{
				Kind:     "ClusterRole",
				Name:     clusterRole,
				APIGroup: "rbac.authorization.k8s.io",
			},
		}
		_, err := n.Clientset().RbacV1().ClusterRoleBindings().Create(roleBinding)
		if err != nil && k8serrors.IsAlreadyExists(err) {
			return n.createClusterRoleBinding()
		}
		return err
	}

	roleBinding.Subjects = append(roleBinding.Subjects, rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      n.Namespace(),
		Namespace: n.Namespace(),
	})
	_, err = n.Clientset().RbacV1().ClusterRoleBindings().Update(roleBinding)
	if err != nil && k8serrors.IsConflict(err) {
		return n.createClusterRoleBinding()
	}
	return err
}

// createServiceAccount creates a ServiceAccount used by the test manager
func (n *Runner) createServiceAccount() error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n.Namespace(),
			Namespace: n.Namespace(),
		},
	}
	_, err := n.Clientset().CoreV1().ServiceAccounts(n.Namespace()).Create(serviceAccount)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// teardownNamespace tears down the cluster namespace
func (n *Runner) teardownNamespace() error {
	step := logging.NewStep(n.Namespace(), "Delete namespace %s", n.Namespace())
	step.Start()

	w, err := n.Clientset().CoreV1().Namespaces().Watch(metav1.ListOptions{
		LabelSelector: "test=" + n.Namespace(),
	})
	if err != nil {
		step.Fail(err)
	}

	err = n.Clientset().CoreV1().Namespaces().Delete(n.Namespace(), &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	for event := range w.ResultChan() {
		switch event.Type {
		case watch.Deleted:
			w.Stop()
		}
	}
	step.Complete()
	return nil
}

// startJob starts running a test job
func (n *Runner) startJob(job *Job) error {
	step := logging.NewStep(job.ID, "Starting job")
	step.Start()
	if err := n.createJob(job); err != nil {
		step.Fail(err)
		return err
	}
	if err := n.awaitJobRunning(job); err != nil {
		step.Fail(err)
		return err
	}
	if err := n.copyBinary(job); err != nil {
		step.Fail(err)
		return err
	}
	if err := n.runBinary(job); err != nil {
		step.Fail(err)
		return err
	}
	if err := n.copyValueFiles(job); err != nil {
		step.Fail(err)
		return err
	}
	if err := n.copyContext(job); err != nil {
		step.Fail(err)
		return err
	}
	if err := n.runJob(job); err != nil {
		step.Fail(err)
		return err
	}
	if err := n.awaitJobReady(job); err != nil {
		step.Fail(err)
		return err
	}
	step.Complete()
	return nil
}

// createJob creates the job to run tests
func (n *Runner) createJob(job *Job) error {
	step := logging.NewStep(job.ID, "Start job")
	step.Start()

	env := make([]corev1.EnvVar, 0, len(job.Env))
	for key, value := range job.Env {
		env = append(env, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}
	env = append(env, corev1.EnvVar{
		Name:  "SERVICE_NAMESPACE",
		Value: n.Namespace(),
	})
	env = append(env, corev1.EnvVar{
		Name:  "SERVICE_NAME",
		Value: job.ID,
	})
	env = append(env, corev1.EnvVar{
		Name:  "JOB_TYPE",
		Value: job.Type,
	})
	env = append(env, corev1.EnvVar{
		Name: "POD_NAMESPACE",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.namespace",
			},
		},
	})
	env = append(env, corev1.EnvVar{
		Name: "POD_NAME",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	})

	if n.server {
		servicePorts := []corev1.ServicePort{
			{
				Name: "management",
				Port: 5000,
			},
		}
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: job.ID,
				Labels: map[string]string{
					"job":  job.ID,
					"type": job.Type,
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"job": job.ID,
				},
				Ports: servicePorts,
			},
		}
		if _, err := n.Clientset().CoreV1().Services(n.Namespace()).Create(svc); err != nil {
			return err
		}
	}

	json, err := json.Marshal(job.JobConfig)
	if err != nil {
		step.Fail(err)
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job.ID,
			Namespace: n.Namespace(),
			Annotations: map[string]string{
				"job":  job.ID,
				"type": job.Type,
			},
		},
		Data: map[string]string{
			configFile: string(json),
		},
	}
	if _, err := n.Clientset().CoreV1().ConfigMaps(n.Namespace()).Create(cm); err != nil {
		return err
	}

	volumes := []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: job.ID,
					},
				},
			},
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "config",
			MountPath: configPath,
			ReadOnly:  true,
		},
	}

	var containerPorts []corev1.ContainerPort
	if n.server {
		containerPorts = []corev1.ContainerPort{
			{
				Name:          "management",
				ContainerPort: 5000,
			},
		}
	}

	var readinessProbe *corev1.Probe
	if n.server {
		readinessProbe = &corev1.Probe{
			Handler: corev1.Handler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(5000),
				},
			},
			PeriodSeconds:    1,
			FailureThreshold: 30,
		}
	} else {
		readinessProbe = &corev1.Probe{
			Handler: corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"stat",
						"/tmp/job-ready",
					},
				},
			},
			PeriodSeconds:    1,
			FailureThreshold: 30,
		}
	}

	zero := int32(0)
	one := int32(1)
	batchJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job.ID,
			Namespace: n.Namespace(),
			Annotations: map[string]string{
				"job":  job.ID,
				"type": job.Type,
			},
		},
		Spec: batchv1.JobSpec{
			Parallelism:  &one,
			Completions:  &one,
			BackoffLimit: &zero,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"job":  job.ID,
						"type": job.Type,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: n.Namespace(),
					RestartPolicy:      corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "job",
							Image:           job.Image,
							ImagePullPolicy: job.ImagePullPolicy,
							Args:            job.Args,
							Env:             env,
							Ports:           containerPorts,
							VolumeMounts:    volumeMounts,
							ReadinessProbe:  readinessProbe,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	if job.Timeout > 0 {
		timeoutSeconds := int64(job.Timeout / time.Second)
		batchJob.Spec.ActiveDeadlineSeconds = &timeoutSeconds
	}

	_, err = n.Clientset().BatchV1().Jobs(n.Namespace()).Create(batchJob)
	if err != nil {
		step.Fail(err)
		return err
	}
	step.Complete()
	return nil
}

// awaitJobRunning blocks until the test job creates a pod in the RUNNING state
func (n *Runner) awaitJobRunning(job *Job) error {
	for {
		pod, err := n.getPod(job, func(pod corev1.Pod) bool {
			return len(pod.Status.ContainerStatuses) > 0 &&
				pod.Status.ContainerStatuses[0].State.Running != nil
		})
		if err != nil {
			return err
		} else if pod != nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// awaitJobReady blocks until the test job creates a ready pod
func (n *Runner) awaitJobReady(job *Job) error {
	for {
		pod, err := n.getPod(job, func(pod corev1.Pod) bool {
			return len(pod.Status.ContainerStatuses) > 0 &&
				pod.Status.ContainerStatuses[0].Ready
		})
		if err != nil {
			return err
		} else if pod != nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// copyBinary copies the job binary to the pod
func (n *Runner) copyBinary(job *Job) error {
	if job.Executable == "" {
		return nil
	}

	step := logging.NewStep(job.ID, "Copy binary %s", path.Base(job.Executable))
	step.Start()

	pod, err := n.getPod(job, func(pod corev1.Pod) bool {
		return true
	})
	if err != nil {
		step.Fail(err)
		return err
	}

	err = files.Copy(n).
		From(job.Executable).
		To(job.Executable).
		On(pod.Name).
		Do()
	if err != nil {
		step.Fail(err)
		return err
	}
	step.Complete()
	return nil
}

// runBinary runs the job binary
func (n *Runner) runBinary(job *Job) error {
	if job.Executable == "" {
		return nil
	}

	step := logging.NewStep(job.ID, "Run binary %s", path.Base(job.Executable))
	step.Start()

	pod, err := n.getPod(job, func(pod corev1.Pod) bool {
		return true
	})
	if err != nil {
		step.Fail(err)
		return err
	}
	err = files.Echo(n).
		String(path.Base(job.Executable)).
		To("/tmp/bin-ready").
		On(pod.Name).
		Do()
	if err != nil {
		step.Fail(err)
		return err
	}
	return nil
}

// copyValueFiles copies the value files to the pod
func (n *Runner) copyValueFiles(job *Job) error {
	if job.ValueFiles == nil || len(job.ValueFiles) == 0 {
		return nil
	}

	step := logging.NewStep(job.ID, "Copy value files")
	step.Start()

	pod, err := n.getPod(job, func(pod corev1.Pod) bool {
		return true
	})
	if err != nil {
		step.Fail(err)
		return err
	}

	for _, valueFiles := range job.ValueFiles {
		for _, valueFile := range valueFiles {
			fileStep := logging.NewStep(job.ID, "Copy value file %s", valueFile)
			fileStep.Start()
			err := files.Copy(n).
				From(valueFile).
				To(valueFile).
				On(pod.Name).
				Do()
			if err != nil {
				fileStep.Fail(err)
				step.Fail(err)
				return err
			}
			fileStep.Complete()
		}
	}
	step.Complete()
	return nil
}

// copyContext copies the job context to the pod
func (n *Runner) copyContext(job *Job) error {
	if job.Context == "" {
		return nil
	}

	step := logging.NewStep(job.ID, "Copy Helm context")
	step.Start()

	pod, err := n.getPod(job, func(pod corev1.Pod) bool {
		return true
	})
	if err != nil {
		step.Fail(err)
		return err
	}

	err = files.Copy(n).
		From(job.Context).
		To(job.Context).
		On(pod.Name).
		Do()
	if err != nil {
		step.Fail(err)
		return err
	}
	step.Complete()
	return nil
}

// runJob runs the job
func (n *Runner) runJob(job *Job) error {
	step := logging.NewStep(job.ID, "Run job")
	step.Start()

	pod, err := n.getPod(job, func(pod corev1.Pod) bool {
		return true
	})
	if err != nil {
		step.Fail(err)
		return err
	}
	err = files.Echo(n).
		String(path.Base(job.Context)).
		To(readyFile).
		On(pod.Name).
		Do()
	if err != nil {
		step.Fail(err)
		return err
	}
	return nil
}

// getStatus gets the status message and exit code of the given pod
func (n *Runner) getStatus(job *Job) (string, int, error) {
	for {
		pod, err := n.getPod(job, func(pod corev1.Pod) bool {
			return len(pod.Status.ContainerStatuses) > 0 &&
				pod.Status.ContainerStatuses[0].State.Terminated != nil
		})
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

// getPod finds the Pod for the given test
func (n *Runner) getPod(job *Job, predicate func(pod corev1.Pod) bool) (*corev1.Pod, error) {
	pods, err := n.Clientset().CoreV1().Pods(n.Namespace()).List(metav1.ListOptions{
		LabelSelector: "job=" + job.ID,
	})
	if err != nil {
		return nil, err
	} else if len(pods.Items) > 0 {
		for _, pod := range pods.Items {
			if predicate(pod) {
				return &pod, nil
			}
		}
	}
	return nil, nil
}
