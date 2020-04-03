package v1

import (
	"github.com/onosproject/helmit/pkg/kubernetes/resource"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func NewRoleBindingFilter(client resource.Client, parent resource.Filter) resource.Filter {
	return func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
		list := &rbacv1.RoleBindingList{}
		err := client.Clientset().
			RbacV1().
			RESTClient().
			Get().
			NamespaceIfScoped(client.Namespace(), RoleBindingKind.Scoped).
			Resource(RoleBindingResource.Name).
			VersionedParams(&metav1.ListOptions{}, metav1.ParameterCodec).
			Timeout(time.Minute).
			Do().
			Into(list)
		if err != nil {
			return false, err
		}
		for _, roleBinding := range list.Items {
			groupVersionKind := metav1.GroupVersionKind{
				Group:   RoleBindingKind.Group,
				Version: RoleBindingKind.Version,
				Kind:    RoleBindingKind.Kind,
			}
			ok, err := parent(groupVersionKind, roleBinding.ObjectMeta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		return false, nil
	}
}
