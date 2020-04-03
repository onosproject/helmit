package v1

import (
	"github.com/onosproject/helmit/pkg/kubernetes/resource"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func NewClusterRoleBindingFilter(client resource.Client, parent resource.Filter) resource.Filter {
	return func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
		list := &rbacv1.ClusterRoleBindingList{}
		err := client.Clientset().
			RbacV1().
			RESTClient().
			Get().
			NamespaceIfScoped(client.Namespace(), ClusterRoleBindingKind.Scoped).
			Resource(ClusterRoleBindingResource.Name).
			VersionedParams(&metav1.ListOptions{}, metav1.ParameterCodec).
			Timeout(time.Minute).
			Do().
			Into(list)
		if err != nil {
			return false, err
		}
		for _, clusterRoleBinding := range list.Items {
			groupVersionKind := metav1.GroupVersionKind{
				Group:   ClusterRoleBindingKind.Group,
				Version: ClusterRoleBindingKind.Version,
				Kind:    ClusterRoleBindingKind.Kind,
			}
			ok, err := parent(groupVersionKind, clusterRoleBinding.ObjectMeta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		return false, nil
	}
}
