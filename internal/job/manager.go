// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package job

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/onosproject/helmit/internal/console"
	"github.com/onosproject/helmit/internal/k8s"
	"io"
	"k8s.io/client-go/kubernetes"
	"path"
	"time"

	"google.golang.org/grpc/codes"

	"google.golang.org/grpc/status"

	"github.com/onosproject/helmit/internal/files"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const defaultServiceAccountName = "cluster-test"
const defaultRoleBindingName = "cluster-test"
const defaultRoleName = "cluster-admin"
const helmitSecretsName = "helmit-secrets"

// NewManager returns a new job manager
func NewManager[C any](jobType Type) *Manager[C] {
	config, err := k8s.GetConfig()
	if err != nil {
		panic(err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	return &Manager[C]{
		Type:   jobType,
		client: client,
	}
}

// Manager manages test jobs within a namespace
type Manager[C any] struct {
	Type   Type
	client *kubernetes.Clientset
}

// Start starts the given job
func (m *Manager[C]) Start(job Job[C], context *console.Context) error {
	if err := m.startJob(job, context); err != nil {
		return err
	}
	return nil
}

func (m *Manager[C]) Run(job Job[C], context *console.Context) error {
	return context.Fork("Running job", func(context *console.Context) error {
		reader, err := m.streamLogs(job)
		if err != nil {
			return err
		}
		defer reader.Close()
		return context.Restore(reader)
	}).Join()
}

// streamLogs streams logs from the given pod
func (m *Manager[C]) streamLogs(job Job[C]) (io.ReadCloser, error) {
	// Get the stream of logs for the pod
	pod, err := m.getPod(job, func(pod corev1.Pod) bool {
		return len(pod.Status.ContainerStatuses) > 0 &&
			pod.Status.ContainerStatuses[0].State.Running != nil
	})
	if err != nil {
		return nil, err
	} else if pod == nil {
		return nil, errors.New("could not find pod")
	}

	req := m.client.CoreV1().Pods(job.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container: "job",
		Follow:    true,
	})
	return req.Stream(context.Background())
}

func (m *Manager[C]) readLogs(job Job[C], pod *corev1.Pod, status *console.Status) error {
	req := m.client.CoreV1().Pods(job.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container: "job",
	})
	reader, err := req.Stream(context.Background())
	if err != nil {
		return err
	}
	bytes, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	status.Report(string(bytes))
	return nil
}

// Stop stops the job and waits for it to exit
func (m *Manager[C]) Stop(job Job[C]) (int, error) {
	_, status, err := m.getStatus(job)
	_ = m.finishJob(job)
	if err != nil {
		return 0, err
	}
	return status, nil
}

// createServiceAccount creates a ServiceAccount used by the test manager
func (m *Manager[C]) createServiceAccount(job Job[C]) error {
	jobObj, err := m.client.BatchV1().Jobs(job.Namespace).Get(context.Background(), job.ID, metav1.GetOptions{})
	if err != nil {
		return err
	}

	serviceAccountName := job.ServiceAccount
	if serviceAccountName == "" {
		serviceAccountName = defaultServiceAccountName
	}

	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: job.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       jobObj.Name,
					UID:        jobObj.UID,
					Kind:       "Job",
					APIVersion: "batch/v1",
				},
			},
		},
	}
	_, err = m.client.CoreV1().ServiceAccounts(job.Namespace).Create(context.Background(), serviceAccount, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// createClusterRoleBinding creates the ClusterRoleBinding required by the test manager
func (m *Manager[C]) createClusterRoleBinding(job Job[C]) error {
	serviceAccountName := job.ServiceAccount
	if serviceAccountName == "" {
		serviceAccountName = defaultServiceAccountName
	}
	roleBinding, err := m.client.RbacV1().ClusterRoleBindings().Get(context.Background(), defaultRoleBindingName, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
		roleBinding = &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: defaultRoleBindingName,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      serviceAccountName,
					Namespace: job.Namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				Kind:     "ClusterRole",
				Name:     defaultRoleName,
				APIGroup: "rbac.authorization.k8s.io",
			},
		}

	}
	roleBinding.Subjects = append(roleBinding.Subjects, rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      serviceAccountName,
		Namespace: job.Namespace,
	})
	_, err = m.client.RbacV1().ClusterRoleBindings().Update(context.Background(), roleBinding, metav1.UpdateOptions{})
	if err != nil && k8serrors.IsConflict(err) {
		return m.createClusterRoleBinding(job)
	}
	return err
}

// startJob starts running a test job
func (m *Manager[C]) startJob(job Job[C], context *console.Context) error {
	err := context.Fork("Setting up cluster", func(context *console.Context) error {
		return context.Run(func(status *console.Status) error {
			if job.CreateNamespace {
				status.Report("Creating namespace")
				if err := m.createNamespace(job); err != nil {
					return err
				}
			}
			status.Report("Creating Job")
			if err := m.createJob(job); err != nil {
				return err
			}
			status.Report("Creating ServiceAccount")
			if err := m.createServiceAccount(job); err != nil {
				return err
			}
			status.Report("Creating ClusterRoleBinding")
			if err := m.createClusterRoleBinding(job); err != nil {
				return err
			}
			status.Report("Creating Secret")
			if err := m.createSecrets(job); err != nil {
				return err
			}
			status.Report("Waiting for job to start")
			if err := m.awaitJobRunning(job); err != nil {
				return err
			}
			return nil
		}).Wait()
	}).Join()
	if err != nil {
		return err
	}

	err = context.Fork("Starting job", func(context *console.Context) error {
		var waiters []console.Waiter
		if job.Executable != "" {
			waiters = append(waiters, context.Run(func(status *console.Status) error {
				status.Reportf("Copying %s", job.Executable)
				return m.copyBinary(job)
			}))
		}
		if job.Context != "" {
			waiters = append(waiters, context.Run(func(status *console.Status) error {
				status.Reportf("Copying %s", job.Context)
				return m.copyContext(job)
			}))
		}
		if len(job.ValueFiles) != 0 {
			for _, valueFiles := range job.ValueFiles {
				for _, valueFile := range valueFiles {
					waiters = append(waiters, func(file string) console.Waiter {
						return context.Run(func(status *console.Status) error {
							status.Reportf("Copying %s", file)
							return m.copyValuesFile(job, valueFile)
						})
					}(valueFile))
				}
			}
		}
		err := console.Wait(waiters...)
		if err != nil {
			return err
		}

		context.Run(func(status *console.Status) error {
			status.Report("Waiting for ready")
			if err := m.runBinary(job); err != nil {
				return err
			}
			if err := m.runJob(job); err != nil {
				return err
			}
			if err := m.awaitJobReady(job); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	}).Join()
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager[C]) createNamespace(job Job[C]) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: job.Namespace,
			Annotations: map[string]string{
				"job": job.ID,
			},
		},
	}
	if _, err := m.client.CoreV1().Namespaces().Create(context.Background(), namespace, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

// createJob creates the job to run tests
func (m *Manager[C]) createJob(job Job[C]) error {
	env := make([]corev1.EnvVar, 0, len(job.Env))
	for key, value := range job.Env {
		env = append(env, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}
	env = append(env, corev1.EnvVar{
		Name:  typeEnv,
		Value: string(m.Type),
	})
	env = append(env, corev1.EnvVar{
		Name:  "SERVICE_NAMESPACE",
		Value: job.Namespace,
	})
	env = append(env, corev1.EnvVar{
		Name:  "SERVICE_NAME",
		Value: job.ID,
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

	specJSON, err := json.Marshal(job.Spec)
	if err != nil {
		return err
	}
	configJSON, err := json.Marshal(job.Config)
	if err != nil {
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
	var readinessProbe *corev1.Probe
	if job.ManagementPort != 0 {
		containerPorts = append(containerPorts, corev1.ContainerPort{
			Name:          "management",
			ContainerPort: int32(job.ManagementPort),
		})
		readinessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(job.ManagementPort),
				},
			},
			PeriodSeconds:    1,
			FailureThreshold: 30,
		}
	} else {
		readinessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
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

	serviceAccount := job.ServiceAccount
	if serviceAccount == "" {
		serviceAccount = defaultServiceAccountName
	}

	labels := job.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["job"] = job.ID

	annotations := job.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	zero := int32(0)
	one := int32(1)
	batchJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job.ID,
			Namespace: job.Namespace,
			Annotations: map[string]string{
				"job": job.ID,
			},
		},
		Spec: batchv1.JobSpec{
			Parallelism:  &one,
			Completions:  &one,
			BackoffLimit: &zero,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccount,
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

	_, err = m.client.BatchV1().Jobs(job.Namespace).Create(context.Background(), batchJob, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	jobObj, err := m.client.BatchV1().Jobs(job.Namespace).Get(context.Background(), job.ID, metav1.GetOptions{})
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job.ID,
			Namespace: job.Namespace,
			Annotations: map[string]string{
				"job": job.ID,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       jobObj.Name,
					UID:        jobObj.UID,
					Kind:       "Job",
					APIVersion: "batch/v1",
				},
			},
		},
		Data: map[string]string{
			specFile:   string(specJSON),
			configFile: string(configJSON),
		},
	}

	if _, err := m.client.CoreV1().ConfigMaps(job.Namespace).Create(context.Background(), cm, metav1.CreateOptions{}); err != nil {
		return err
	}

	if job.ManagementPort != 0 {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      job.ID,
				Namespace: job.Namespace,
				Labels: map[string]string{
					"job": job.ID,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						Name:       jobObj.Name,
						UID:        jobObj.UID,
						Kind:       "Job",
						APIVersion: "batch/v1",
					},
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"job": job.ID,
				},
				Ports: []corev1.ServicePort{
					{
						Name: "management",
						Port: int32(job.ManagementPort),
					},
				},
			},
		}

		if _, err := m.client.CoreV1().Services(job.Namespace).Create(context.Background(), svc, metav1.CreateOptions{}); err != nil {
			return err
		}
	}
	return nil
}

// awaitJobRunning blocks until the test job creates a pod in the RUNNING state
func (m *Manager[C]) awaitJobRunning(job Job[C]) error {
	for {
		pod, err := m.getPod(job, func(pod corev1.Pod) bool {
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
func (m *Manager[C]) awaitJobReady(job Job[C]) error {
	for {
		pod, err := m.getPod(job, func(pod corev1.Pod) bool {
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
func (m *Manager[C]) copyBinary(job Job[C]) error {
	if job.Executable == "" {
		return nil
	}

	pod, err := m.getPod(job, func(pod corev1.Pod) bool {
		return true
	})
	if err != nil {
		return err
	}

	copy := files.CopyOptions{
		Namespace: job.Namespace,
		From:      job.Executable,
		To:        path.Base(job.Executable),
		Pod:       pod.Name,
		Container: "job",
	}
	err = copy.Do(context.Background())
	if err != nil {
		return err
	}
	return nil
}

// runBinary runs the job binary
func (m *Manager[C]) runBinary(job Job[C]) error {
	if job.Executable == "" {
		return nil
	}

	pod, err := m.getPod(job, func(pod corev1.Pod) bool {
		return true
	})
	if err != nil {
		return err
	}
	echo := files.EchoOptions{
		Namespace: job.Namespace,
		Pod:       pod.Name,
		Container: "job",
		File:      "/tmp/bin-ready",
		Bytes:     []byte(path.Base(job.Executable)),
	}
	if err := echo.Do(context.Background()); err != nil {
		return err
	}
	return nil
}

func (m *Manager[C]) copyValuesFile(job Job[C], path string) error {
	pod, err := m.getPod(job, func(pod corev1.Pod) bool {
		return true
	})
	if err != nil {
		return err
	}

	copy := files.CopyOptions{
		Namespace: job.Namespace,
		From:      path,
		To:        path,
		Pod:       pod.Name,
		Container: "job",
	}
	if err := copy.Do(context.Background()); err != nil {
		return err
	}
	return nil
}

// copyContext copies the job context to the pod
func (m *Manager[C]) copyContext(job Job[C]) error {
	if job.Context == "" {
		return nil
	}

	pod, err := m.getPod(job, func(pod corev1.Pod) bool {
		return true
	})
	if err != nil {
		return err
	}

	copy := files.CopyOptions{
		Namespace: job.Namespace,
		From:      job.Context,
		To:        job.Context,
		Pod:       pod.Name,
		Container: "job",
	}
	err = copy.Do(context.Background())
	if err != nil {
		return err
	}
	return nil
}

// createSecrets copies over the CLI secrets into the pod
func (m *Manager[C]) createSecrets(job Job[C]) error {
	jobObj, err := m.client.BatchV1().Jobs(job.Namespace).Get(context.Background(), job.ID, metav1.GetOptions{})
	if err != nil {
		return err
	}
	secretData := make(map[string][]byte)

	for k, v := range job.Secrets {
		secretData[k] = []byte(v)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      helmitSecretsName,
			Namespace: job.Namespace,
			Labels: map[string]string{
				"job": job.ID,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       jobObj.Name,
					UID:        jobObj.UID,
					Kind:       "Job",
					APIVersion: "batch/v1",
				},
			},
		},
		Data: secretData,
	}
	_, _ = m.client.CoreV1().Secrets(job.Namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	return nil
}

// runJob runs the job
func (m *Manager[C]) runJob(job Job[C]) error {
	pod, err := m.getPod(job, func(pod corev1.Pod) bool {
		return true
	})
	if err != nil {
		return err
	}
	echo := files.EchoOptions{
		Namespace: job.Namespace,
		Pod:       pod.Name,
		Container: "job",
		File:      readyFile,
		Bytes:     []byte(path.Base(job.Context)),
	}
	if err := echo.Do(context.Background()); err != nil {
		return err
	}
	return nil
}

// getStatus gets the status message and exit code of the given pod
func (m *Manager[C]) getStatus(job Job[C]) (string, int, error) {
	for {
		pod, err := m.getPod(job, func(pod corev1.Pod) bool {
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
func (m *Manager[C]) getPod(job Job[C], predicate func(pod corev1.Pod) bool) (*corev1.Pod, error) {
	pods, err := m.client.CoreV1().Pods(job.Namespace).List(context.Background(), metav1.ListOptions{
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

// stopJob stops a job
func (m *Manager[C]) finishJob(job Job[C]) error {
	if err := m.deleteJob(job); err != nil {
		return err
	}
	if job.CreateNamespace && !job.NoTeardown {
		if err := m.deleteNamespace(job); err != nil {
			return err
		}
	}
	return nil
}

// deleteJob deletes a job
func (m *Manager[C]) deleteJob(job Job[C]) error {
	deleteOptions := metav1.DeleteOptions{}
	deletePropagation := metav1.DeletePropagationBackground

	deleteOptions.PropagationPolicy = &deletePropagation

	err := m.client.BatchV1().Jobs(job.Namespace).Delete(context.Background(), job.ID, deleteOptions)
	stat, ok := status.FromError(err)
	if err != nil && !k8serrors.IsNotFound(err) && ok && stat.Code() != codes.Unavailable {
		return err
	}
	return nil
}

// deleteNamespace deletes a job
func (m *Manager[C]) deleteNamespace(job Job[C]) error {
	deleteOptions := metav1.DeleteOptions{}
	deletePropagation := metav1.DeletePropagationBackground

	deleteOptions.PropagationPolicy = &deletePropagation

	err := m.client.CoreV1().Namespaces().Delete(context.Background(), job.Namespace, deleteOptions)
	stat, ok := status.FromError(err)
	if err != nil && !k8serrors.IsNotFound(err) && ok && stat.Code() != codes.Unavailable {
		return err
	}
	return nil
}
