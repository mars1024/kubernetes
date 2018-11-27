package swarm

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"time"
)

var defaultTimeout = time.Second * 10

// Option defines a type used to update http.Request.
type Option func(*http.Request) error

// WithForm sets the form of http.Request.
func WithForm(key string, value string) Option {
	return func(r *http.Request) error {
		if len(r.Form) == 0 {
			r.Form = make(url.Values, 0)
		}
		r.Form.Add(key, value)
		return nil
	}
}

// WithContext sets the ctx of http.Request.
func WithContext(ctx context.Context) Option {
	return func(r *http.Request) error {
		r2 := r.WithContext(ctx)
		*r = *r2
		return nil
	}
}

// WithHeader sets the Header of http.Request.
func WithHeader(key string, value string) Option {
	return func(r *http.Request) error {
		r.Header.Add(key, value)
		return nil
	}
}

// WithQuery sets the query field in URL.
func WithQuery(query url.Values) Option {
	return func(r *http.Request) error {
		r.URL.RawQuery = query.Encode()
		return nil
	}
}

// WithRawData sets the input data with raw data
func WithRawData(data io.ReadCloser) Option {
	return func(r *http.Request) error {
		r.Body = data
		return nil
	}
}

// WithJSONBody encodes the input data to JSON and sets it to the body in http.Request
func WithJSONBody(obj interface{}) Option {
	return func(r *http.Request) error {
		b := bytes.NewBuffer([]byte{})

		if obj != nil {
			err := json.NewEncoder(b).Encode(obj)

			if err != nil {
				return err
			}
		}
		r.Body = ioutil.NopCloser(b)
		r.Header.Set("Content-Type", "application/json")
		return nil
	}
}

// DecodeBody decodes body to obj.
func DecodeBody(obj interface{}, body io.ReadCloser) error {
	defer body.Close()
	return json.NewDecoder(body).Decode(obj)
}

// CreateHttpsConfig create TLSConf.
func CreateHttpsConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	// Load client cert
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		fmt.Errorf("can not load X509 Key Pair", err.Error())
		return nil, err
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		fmt.Errorf("can not read ca cert from file", err.Error())
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS client
	isecureSkipVerify := true
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: isecureSkipVerify,
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig, nil
}

// httpsDo sends http request.
func httpsDo(req *http.Request) (*http.Response, error) {
	// get tlsconfig
	sslDir := TLSDir
	tlsConfig, _ := CreateHttpsConfig(
		path.Join(sslDir, "cert.pem"),
		path.Join(sslDir, "key.pem"),
		path.Join(sslDir, "ca.pem"))

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(30) * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 500,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 30 * time.Second,
	}

	var client = &http.Client{
		// Note: the timeout CAN NOT be too small.
		Timeout:   time.Duration(600) * time.Second,
		Transport: transport,
	}

	resp, err := client.Do(req)
	return resp, err
}

// newRequest creates request targeting on specific host/path by method.
func newRequest(method, url string, opts ...Option) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		err := opt(req)
		if err != nil {
			return nil, err
		}
	}
	return req, nil
}
