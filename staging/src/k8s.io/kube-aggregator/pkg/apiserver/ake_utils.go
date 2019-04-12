package apiserver

import (
	"net/url"
	"strings"
)

const (
	// AKEApiServiceAddrAnn denotes the override apiserver address
	// in the form of (https://)host:port, (https://)host.
	// e.g. https://test.me:8080, test.me:80, test.url
	AKEApiServiceAddrAnn = "apiservice.cafe.sofastack.io/address"
)

func parseAKEApiService(serverAddr string) (string, string, error) {
	if !strings.HasPrefix(serverAddr, "https://") && !strings.HasPrefix(serverAddr, "http://") {
		serverAddr = "https://" + serverAddr
	}
	serverUrl, err := url.Parse(serverAddr)
	if err != nil {
		return "", "", err
	}
	return serverUrl.Host, serverUrl.Hostname(), nil
}
