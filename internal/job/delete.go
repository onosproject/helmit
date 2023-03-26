// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package job

import (
	"context"
	"github.com/onosproject/helmit/internal/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Delete delets the job resources
func (j *Job[T]) Delete(ctx context.Context, log logging.Logger) error {
	if err := j.init(); err != nil {
		return err
	}

	if err := j.deleteJob(ctx, log); err != nil {
		return err
	}
	if err := j.deleteConfigMap(ctx, log); err != nil {
		return err
	}
	if j.DeleteNamespace {
		if err := j.deleteNamespace(ctx, log); err != nil {
			return err
		}
	}
	return nil
}

// deleteConfigMap deletes the job ConfigMap
func (j *Job[T]) deleteConfigMap(ctx context.Context, log logging.Logger) error {
	log.Logf("Deleting ConfigMap %s", j.ID)
	err := j.client.CoreV1().ConfigMaps(j.Namespace).Delete(ctx, j.ID, getDeleteOptions())
	stat, ok := status.FromError(err)
	if err != nil && !k8serrors.IsNotFound(err) && ok && stat.Code() != codes.Unavailable {
		return err
	}
	return nil
}

// deleteJob deletes a job
func (j *Job[T]) deleteJob(ctx context.Context, log logging.Logger) error {
	log.Logf("Deleting Job %s", j.ID)
	err := j.client.BatchV1().Jobs(j.Namespace).Delete(ctx, j.ID, getDeleteOptions())
	stat, ok := status.FromError(err)
	if err != nil && !k8serrors.IsNotFound(err) && ok && stat.Code() != codes.Unavailable {
		return err
	}
	return nil
}

// deleteNamespace deletes a job
func (j *Job[T]) deleteNamespace(ctx context.Context, log logging.Logger) error {
	log.Logf("Deleting Namespace %s", j.Namespace)
	err := j.client.CoreV1().Namespaces().Delete(ctx, j.Namespace, getDeleteOptions())
	stat, ok := status.FromError(err)
	if err != nil && !k8serrors.IsNotFound(err) && ok && stat.Code() != codes.Unavailable {
		return err
	}
	return nil
}

func getDeleteOptions() metav1.DeleteOptions {
	deletePropagation := metav1.DeletePropagationBackground
	return metav1.DeleteOptions{
		PropagationPolicy: &deletePropagation,
	}
}
