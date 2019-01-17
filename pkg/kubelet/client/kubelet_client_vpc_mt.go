package client

import (
	"fmt"
	"net"
	"strconv"
	"net/http"

	"k8s.io/api/core/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/apimachinery/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
)

const (
	AnnotationNodeReverseAddress = "aks.cafe.sofastack.io/reverse-address"
	// Legacy reverse address annotation for backward compatibility
	LegacyAnnotationNodeReverseAddress = "node.cloud.alipay.com/reverse-address"
)

// ParseHostPort parses a network address of the form "host:port", "ipv4:port", "[ipv6]:port" into host and port;
// If the string is not a valid representation of network address, ParseHostPort returns an error.
func parseHostPort(hostport string) (string, string, error) {
	var host, port string
	var err error

	// try to split host and port
	if host, port, err = net.SplitHostPort(hostport); err != nil {
		return "", "", fmt.Errorf("hostport must be a valid representation of network address")
	}

	// if port is defined, parse and validate it
	if _, err = parsePort(port); err != nil {
		return "", "", fmt.Errorf("port must be a valid number between 1 and 65535, inclusive")
	}

	// if host is a valid IP, returns it
	if ip := net.ParseIP(host); ip != nil {
		return host, port, nil
	}

	return "", "", fmt.Errorf("host must be a valid IP address")
}

// ParsePort parses a string representing a TCP port.
// If the string is not a valid representation of a TCP port, ParsePort returns an error.
func parsePort(port string) (int, error) {
	if portInt, err := strconv.Atoi(port); err == nil && (1 <= portInt && portInt <= 65535) {
		return portInt, nil
	}

	return 0, fmt.Errorf("port must be a valid number between 1 and 65535, inclusive")
}

func NewVPCNodeConnectionInfoGetter(config KubeletClientConfig, nodeClient corev1client.NodeInterface, delegate ConnectionInfoGetter) (ConnectionInfoGetter, error) {
	scheme := "http"
	if config.EnableHttps {
		scheme = "https"
	}

	transport, err := MakeTransport(&config)
	if err != nil {
		return nil, err
	}

	return &vpcNodeConnectionInfoGetter{
		scheme:       scheme,
		roundTripper: transport,
		nodeClient:   nodeClient,
		delegate:     delegate,
	}, nil
}

type vpcNodeConnectionInfoGetter struct {
	delegate     ConnectionInfoGetter
	scheme       string
	roundTripper http.RoundTripper
	nodeClient   corev1client.NodeInterface
}

func (k *vpcNodeConnectionInfoGetter) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *k
	copied.nodeClient = k.nodeClient.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(corev1client.NodeInterface)
	return &copied
}

func (k *vpcNodeConnectionInfoGetter) GetConnectionInfo(nodeName types.NodeName) (*ConnectionInfo, error) {
	node, err := k.nodeClient.Get(string(nodeName), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	nodeReverseAddress, ok := getNodeReverseAddress(node)
	if !ok {
		return k.delegate.GetNodeConnectionInfo(node)
	}
	host, port, err := parseHostPort(nodeReverseAddress)
	if err != nil {
		return nil, err
	}

	return &ConnectionInfo{
		Scheme:    k.scheme,
		Hostname:  host,
		Port:      port,
		Transport: k.roundTripper,
	}, nil
}

func (k *vpcNodeConnectionInfoGetter) GetNodeConnectionInfo(node *v1.Node) (*ConnectionInfo, error) {
	nodeReverseAddress, ok := node.Annotations[AnnotationNodeReverseAddress]
	if !ok {
		return k.delegate.GetNodeConnectionInfo(node)
	}
	host, port, err := parseHostPort(nodeReverseAddress)
	if err != nil {
		return nil, err
	}

	return &ConnectionInfo{
		Scheme:    k.scheme,
		Hostname:  host,
		Port:      port,
		Transport: k.roundTripper,
	}, nil
}

func getNodeReverseAddress(node *v1.Node) (string, bool) {
	nodeReverseAddress, ok := node.Annotations[AnnotationNodeReverseAddress]
	if !ok {
		// fallback to legacy reverse address
		nodeReverseAddress, ok = node.Annotations[LegacyAnnotationNodeReverseAddress]
	}
	return nodeReverseAddress, ok
}
