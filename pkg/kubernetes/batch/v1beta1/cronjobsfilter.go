package v1beta1

import (
	"github.com/onosproject/helmit/pkg/kubernetes/resource"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func NewCronJobFilter(client resource.Client, parent resource.Filter) resource.Filter {
	return func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
		list := &batchv1beta1.CronJobList{}
		err := client.Clientset().
			BatchV1beta1().
			RESTClient().
			Get().
			NamespaceIfScoped(client.Namespace(), CronJobKind.Scoped).
			Resource(CronJobResource.Name).
			VersionedParams(&metav1.ListOptions{}, metav1.ParameterCodec).
			Timeout(time.Minute).
			Do().
			Into(list)
		if err != nil {
			return false, err
		}
		for _, cronJob := range list.Items {
			groupVersionKind := metav1.GroupVersionKind{
				Group:   CronJobKind.Group,
				Version: CronJobKind.Version,
				Kind:    CronJobKind.Kind,
			}
			ok, err := parent(groupVersionKind, cronJob.ObjectMeta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		return false, nil
	}
}
