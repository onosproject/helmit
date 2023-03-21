package job

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (j *Job) Delete(ctx context.Context) error {
	if err := j.init(); err != nil {
		return err
	}

	if err := j.deleteJob(ctx); err != nil {
		return err
	}
	if err := j.deleteConfigMap(ctx); err != nil {
		return err
	}
	if j.DeleteNamespace {
		if err := j.deleteNamespace(ctx); err != nil {
			return err
		}
	}
	return nil
}

// deleteConfigMap deletes the job ConfigMap
func (j *Job) deleteConfigMap(ctx context.Context) error {
	err := j.client.CoreV1().ConfigMaps(j.Namespace).Delete(ctx, j.ID, getDeleteOptions())
	stat, ok := status.FromError(err)
	if err != nil && !k8serrors.IsNotFound(err) && ok && stat.Code() != codes.Unavailable {
		return err
	}
	return nil
}

// deleteJob deletes a job
func (j *Job) deleteJob(ctx context.Context) error {
	err := j.client.BatchV1().Jobs(j.Namespace).Delete(ctx, j.ID, getDeleteOptions())
	stat, ok := status.FromError(err)
	if err != nil && !k8serrors.IsNotFound(err) && ok && stat.Code() != codes.Unavailable {
		return err
	}
	return nil
}

// deleteNamespace deletes a job
func (j *Job) deleteNamespace(ctx context.Context) error {
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
