package v1

import (
	"github.com/onosproject/helmit/pkg/kubernetes/resource"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func NewReplicaSetFilter(client resource.Client, parent resource.Filter) resource.Filter {
	return func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
		list := &appsv1.ReplicaSetList{}
		err := client.Clientset().
			AppsV1().
			RESTClient().
			Get().
			NamespaceIfScoped(client.Namespace(), ReplicaSetKind.Scoped).
			Resource(ReplicaSetResource.Name).
			VersionedParams(&metav1.ListOptions{}, metav1.ParameterCodec).
			Timeout(time.Minute).
			Do().
			Into(list)
		if err != nil {
			return false, err
		}
		for _, replicaSet := range list.Items {
			groupVersionKind := metav1.GroupVersionKind{
				Group:   ReplicaSetKind.Group,
				Version: ReplicaSetKind.Version,
				Kind:    ReplicaSetKind.Kind,
			}
			ok, err := parent(groupVersionKind, replicaSet.ObjectMeta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		return false, nil
	}
}
