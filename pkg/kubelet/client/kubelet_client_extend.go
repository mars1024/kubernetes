package client

import (
	"fmt"
	"net"
	"strconv"
)

func (k *NodeConnectionInfoGetter) GetConnectionInfoForVPC(nodeReverseAddress string) (*ConnectionInfo, error) {
	host, port, err := parseHostPort(nodeReverseAddress)
	if err != nil {
		return nil, err
	}

	return &ConnectionInfo{
		Scheme:    k.scheme,
		Hostname:  host,
		Port:      port,
		Transport: k.transport,
	}, nil
}

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
