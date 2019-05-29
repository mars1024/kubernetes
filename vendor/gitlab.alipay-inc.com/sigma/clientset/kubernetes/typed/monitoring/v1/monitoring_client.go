/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"gitlab.alipay-inc.com/sigma/clientset/kubernetes/scheme"
	v1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/monitoring/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	rest "k8s.io/client-go/rest"
)

type MonitoringV1Interface interface {
	RESTClient() rest.Interface
	ClusterScrapeConfigsGetter
	MonitoringRulesGetter
	NotificationChannelsGetter
	NotificationGroupsGetter
	NotificationReceiversGetter
	NotificationTemplatesGetter
	ScrapeConfigsGetter
}

// MonitoringV1Client is used to interact with features provided by the monitoring.sigma.alipay.com group.
type MonitoringV1Client struct {
	restClient rest.Interface
}

func (c *MonitoringV1Client) ClusterScrapeConfigs() ClusterScrapeConfigInterface {
	return newClusterScrapeConfigs(c)
}

func (c *MonitoringV1Client) MonitoringRules() MonitoringRuleInterface {
	return newMonitoringRules(c)
}

func (c *MonitoringV1Client) NotificationChannels() NotificationChannelInterface {
	return newNotificationChannels(c)
}

func (c *MonitoringV1Client) NotificationGroups() NotificationGroupInterface {
	return newNotificationGroups(c)
}

func (c *MonitoringV1Client) NotificationReceivers() NotificationReceiverInterface {
	return newNotificationReceivers(c)
}

func (c *MonitoringV1Client) NotificationTemplates() NotificationTemplateInterface {
	return newNotificationTemplates(c)
}

func (c *MonitoringV1Client) ScrapeConfigs(namespace string) ScrapeConfigInterface {
	return newScrapeConfigs(c, namespace)
}

// NewForConfig creates a new MonitoringV1Client for the given config.
func NewForConfig(c *rest.Config) (*MonitoringV1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &MonitoringV1Client{client}, nil
}

// NewForConfigOrDie creates a new MonitoringV1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *MonitoringV1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new MonitoringV1Client for the given RESTClient.
func New(c rest.Interface) *MonitoringV1Client {
	return &MonitoringV1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1.SchemeGroupVersion
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
func (c *MonitoringV1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
