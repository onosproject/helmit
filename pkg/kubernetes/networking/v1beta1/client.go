// Code generated by helmit-generate. DO NOT EDIT.

package v1beta1

import (
	"github.com/onosproject/helmit/pkg/kubernetes/resource"
)

type Client interface {
	IngressesClient
}

func NewClient(resources resource.Client, filter resource.Filter) Client {
	return &client{
		Client:          resources,
		IngressesClient: NewIngressesClient(resources, filter),
	}
}

type client struct {
	resource.Client
	IngressesClient
}
