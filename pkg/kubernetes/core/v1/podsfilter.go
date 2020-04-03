package v1

import (
	"github.com/onosproject/helmit/pkg/kubernetes/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func NewPodFilter(client resource.Client, parent resource.Filter) resource.Filter {
	return func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
		list := &corev1.PodList{}
		err := client.Clientset().
			CoreV1().
			RESTClient().
			Get().
			NamespaceIfScoped(client.Namespace(), PodKind.Scoped).
			Resource(PodResource.Name).
			VersionedParams(&metav1.ListOptions{}, metav1.ParameterCodec).
			Timeout(time.Minute).
			Do().
			Into(list)
		if err != nil {
			return false, err
		}
		for _, pod := range list.Items {
			groupVersionKind := metav1.GroupVersionKind{
				Group:   PodKind.Group,
				Version: PodKind.Version,
				Kind:    PodKind.Kind,
			}
			ok, err := parent(groupVersionKind, pod.ObjectMeta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		return false, nil
	}
}
