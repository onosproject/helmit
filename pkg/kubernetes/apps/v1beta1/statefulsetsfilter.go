package v1beta1

import (
	"github.com/onosproject/helmit/pkg/kubernetes/resource"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func NewStatefulSetFilter(client resource.Client, parent resource.Filter) resource.Filter {
	return func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
		list := &appsv1beta1.StatefulSetList{}
		err := client.Clientset().
			AppsV1beta1().
			RESTClient().
			Get().
			NamespaceIfScoped(client.Namespace(), StatefulSetKind.Scoped).
			Resource(StatefulSetResource.Name).
			VersionedParams(&metav1.ListOptions{}, metav1.ParameterCodec).
			Timeout(time.Minute).
			Do().
			Into(list)
		if err != nil {
			return false, err
		}
		for _, statefulSet := range list.Items {
			groupVersionKind := metav1.GroupVersionKind{
				Group:   StatefulSetKind.Group,
				Version: StatefulSetKind.Version,
				Kind:    StatefulSetKind.Kind,
			}
			ok, err := parent(groupVersionKind, statefulSet.ObjectMeta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		return false, nil
	}
}
