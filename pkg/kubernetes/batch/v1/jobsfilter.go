package v1

import (
	"github.com/onosproject/helmit/pkg/kubernetes/resource"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func NewJobFilter(client resource.Client, parent resource.Filter) resource.Filter {
	return func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
		list := &batchv1.JobList{}
		err := client.Clientset().
			BatchV1().
			RESTClient().
			Get().
			NamespaceIfScoped(client.Namespace(), JobKind.Scoped).
			Resource(JobResource.Name).
			VersionedParams(&metav1.ListOptions{}, metav1.ParameterCodec).
			Timeout(time.Minute).
			Do().
			Into(list)
		if err != nil {
			return false, err
		}
		for _, job := range list.Items {
			groupVersionKind := metav1.GroupVersionKind{
				Group:   JobKind.Group,
				Version: JobKind.Version,
				Kind:    JobKind.Kind,
			}
			ok, err := parent(groupVersionKind, job.ObjectMeta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		return false, nil
	}
}
