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
	"k8s.io/apimachinery/pkg/api/errors"
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
		filter:    filterRelease(parentClient, release),
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

func filterRelease(client resource.Client, release *helm.HelmRelease) resource.Filter {
	return func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
		resources, err := release.GetResources()
		if err != nil {
			return false, err
		}
		return filterResources(client, resources, kind, meta)
	}
}

func filterResources(client resource.Client, resources helmkube.ResourceList, kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
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
	return filterOwners(client, resources, kind, meta)
}

func filterOwners(client resource.Client, resources helmkube.ResourceList, kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
	for _, owner := range meta.OwnerReferences {
		ok, err := filterOwner(client, resources, kind, meta, owner)
		if ok {
			return true, nil
		} else if err != nil {
			return false, err
		}
	}
	if isSameKind(kind, corev1.PodKind) {
		instance, ok := meta.Labels["app.kubernetes.io/instance"]
		if ok {
			daemonSetClient := appsv1.NewDaemonSetsReader(client, resource.NoFilter)
			daemonSet, err := daemonSetClient.Get(instance)
			if err != nil && !errors.IsNotFound(err) {
				return false, err
			} else if err == nil {
				groupVersionKind := metav1.GroupVersionKind{
					Group:   appsv1.DaemonSetKind.Group,
					Version: appsv1.DaemonSetKind.Version,
					Kind:    appsv1.DaemonSetKind.Kind,
				}
				return filterResources(client, resources, groupVersionKind, daemonSet.Object.ObjectMeta)
			}
		}
	}
	if isSameKind(kind, appsv1.ReplicaSetKind) {
		instance, ok := meta.Labels["app.kubernetes.io/instance"]
		if ok {
			deploymentClient := appsv1.NewDeploymentsReader(client, resource.NoFilter)
			deployment, err := deploymentClient.Get(instance)
			if err != nil && !errors.IsNotFound(err) {
				return false, err
			} else if err == nil {
				groupVersionKind := metav1.GroupVersionKind{
					Group:   appsv1.DeploymentKind.Group,
					Version: appsv1.DeploymentKind.Version,
					Kind:    appsv1.DeploymentKind.Kind,
				}
				return filterResources(client, resources, groupVersionKind, deployment.Object.ObjectMeta)
			}
		}
	}
	if isSameKind(kind, corev1.PodKind) {
		instance, ok := meta.Labels["app.kubernetes.io/instance"]
		if ok {
			replicaSetClient := appsv1.NewReplicaSetsReader(client, resource.NoFilter)
			replicaSet, err := replicaSetClient.Get(instance)
			if err != nil && !errors.IsNotFound(err) {
				return false, err
			} else if err == nil {
				groupVersionKind := metav1.GroupVersionKind{
					Group:   appsv1.ReplicaSetKind.Group,
					Version: appsv1.ReplicaSetKind.Version,
					Kind:    appsv1.ReplicaSetKind.Kind,
				}
				return filterResources(client, resources, groupVersionKind, replicaSet.Object.ObjectMeta)
			}
		}
	}
	if isSameKind(kind, corev1.PodKind) {
		instance, ok := meta.Labels["app.kubernetes.io/instance"]
		if ok {
			statefulSetClient := appsv1.NewStatefulSetsReader(client, resource.NoFilter)
			statefulSet, err := statefulSetClient.Get(instance)
			if err != nil && !errors.IsNotFound(err) {
				return false, err
			} else if err == nil {
				groupVersionKind := metav1.GroupVersionKind{
					Group:   appsv1.StatefulSetKind.Group,
					Version: appsv1.StatefulSetKind.Version,
					Kind:    appsv1.StatefulSetKind.Kind,
				}
				return filterResources(client, resources, groupVersionKind, statefulSet.Object.ObjectMeta)
			}
		}
	}
	if isSameKind(kind, appsv1.ReplicaSetKind) {
		instance, ok := meta.Labels["app.kubernetes.io/instance"]
		if ok {
			deploymentClient := appsv1beta1.NewDeploymentsReader(client, resource.NoFilter)
			deployment, err := deploymentClient.Get(instance)
			if err != nil && !errors.IsNotFound(err) {
				return false, err
			} else if err == nil {
				groupVersionKind := metav1.GroupVersionKind{
					Group:   appsv1beta1.DeploymentKind.Group,
					Version: appsv1beta1.DeploymentKind.Version,
					Kind:    appsv1beta1.DeploymentKind.Kind,
				}
				return filterResources(client, resources, groupVersionKind, deployment.Object.ObjectMeta)
			}
		}
	}
	if isSameKind(kind, corev1.PodKind) {
		instance, ok := meta.Labels["app.kubernetes.io/instance"]
		if ok {
			deploymentClient := appsv1beta1.NewDeploymentsReader(client, resource.NoFilter)
			deployment, err := deploymentClient.Get(instance)
			if err != nil && !errors.IsNotFound(err) {
				return false, err
			} else if err == nil {
				groupVersionKind := metav1.GroupVersionKind{
					Group:   appsv1beta1.DeploymentKind.Group,
					Version: appsv1beta1.DeploymentKind.Version,
					Kind:    appsv1beta1.DeploymentKind.Kind,
				}
				return filterResources(client, resources, groupVersionKind, deployment.Object.ObjectMeta)
			}
		}
	}
	if isSameKind(kind, appsv1.ReplicaSetKind) {
		instance, ok := meta.Labels["app.kubernetes.io/instance"]
		if ok {
			statefulSetClient := appsv1beta1.NewStatefulSetsReader(client, resource.NoFilter)
			statefulSet, err := statefulSetClient.Get(instance)
			if err != nil && !errors.IsNotFound(err) {
				return false, err
			} else if err == nil {
				groupVersionKind := metav1.GroupVersionKind{
					Group:   appsv1beta1.StatefulSetKind.Group,
					Version: appsv1beta1.StatefulSetKind.Version,
					Kind:    appsv1beta1.StatefulSetKind.Kind,
				}
				return filterResources(client, resources, groupVersionKind, statefulSet.Object.ObjectMeta)
			}
		}
	}
	if isSameKind(kind, corev1.PodKind) {
		instance, ok := meta.Labels["app.kubernetes.io/instance"]
		if ok {
			statefulSetClient := appsv1beta1.NewStatefulSetsReader(client, resource.NoFilter)
			statefulSet, err := statefulSetClient.Get(instance)
			if err != nil && !errors.IsNotFound(err) {
				return false, err
			} else if err == nil {
				groupVersionKind := metav1.GroupVersionKind{
					Group:   appsv1beta1.StatefulSetKind.Group,
					Version: appsv1beta1.StatefulSetKind.Version,
					Kind:    appsv1beta1.StatefulSetKind.Kind,
				}
				return filterResources(client, resources, groupVersionKind, statefulSet.Object.ObjectMeta)
			}
		}
	}
	if isSameKind(kind, corev1.EndpointsKind) {
		instance, ok := meta.Labels["app.kubernetes.io/instance"]
		if ok {
			serviceClient := corev1.NewServicesReader(client, resource.NoFilter)
			service, err := serviceClient.Get(instance)
			if err != nil && !errors.IsNotFound(err) {
				return false, err
			} else if err == nil {
				groupVersionKind := metav1.GroupVersionKind{
					Group:   corev1.ServiceKind.Group,
					Version: corev1.ServiceKind.Version,
					Kind:    corev1.ServiceKind.Kind,
				}
				return filterResources(client, resources, groupVersionKind, service.Object.ObjectMeta)
			}
		}
	}
	return false, nil
}

func filterOwner(client resource.Client, resources helmkube.ResourceList, kind metav1.GroupVersionKind, meta metav1.ObjectMeta, owner metav1.OwnerReference) (bool, error) {
	for _, resource := range resources {
		resourceKind := resource.Object.GetObjectKind().GroupVersionKind()
		if resourceKind.Kind == owner.Kind &&
			resourceKind.GroupVersion().String() == owner.APIVersion &&
			resource.Name == owner.Name {
			return true, nil
		}
	}
	return false, nil
}

func isSameKind(groupVersionKind metav1.GroupVersionKind, kind resource.Kind) bool {
	return groupVersionKind.Group == kind.Group &&
		groupVersionKind.Version == kind.Version &&
		groupVersionKind.Kind == kind.Kind
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
