// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package job

import (
	"context"
	"encoding/json"
	"github.com/onosproject/helmit/internal/logging"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Create creates the job resources
func (j *Job[T]) Create(ctx context.Context, log logging.Logger) error {
	if err := j.init(); err != nil {
		return err
	}

	if j.CreateNamespace {
		if err := j.createNamespace(ctx, log); err != nil {
			return err
		}
	}
	if err := j.createClusterRoleBinding(ctx, log); err != nil {
		return err
	}
	if err := j.createJob(ctx, log); err != nil {
		return err
	}
	if err := j.createConfigMap(ctx, log); err != nil {
		return err
	}
	if err := j.createServiceAccount(ctx, log); err != nil {
		return err
	}
	if err := j.createSecrets(ctx, log); err != nil {
		return err
	}
	if err := j.waitForRunning(ctx, log); err != nil {
		return err
	}
	if err := j.copyExecutable(ctx, log); err != nil {
		return err
	}
	if err := j.copyContext(ctx, log); err != nil {
		return err
	}
	if err := j.copyValueFiles(ctx, log); err != nil {
		return err
	}
	if err := j.runExecutable(ctx, log); err != nil {
		return err
	}
	return nil
}

func (j *Job[T]) createNamespace(ctx context.Context, log logging.Logger) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: j.Namespace,
			Annotations: map[string]string{
				"job": j.ID,
			},
		},
	}
	log.Logf("Creating Namespace %s", namespace.Name)
	if _, err := j.client.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

// createJob creates the job to run tests
func (j *Job[T]) createJob(ctx context.Context, log logging.Logger) error {
	env := make([]corev1.EnvVar, 0, len(j.Env))
	for key, value := range j.Env {
		env = append(env, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}
	env = append(env, corev1.EnvVar{
		Name:  "SERVICE_NAMESPACE",
		Value: j.Namespace,
	})
	env = append(env, corev1.EnvVar{
		Name:  "SERVICE_NAME",
		Value: j.ID,
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

	volumes := []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: j.ID,
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

	if j.Secrets != nil && len(j.Secrets) > 0 {
		volumes = append(volumes, corev1.Volume{
			Name: "secrets",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: j.ID,
				},
			},
		})

		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "secrets",
			MountPath: secretsPath,
			ReadOnly:  true,
		})
	}

	var containerPorts []corev1.ContainerPort
	readinessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"stat",
					readyFile,
				},
			},
		},
		PeriodSeconds:    1,
		FailureThreshold: 30,
	}

	serviceAccount := j.ServiceAccount
	if serviceAccount == "" {
		serviceAccount = j.ID
	}

	labels := j.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["job"] = j.ID

	annotations := j.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	zero := int32(0)
	one := int32(1)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        j.ID,
			Namespace:   j.Namespace,
			Labels:      labels,
			Annotations: annotations,
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
							Image:           j.Image,
							ImagePullPolicy: j.ImagePullPolicy,
							Args:            j.Args,
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

	log.Logf("Creating Job %s", job.Name)
	_, err := j.client.BatchV1().Jobs(j.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// createServiceAccount creates a ServiceAccount used by the test manager
func (j *Job[T]) createServiceAccount(ctx context.Context, log logging.Logger) error {
	jobObj, err := j.client.BatchV1().Jobs(j.Namespace).Get(ctx, j.ID, metav1.GetOptions{})
	if err != nil {
		return err
	}

	serviceAccountName := j.ServiceAccount
	if serviceAccountName == "" {
		serviceAccountName = j.ID
	}

	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: j.Namespace,
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
	log.Logf("Creating ServiceAccount %s", serviceAccount.Name)
	_, err = j.client.CoreV1().ServiceAccounts(j.Namespace).Create(ctx, serviceAccount, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// createClusterRoleBinding creates the ClusterRoleBinding required by the test manager
func (j *Job[T]) createClusterRoleBinding(ctx context.Context, log logging.Logger) error {
	serviceAccountName := j.ServiceAccount
	if serviceAccountName == "" {
		serviceAccountName = j.ID
	}
	roleBinding, err := j.client.RbacV1().ClusterRoleBindings().Get(ctx, defaultRoleBindingName, metav1.GetOptions{})
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
					Namespace: j.Namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				Kind:     "ClusterRole",
				Name:     defaultRoleName,
				APIGroup: "rbac.authorization.k8s.io",
			},
		}
		log.Logf("Creating ClusterRoleBinding %s", roleBinding.Name)
		_, err = j.client.RbacV1().ClusterRoleBindings().Create(ctx, roleBinding, metav1.CreateOptions{})
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			return err
		}
		return nil
	}

	roleBinding.Subjects = append(roleBinding.Subjects, rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      serviceAccountName,
		Namespace: j.Namespace,
	})
	log.Logf("Updating ClusterRoleBinding %s", roleBinding.Name)
	_, err = j.client.RbacV1().ClusterRoleBindings().Update(ctx, roleBinding, metav1.UpdateOptions{})
	if err != nil && k8serrors.IsConflict(err) {
		return j.createClusterRoleBinding(ctx, log)
	}
	return err
}

func (j *Job[T]) createConfigMap(ctx context.Context, log logging.Logger) error {
	configJSON, err := json.Marshal(j.Config)
	if err != nil {
		return err
	}

	jobObj, err := j.client.BatchV1().Jobs(j.Namespace).Get(ctx, j.ID, metav1.GetOptions{})
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      j.ID,
			Namespace: j.Namespace,
			Annotations: map[string]string{
				"job": j.ID,
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
			configFile: string(configJSON),
		},
	}

	log.Logf("Creating ConfigMap %s", cm.Name)
	if _, err := j.client.CoreV1().ConfigMaps(j.Namespace).Create(ctx, cm, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

// createSecrets copies over the CLI secrets into the pod
func (j *Job[T]) createSecrets(ctx context.Context, log logging.Logger) error {
	if len(j.Secrets) == 0 {
		return nil
	}

	jobObj, err := j.client.BatchV1().Jobs(j.Namespace).Get(ctx, j.ID, metav1.GetOptions{})
	if err != nil {
		return err
	}
	secretData := make(map[string][]byte)

	for k, v := range j.Secrets {
		secretData[k] = []byte(v)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      j.ID,
			Namespace: j.Namespace,
			Labels: map[string]string{
				"job": j.ID,
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
	log.Logf("Creating Secret %s", secret.Name)
	if _, err := j.client.CoreV1().Secrets(j.Namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}
