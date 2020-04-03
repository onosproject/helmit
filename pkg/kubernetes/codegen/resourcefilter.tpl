package {{ .Reader.Package.Name }}

import (
    "github.com/onosproject/helmit/pkg/kubernetes/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	{{ .Resource.Kind.Package.Alias }} {{ .Resource.Kind.Package.Path | quote }}
	"time"
)

func {{ .Filter.Types.Func }}(client resource.Client, parent resource.Filter) resource.Filter {
    return func(kind metav1.GroupVersionKind, meta metav1.ObjectMeta) (bool, error) {
        list := &{{ (printf "%s.%s" .Resource.Kind.Package.Alias .Resource.Kind.ListKind) }}{}
        err := client.Clientset().
            {{ .Group.Names.Proper }}().
            RESTClient().
            Get().
            NamespaceIfScoped(client.Namespace(), {{ .Resource.Types.Kind }}.Scoped).
            Resource({{ .Resource.Types.Resource }}.Name).
            VersionedParams(&metav1.ListOptions{}, metav1.ParameterCodec).
            Timeout(time.Minute).
            Do().
            Into(list)
        if err != nil {
            return false, err
        }

        {{- $singular := (.Resource.Names.Singular | toLowerCamel) }}
        for _, {{ $singular }} := range list.Items {
            groupVersionKind := metav1.GroupVersionKind{
                Group:   {{ .Resource.Types.Kind }}.Group,
                Version: {{ .Resource.Types.Kind }}.Version,
                Kind:    {{ .Resource.Types.Kind }}.Kind,
            }
            ok, err := parent(groupVersionKind, {{ $singular }}.ObjectMeta)
            if ok {
                return true, nil
            } else if err != nil {
                return false, err
            }
        }
        return false, nil
    }
}
