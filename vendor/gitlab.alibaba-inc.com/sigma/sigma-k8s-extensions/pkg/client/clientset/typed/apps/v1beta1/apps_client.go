package v1beta1

import (
	v1beta1 "gitlab.alibaba-inc.com/sigma/sigma-k8s-extensions/pkg/apis/apps/v1beta1"
	scheme "gitlab.alibaba-inc.com/sigma/sigma-k8s-extensions/pkg/client/clientset/scheme"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	rest "k8s.io/client-go/rest"
)

type AppsV1beta1Interface interface {
	RESTClient() rest.Interface
	InPlaceSetsGetter
	CapacityPreviewsGetter
}

// AppsV1beta1Client is used to interact with features provided by the apps.sigma.ali group.
type AppsV1beta1Client struct {
	restClient rest.Interface
}

func (c *AppsV1beta1Client) CapacityPreviews(namespace string) CapacityPreviewInterface {
	return newCapacityPreviews(c, namespace)
}

func (c *AppsV1beta1Client) InPlaceSets(namespace string) InPlaceSetInterface {
	return newInPlaceSets(c, namespace)
}

// NewForConfig creates a new AppsV1beta1Client for the given config.
func NewForConfig(c *rest.Config) (*AppsV1beta1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &AppsV1beta1Client{client}, nil
}

// NewForConfigOrDie creates a new AppsV1beta1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *AppsV1beta1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new AppsV1beta1Client for the given RESTClient.
func New(c rest.Interface) *AppsV1beta1Client {
	return &AppsV1beta1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1beta1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *AppsV1beta1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
