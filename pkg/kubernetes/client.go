package kubernetes

import (
	"github.com/onosproject/helmit/pkg/helm"
	appsv1 "github.com/onosproject/helmit/pkg/kubernetes/apps/v1"
	appsv1beta1 "github.com/onosproject/helmit/pkg/kubernetes/apps/v1beta1"
	batchv1 "github.com/onosproject/helmit/pkg/kubernetes/batch/v1"
	batchv1beta1 "github.com/onosproject/helmit/pkg/kubernetes/batch/v1beta1"
	batchv2alpha1 "github.com/onosproject/helmit/pkg/kubernetes/batch/v2alpha1"
	"github.com/onosproject/helmit/pkg/kubernetes/config"
	corev1 "github.com/onosproject/helmit/pkg/kubernetes/core/v1"
	extensionsv1beta1 "github.com/onosproject/helmit/pkg/kubernetes/extensions/v1beta1"
	networkingv1beta1 "github.com/onosproject/helmit/pkg/kubernetes/networking/v1beta1"
	rbacv1 "github.com/onosproject/helmit/pkg/kubernetes/rbac/v1"
	"github.com/onosproject/helmit/pkg/kubernetes/resource"
	helmkube "helm.sh/helm/v3/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// New returns a new Kubernetes client for the current namespace
func New() (Client, error) {
	return NewForNamespace(config.GetNamespaceFromEnv())
}

// NewOrDie returns a new Kubernetes client for the current namespace
func NewOrDie() Client {
	client, err := New()
	if err != nil {
		panic(err)
	}
	return client
}

// NewForNamespace returns a new Kubernetes client for the given namespace
func NewForNamespace(namespace string) (Client, error) {
	kubernetesConfig, err := config.GetRestConfig()
	if err != nil {
		return nil, err
	}
	kubernetesClient, err := kubernetes.NewForConfig(kubernetesConfig)
	if err != nil {
		return nil, err
	}
	return &client{
		namespace: namespace,
		config:    kubernetesConfig,
		client:    kubernetesClient,
		filter:    resource.NoFilter,
	}, nil
}

// NewForNamespaceOrDie returns a new Kubernetes client for the given namespace
func NewForNamespaceOrDie(namespace string) Client {
	client, err := NewForNamespace(namespace)
	if err != nil {
		panic(err)
	}
	return client
}

// Client is a Kubernetes client
type Client interface {
	// Namespace returns the client namespace
	Namespace() string

	// Config returns the Kubernetes REST client configuration
	Config() *rest.Config

	// Clientset returns the client's Clientset
	Clientset() *kubernetes.Clientset
	AppsV1() appsv1.Client
	AppsV1beta1() appsv1beta1.Client
	BatchV1() batchv1.Client
	BatchV1beta1() batchv1beta1.Client
	BatchV2alpha1() batchv2alpha1.Client
	CoreV1() corev1.Client
	ExtensionsV1beta1() extensionsv1beta1.Client
	NetworkingV1beta1() networkingv1beta1.Client
	RbacV1() rbacv1.Client
}

// NewForRelease returns a new Kubernetes client for the given release
func NewForRelease(release *helm.HelmRelease) (Client, error) {
	kubernetesConfig, err := config.GetRestConfig()
	if err != nil {
		return nil, err
	}
	kubernetesClient, err := kubernetes.NewForConfig(kubernetesConfig)
	if err != nil {
		return nil, err
	}
	parentClient := &client{
		namespace: release.Namespace(),
		config:    kubernetesConfig,
		client:    kubernetesClient,
		filter:    resource.NoFilter,
	}
	return &client{
		namespace: release.Namespace(),
		config:    kubernetesConfig,
		client:    kubernetesClient,
		filter:    newReleaseFilter(parentClient, release),
	}, nil
}

// NewForReleaseOrDie returns a new Kubernetes client for the given release
func NewForReleaseOrDie(release *helm.HelmRelease) Client {
	client, err := NewForRelease(release)
	if err != nil {
		panic(err)
	}
	return client
}

func newReleaseFilter(client resource.Client, release *helm.HelmRelease) resource.Filter {
	return func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
		resources, err := release.GetResources()
		if err != nil {
			return false, err
		}
		for _, resource := range resources {
			resourceKind := resource.Object.GetObjectKind().GroupVersionKind()
			if resourceKind.Group == kind.Group &&
				resourceKind.Version == kind.Version &&
				resourceKind.Kind == kind.Kind &&
				resource.Namespace == meta.Namespace &&
				resource.Name == meta.Name {
				return true, nil
			}
		}
		return filterOwnerReferences(client, resources, kind, meta)
	}
}

func filterOwnerReferences(client resource.Client, resources helmkube.ResourceList, kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
	for _, owner := range meta.OwnerReferences {
		for _, resource := range resources {
			resourceKind := resource.Object.GetObjectKind().GroupVersionKind()
			if resourceKind.Kind == owner.Kind &&
				resourceKind.GroupVersion().String() == owner.APIVersion &&
				resource.Name == owner.Name {
				return true, nil
			}
		}
		if owner.APIVersion == "apps.v1" {
			filter := appsv1.NewDaemonSetFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "apps.v1" {
			filter := appsv1.NewDeploymentFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "apps.v1" {
			filter := appsv1.NewReplicaSetFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "apps.v1" {
			filter := appsv1.NewStatefulSetFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "apps.v1beta1" {
			filter := appsv1beta1.NewDeploymentFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "apps.v1beta1" {
			filter := appsv1beta1.NewStatefulSetFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "batch.v1" {
			filter := batchv1.NewJobFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "batch.v1beta1" {
			filter := batchv1beta1.NewCronJobFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "batch.v2alpha1" {
			filter := batchv2alpha1.NewCronJobFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "core.v1" {
			filter := corev1.NewConfigMapFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "core.v1" {
			filter := corev1.NewEndpointsFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "core.v1" {
			filter := corev1.NewNodeFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "core.v1" {
			filter := corev1.NewPodFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "core.v1" {
			filter := corev1.NewSecretFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "core.v1" {
			filter := corev1.NewServiceFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "extensions.v1beta1" {
			filter := extensionsv1beta1.NewIngressFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "networking.v1beta1" {
			filter := networkingv1beta1.NewIngressFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "rbac.authorization.k8s.io.v1" {
			filter := rbacv1.NewClusterRoleFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "rbac.authorization.k8s.io.v1" {
			filter := rbacv1.NewClusterRoleBindingFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "rbac.authorization.k8s.io.v1" {
			filter := rbacv1.NewRoleFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
		if owner.APIVersion == "rbac.authorization.k8s.io.v1" {
			filter := rbacv1.NewRoleBindingFilter(client, func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
				return filterOwnerReferences(client, resources, kind, meta)
			})
			ok, err := filter(kind, meta)
			if ok {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}
	}
	return false, nil
}

type client struct {
	namespace string
	config    *rest.Config
	client    *kubernetes.Clientset
	filter    resource.Filter
}

func (c *client) Namespace() string {
	return c.namespace
}

func (c *client) Config() *rest.Config {
	return c.config
}

func (c *client) Clientset() *kubernetes.Clientset {
	return c.client
}
func (c *client) AppsV1() appsv1.Client {
	return appsv1.NewClient(c, c.filter)
}

func (c *client) AppsV1beta1() appsv1beta1.Client {
	return appsv1beta1.NewClient(c, c.filter)
}

func (c *client) BatchV1() batchv1.Client {
	return batchv1.NewClient(c, c.filter)
}

func (c *client) BatchV1beta1() batchv1beta1.Client {
	return batchv1beta1.NewClient(c, c.filter)
}

func (c *client) BatchV2alpha1() batchv2alpha1.Client {
	return batchv2alpha1.NewClient(c, c.filter)
}

func (c *client) CoreV1() corev1.Client {
	return corev1.NewClient(c, c.filter)
}

func (c *client) ExtensionsV1beta1() extensionsv1beta1.Client {
	return extensionsv1beta1.NewClient(c, c.filter)
}

func (c *client) NetworkingV1beta1() networkingv1beta1.Client {
	return networkingv1beta1.NewClient(c, c.filter)
}

func (c *client) RbacV1() rbacv1.Client {
	return rbacv1.NewClient(c, c.filter)
}
