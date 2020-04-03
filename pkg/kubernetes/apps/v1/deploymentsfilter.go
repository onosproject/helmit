package v1

import (
	"github.com/onosproject/helmit/pkg/kubernetes/resource"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func NewDeploymentFilter(client resource.Client, parent resource.Filter) resource.Filter {
	return func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
		list := &appsv1.DeploymentList{}
		err := client.Clientset().
			AppsV1().
			RESTClient().
			Get().
			NamespaceIfScoped(client.Namespace(), DeploymentKind.Scoped).
			Resource(DeploymentResource.Name).
			VersionedParams(&metav1.ListOptions{}, metav1.ParameterCodec).
			Timeout(time.Minute).
			Do().
			Into(list)
		if err != nil {
			return false, err
		}
		for _, deployment := range list.Items {
			groupVersionKind := metav1.GroupVersionKind{
				Group:   DeploymentKind.Group,
				Version: DeploymentKind.Version,
				Kind:    DeploymentKind.Kind,
			}
			ok, err := parent(groupVersionKind, deployment.ObjectMeta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		return false, nil
	}
}
