package v1

import (
	"github.com/onosproject/helmit/pkg/kubernetes/resource"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func NewRoleFilter(client resource.Client, parent resource.Filter) resource.Filter {
	return func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
		list := &rbacv1.RoleList{}
		err := client.Clientset().
			RbacV1().
			RESTClient().
			Get().
			NamespaceIfScoped(client.Namespace(), RoleKind.Scoped).
			Resource(RoleResource.Name).
			VersionedParams(&metav1.ListOptions{}, metav1.ParameterCodec).
			Timeout(time.Minute).
			Do().
			Into(list)
		if err != nil {
			return false, err
		}
		for _, role := range list.Items {
			groupVersionKind := metav1.GroupVersionKind{
				Group:   RoleKind.Group,
				Version: RoleKind.Version,
				Kind:    RoleKind.Kind,
			}
			ok, err := parent(groupVersionKind, role.ObjectMeta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		return false, nil
	}
}
