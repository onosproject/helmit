package v1beta1

import (
	"github.com/onosproject/helmit/pkg/kubernetes/resource"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func NewIngressFilter(client resource.Client, parent resource.Filter) resource.Filter {
	return func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
		list := &extensionsv1beta1.IngressList{}
		err := client.Clientset().
			ExtensionsV1beta1().
			RESTClient().
			Get().
			NamespaceIfScoped(client.Namespace(), IngressKind.Scoped).
			Resource(IngressResource.Name).
			VersionedParams(&metav1.ListOptions{}, metav1.ParameterCodec).
			Timeout(time.Minute).
			Do().
			Into(list)
		if err != nil {
			return false, err
		}
		for _, ingress := range list.Items {
			groupVersionKind := metav1.GroupVersionKind{
				Group:   IngressKind.Group,
				Version: IngressKind.Version,
				Kind:    IngressKind.Kind,
			}
			ok, err := parent(groupVersionKind, ingress.ObjectMeta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		return false, nil
	}
}
