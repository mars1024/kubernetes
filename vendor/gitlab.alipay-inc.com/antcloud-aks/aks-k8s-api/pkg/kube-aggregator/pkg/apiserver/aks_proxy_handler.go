package apiserver

import (
	"net"
	"context"
	"net/http"
	"net/url"
	"time"
	"net/http/httputil"

	restclient "k8s.io/client-go/rest"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/transport"
	"strconv"
)

func newLocalProxyHandler(ip string, port int32, protocol string) *aksProxyHandler {
	proxyRoundTripper, _ := restclient.TransportFor(&restclient.Config{
		TLSClientConfig: restclient.TLSClientConfig{
			Insecure:   true,
			ServerName: ip,
		},
	})

	return &aksProxyHandler{
		ip:                ip,
		port:              port,
		protocol:          protocol,
		proxyRoundTripper: proxyRoundTripper,
	}
}

type aksProxyHandler struct {
	ip                string
	port              int32
	protocol          string
	proxyRoundTripper http.RoundTripper
}

func (r *aksProxyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	user, _ := genericapirequest.UserFrom(req.Context())

	authProxyRoundTripper := transport.NewAuthProxyRoundTripper(user.GetName(), user.GetGroups(), user.GetExtra(), r.proxyRoundTripper)

	localURL := &url.URL{
		Scheme: r.protocol,
		Host:   net.JoinHostPort(r.ip, strconv.Itoa(int(r.port))),
	}
	// WithContext creates a shallow clone of the request with the new context.
	newReq := req.WithContext(context.Background())
	newReq.Header = utilnet.CloneHeader(req.Header)
	newReq.URL = localURL

	proxy := httputil.NewSingleHostReverseProxy(localURL)
	proxy.Transport = authProxyRoundTripper
	proxy.FlushInterval = 200 * time.Millisecond // default flush internal
	proxy.ServeHTTP(w, req)
}
