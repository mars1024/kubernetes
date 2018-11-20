/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testing

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// SetupPrometheusServer setups fake prometheus server
func SetupPrometheusServer(t *testing.T, address string, handler http.Handler) (net.Listener, func()) {
	u, err := url.Parse(address)
	if err != nil {
		t.Fatal("failed to url.Parse with", address, "err:", err)
	}
	var listener net.Listener
	switch u.Scheme {
	case "unix":
		var addr *net.UnixAddr
		addr, err = net.ResolveUnixAddr("unix", u.Path)
		if err != nil {
			t.Fatal("failed to net.ResolveUnixAddr:", address, "err:", err)
		}
		listener, err = net.ListenUnix("unix", addr)

	case "tcp4":
		var addr *net.TCPAddr
		addr, err = net.ResolveTCPAddr("tcp4", u.Path)
		if err != nil {
			t.Fatal("failed to net.ResolveTCPAddr:", address, "err:", err)
		}
		listener, err = net.ListenTCP("tcp4", addr)
	default:
		t.Fatal("unsupport scheme")
	}
	if err != nil {
		t.Fatal("failed to listen at:", address, "err:", err)
	}

	mux := &http.ServeMux{}
	mux.Handle("/api/v1/", handler)
	server := &http.Server{
		Handler: mux,
	}
	go server.Serve(listener)

	return listener, func() {
		listener.Close()
		server.Close()
	}
}

// SetupPrometheusServerWithUnixSocket setups fake prometheus server with unix socket
func SetupPrometheusServerWithUnixSocket(t *testing.T, handler http.Handler) (string, func()) {
	fd, err := ioutil.TempFile("", "sketch")
	assert.NoError(t, err)
	filepath := fd.Name()
	fd.Close()
	os.Remove(filepath)

	address := "unix://" + filepath

	_, clean := SetupPrometheusServer(t, address, handler)
	return address, clean
}
